// Copyright 2016 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package android

import (
	"path/filepath"
	"regexp"
	"strings"

	"android/soong/bazel"
	"android/soong/bazel/cquery"

	"github.com/google/blueprint"
)

func init() {
	RegisterModuleType("filegroup", FileGroupFactory)
}

var PrepareForTestWithFilegroup = FixtureRegisterWithContext(func(ctx RegistrationContext) {
	ctx.RegisterModuleType("filegroup", FileGroupFactory)
})

var convertedProtoLibrarySuffix = "_bp2build_converted"

// IsFilegroup checks that a module is a filegroup type
func IsFilegroup(ctx bazel.OtherModuleContext, m blueprint.Module) bool {
	return ctx.OtherModuleType(m) == "filegroup"
}

var (
	// ignoring case, checks for proto or protos as an independent word in the name, whether at the
	// beginning, end, or middle. e.g. "proto.foo", "bar-protos", "baz_proto_srcs" would all match
	filegroupLikelyProtoPattern = regexp.MustCompile("(?i)(^|[^a-z])proto(s)?([^a-z]|$)")
	filegroupLikelyAidlPattern  = regexp.MustCompile("(?i)(^|[^a-z])aidl([^a-z]|$)")

	ProtoSrcLabelPartition = bazel.LabelPartition{
		Extensions:  []string{".proto"},
		LabelMapper: isFilegroupWithPattern(filegroupLikelyProtoPattern),
	}
	AidlSrcLabelPartition = bazel.LabelPartition{
		Extensions:  []string{".aidl"},
		LabelMapper: isFilegroupWithPattern(filegroupLikelyAidlPattern),
	}
)

func isFilegroupWithPattern(pattern *regexp.Regexp) bazel.LabelMapper {
	return func(ctx bazel.OtherModuleContext, label bazel.Label) (string, bool) {
		m, exists := ctx.ModuleFromName(label.OriginalModuleName)
		labelStr := label.Label
		if !exists || !IsFilegroup(ctx, m) {
			return labelStr, false
		}
		likelyMatched := pattern.MatchString(label.OriginalModuleName)
		return labelStr, likelyMatched
	}
}

// https://docs.bazel.build/versions/master/be/general.html#filegroup
type bazelFilegroupAttributes struct {
	Srcs bazel.LabelListAttribute
}

type bazelAidlLibraryAttributes struct {
	Srcs                bazel.LabelListAttribute
	Strip_import_prefix *string
}

// api srcs can be contained in filegroups.
// this should be generated in api_bp2build workspace as well.
func (fg *fileGroup) ConvertWithApiBp2build(ctx TopDownMutatorContext) {
	fg.ConvertWithBp2build(ctx)
}

// ConvertWithBp2build performs bp2build conversion of filegroup
func (fg *fileGroup) ConvertWithBp2build(ctx TopDownMutatorContext) {
	srcs := bazel.MakeLabelListAttribute(
		BazelLabelForModuleSrcExcludes(ctx, fg.properties.Srcs, fg.properties.Exclude_srcs))

	// For Bazel compatibility, don't generate the filegroup if there is only 1
	// source file, and that the source file is named the same as the module
	// itself. In Bazel, eponymous filegroups like this would be an error.
	//
	// Instead, dependents on this single-file filegroup can just depend
	// on the file target, instead of rule target, directly.
	//
	// You may ask: what if a filegroup has multiple files, and one of them
	// shares the name? The answer: we haven't seen that in the wild, and
	// should lock Soong itself down to prevent the behavior. For now,
	// we raise an error if bp2build sees this problem.
	for _, f := range srcs.Value.Includes {
		if f.Label == fg.Name() {
			if len(srcs.Value.Includes) > 1 {
				ctx.ModuleErrorf("filegroup '%s' cannot contain a file with the same name", fg.Name())
			}
			return
		}
	}

	// Convert module that has only AIDL files to aidl_library
	// If the module has a mixed bag of AIDL and non-AIDL files, split the filegroup manually
	// and then convert
	if fg.ShouldConvertToAidlLibrary(ctx) {
		attrs := &bazelAidlLibraryAttributes{
			Srcs:                srcs,
			Strip_import_prefix: fg.properties.Path,
		}

		props := bazel.BazelTargetModuleProperties{
			Rule_class:        "aidl_library",
			Bzl_load_location: "//build/bazel/rules/aidl:library.bzl",
		}

		ctx.CreateBazelTargetModule(props, CommonAttributes{Name: fg.Name()}, attrs)
	} else {
		if fg.ShouldConvertToProtoLibrary(ctx) {
			// TODO(b/246997908): we can remove this tag if we could figure out a
			// solution for this bug.
			attrs := &ProtoAttrs{
				Srcs:                srcs,
				Strip_import_prefix: fg.properties.Path,
			}

			tags := []string{"manual"}
			ctx.CreateBazelTargetModule(
				bazel.BazelTargetModuleProperties{Rule_class: "proto_library"},
				CommonAttributes{
					Name: fg.Name() + convertedProtoLibrarySuffix,
					Tags: bazel.MakeStringListAttribute(tags),
				},
				attrs)
		}

		// TODO(b/242847534): Still convert to a filegroup because other unconverted
		// modules may depend on the filegroup
		attrs := &bazelFilegroupAttributes{
			Srcs: srcs,
		}

		props := bazel.BazelTargetModuleProperties{
			Rule_class:        "filegroup",
			Bzl_load_location: "//build/bazel/rules:filegroup.bzl",
		}

		ctx.CreateBazelTargetModule(props, CommonAttributes{Name: fg.Name()}, attrs)
	}
}

type fileGroupProperties struct {
	// srcs lists files that will be included in this filegroup
	Srcs []string `android:"path"`

	Exclude_srcs []string `android:"path"`

	// The base path to the files.  May be used by other modules to determine which portion
	// of the path to use.  For example, when a filegroup is used as data in a cc_test rule,
	// the base path is stripped off the path and the remaining path is used as the
	// installation directory.
	Path *string

	// Create a make variable with the specified name that contains the list of files in the
	// filegroup, relative to the root of the source tree.
	Export_to_make_var *string
}

type fileGroup struct {
	ModuleBase
	BazelModuleBase
	FileGroupAsLibrary
	properties fileGroupProperties
	srcs       Paths
}

var _ MixedBuildBuildable = (*fileGroup)(nil)
var _ SourceFileProducer = (*fileGroup)(nil)
var _ FileGroupAsLibrary = (*fileGroup)(nil)

// filegroup contains a list of files that are referenced by other modules
// properties (such as "srcs") using the syntax ":<name>". filegroup are
// also be used to export files across package boundaries.
func FileGroupFactory() Module {
	module := &fileGroup{}
	module.AddProperties(&module.properties)
	InitAndroidModule(module)
	InitBazelModule(module)
	return module
}

var _ blueprint.JSONActionSupplier = (*fileGroup)(nil)

func (fg *fileGroup) JSONActions() []blueprint.JSONAction {
	ins := make([]string, 0, len(fg.srcs))
	outs := make([]string, 0, len(fg.srcs))
	for _, p := range fg.srcs {
		ins = append(ins, p.String())
		outs = append(outs, p.Rel())
	}
	return []blueprint.JSONAction{
		blueprint.JSONAction{
			Inputs:  ins,
			Outputs: outs,
		},
	}
}

func (fg *fileGroup) GenerateAndroidBuildActions(ctx ModuleContext) {
	fg.srcs = PathsForModuleSrcExcludes(ctx, fg.properties.Srcs, fg.properties.Exclude_srcs)
	if fg.properties.Path != nil {
		fg.srcs = PathsWithModuleSrcSubDir(ctx, fg.srcs, String(fg.properties.Path))
	}
}

func (fg *fileGroup) Srcs() Paths {
	return append(Paths{}, fg.srcs...)
}

func (fg *fileGroup) MakeVars(ctx MakeVarsModuleContext) {
	if makeVar := String(fg.properties.Export_to_make_var); makeVar != "" {
		ctx.StrictRaw(makeVar, strings.Join(fg.srcs.Strings(), " "))
	}
}

func (fg *fileGroup) QueueBazelCall(ctx BaseModuleContext) {
	bazelCtx := ctx.Config().BazelContext

	bazelCtx.QueueBazelRequest(
		fg.GetBazelLabel(ctx, fg),
		cquery.GetOutputFiles,
		configKey{arch: Common.String(), osType: CommonOS})
}

func (fg *fileGroup) IsMixedBuildSupported(ctx BaseModuleContext) bool {
	// TODO(b/247782695), TODO(b/242847534) Fix mixed builds for filegroups
	return false
}

func (fg *fileGroup) ProcessBazelQueryResponse(ctx ModuleContext) {
	bazelCtx := ctx.Config().BazelContext
	// This is a short-term solution because we rely on info from Android.bp to handle
	// a converted module. This will block when we want to remove Android.bp for all
	// converted modules at some point.
	// TODO(b/242847534): Implement a long-term solution in which we don't need to rely
	// on info form Android.bp for modules that are already converted to Bazel
	relativeRoot := ctx.ModuleDir()
	if fg.properties.Path != nil {
		relativeRoot = filepath.Join(relativeRoot, *fg.properties.Path)
	}

	filePaths, err := bazelCtx.GetOutputFiles(fg.GetBazelLabel(ctx, fg), configKey{arch: Common.String(), osType: CommonOS})
	if err != nil {
		ctx.ModuleErrorf(err.Error())
		return
	}

	bazelOuts := make(Paths, 0, len(filePaths))
	for _, p := range filePaths {
		bazelOuts = append(bazelOuts, PathForBazelOutRelative(ctx, relativeRoot, p))
	}
	fg.srcs = bazelOuts
}

func (fg *fileGroup) ShouldConvertToAidlLibrary(ctx BazelConversionPathContext) bool {
	return fg.shouldConvertToLibrary(ctx, ".aidl")
}

func (fg *fileGroup) ShouldConvertToProtoLibrary(ctx BazelConversionPathContext) bool {
	return fg.shouldConvertToLibrary(ctx, ".proto")
}

func (fg *fileGroup) shouldConvertToLibrary(ctx BazelConversionPathContext, suffix string) bool {
	if len(fg.properties.Srcs) == 0 || !fg.ShouldConvertWithBp2build(ctx) {
		return false
	}
	for _, src := range fg.properties.Srcs {
		if !strings.HasSuffix(src, suffix) {
			return false
		}
	}
	return true
}

func (fg *fileGroup) GetAidlLibraryLabel(ctx BazelConversionPathContext) string {
	return fg.getFileGroupAsLibraryLabel(ctx)
}

func (fg *fileGroup) GetProtoLibraryLabel(ctx BazelConversionPathContext) string {
	return fg.getFileGroupAsLibraryLabel(ctx) + convertedProtoLibrarySuffix
}

func (fg *fileGroup) getFileGroupAsLibraryLabel(ctx BazelConversionPathContext) string {
	if ctx.OtherModuleDir(fg.module) == ctx.ModuleDir() {
		return ":" + fg.Name()
	} else {
		return fg.GetBazelLabel(ctx, fg)
	}
}

// Given a name in srcs prop, check to see if the name references a filegroup
// and the filegroup is converted to aidl_library
func IsConvertedToAidlLibrary(ctx BazelConversionPathContext, name string) bool {
	if fg, ok := ToFileGroupAsLibrary(ctx, name); ok {
		return fg.ShouldConvertToAidlLibrary(ctx)
	}
	return false
}

func ToFileGroupAsLibrary(ctx BazelConversionPathContext, name string) (FileGroupAsLibrary, bool) {
	if module, ok := ctx.ModuleFromName(name); ok {
		if IsFilegroup(ctx, module) {
			if fg, ok := module.(FileGroupAsLibrary); ok {
				return fg, true
			}
		}
	}
	return nil, false
}
