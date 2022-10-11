// Copyright 2017 Google Inc. All rights reserved.
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

package python

// This file contains the module types for building Python binary.

import (
	"fmt"

	"android/soong/android"
	"android/soong/bazel"

	"github.com/google/blueprint/proptools"
)

func init() {
	registerPythonBinaryComponents(android.InitRegistrationContext)
}

func registerPythonBinaryComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("python_binary_host", PythonBinaryHostFactory)
}

type bazelPythonBinaryAttributes struct {
	Main           *string
	Srcs           bazel.LabelListAttribute
	Deps           bazel.LabelListAttribute
	Python_version *string
	Imports        bazel.StringListAttribute
}

func pythonBinaryBp2Build(ctx android.TopDownMutatorContext, m *Module) {
	var main *string
	for _, propIntf := range m.GetProperties() {
		if props, ok := propIntf.(*BinaryProperties); ok {
			// main is optional.
			if props.Main != nil {
				main = props.Main
				break
			}
		}
	}

	// TODO(b/182306917): this doesn't fully handle all nested props versioned
	// by the python version, which would have been handled by the version split
	// mutator. This is sufficient for very simple python_binary_host modules
	// under Bionic.
	py3Enabled := proptools.BoolDefault(m.properties.Version.Py3.Enabled, false)
	py2Enabled := proptools.BoolDefault(m.properties.Version.Py2.Enabled, false)
	var python_version *string
	if py3Enabled && py2Enabled {
		panic(fmt.Errorf(
			"error for '%s' module: bp2build's python_binary_host converter does not support "+
				"converting a module that is enabled for both Python 2 and 3 at the same time.", m.Name()))
	} else if py2Enabled {
		python_version = &pyVersion2
	} else {
		// do nothing, since python_version defaults to PY3.
	}

	baseAttrs := m.makeArchVariantBaseAttributes(ctx)
	attrs := &bazelPythonBinaryAttributes{
		Main:           main,
		Srcs:           baseAttrs.Srcs,
		Deps:           baseAttrs.Deps,
		Python_version: python_version,
		Imports:        baseAttrs.Imports,
	}

	props := bazel.BazelTargetModuleProperties{
		// Use the native py_binary rule.
		Rule_class: "py_binary",
	}

	ctx.CreateBazelTargetModule(props, android.CommonAttributes{
		Name: m.Name(),
		Data: baseAttrs.Data,
	}, attrs)
}

type BinaryProperties struct {
	// the name of the source file that is the main entry point of the program.
	// this file must also be listed in srcs.
	// If left unspecified, module name is used instead.
	// If name doesn’t match any filename in srcs, main must be specified.
	Main *string `android:"arch_variant"`

	// set the name of the output binary.
	Stem *string `android:"arch_variant"`

	// append to the name of the output binary.
	Suffix *string `android:"arch_variant"`

	// list of compatibility suites (for example "cts", "vts") that the module should be
	// installed into.
	Test_suites []string `android:"arch_variant"`

	// whether to use `main` when starting the executable. The default is true, when set to
	// false it will act much like the normal `python` executable, but with the sources and
	// libraries automatically included in the PYTHONPATH.
	Autorun *bool `android:"arch_variant"`

	// Flag to indicate whether or not to create test config automatically. If AndroidTest.xml
	// doesn't exist next to the Android.bp, this attribute doesn't need to be set to true
	// explicitly.
	Auto_gen_config *bool

	// Currently, both the root of the zipfile and all the directories 1 level
	// below that are added to the python path. When this flag is set to true,
	// only the root of the zipfile will be added to the python path. This flag
	// will be removed after all the python modules in the tree have been updated
	// to support it. When using embedded_launcher: true, this is already the
	// behavior. The default is currently false.
	Dont_add_top_level_directories_to_path *bool

	// Setting this to true will mimic Python 3.11+'s PYTHON_SAFE_PATH environment
	// variable or -P flag, even on older python versions. This is a temporary
	// flag while modules are changed to support it, eventually true will be the
	// default and the flag will be removed. The default is currently false. It
	// is only applicable when embedded_launcher is false, when embedded_launcher
	// is true this is already implied.
	Dont_add_entrypoint_folder_to_path *bool
}

type binaryDecorator struct {
	binaryProperties BinaryProperties

	*pythonInstaller
}

type IntermPathProvider interface {
	IntermPathForModuleOut() android.OptionalPath
}

func NewBinary(hod android.HostOrDeviceSupported) (*Module, *binaryDecorator) {
	module := newModule(hod, android.MultilibFirst)
	decorator := &binaryDecorator{pythonInstaller: NewPythonInstaller("bin", "")}

	module.bootstrapper = decorator
	module.installer = decorator

	return module, decorator
}

func PythonBinaryHostFactory() android.Module {
	module, _ := NewBinary(android.HostSupported)

	android.InitBazelModule(module)

	return module.init()
}

func (binary *binaryDecorator) autorun() bool {
	return BoolDefault(binary.binaryProperties.Autorun, true)
}

func (binary *binaryDecorator) bootstrapperProps() []interface{} {
	return []interface{}{&binary.binaryProperties}
}

func (binary *binaryDecorator) bootstrap(ctx android.ModuleContext, actualVersion string,
	embeddedLauncher bool, srcsPathMappings []pathMapping, srcsZip android.Path,
	depsSrcsZips android.Paths) android.OptionalPath {

	main := ""
	if binary.autorun() {
		main = binary.getPyMainFile(ctx, srcsPathMappings)
	}

	var launcherPath android.OptionalPath
	if embeddedLauncher {
		ctx.VisitDirectDepsWithTag(launcherTag, func(m android.Module) {
			if provider, ok := m.(IntermPathProvider); ok {
				if launcherPath.Valid() {
					panic(fmt.Errorf("launcher path was found before: %q",
						launcherPath))
				}
				launcherPath = provider.IntermPathForModuleOut()
			}
		})
	}

	addTopDirectoriesToPath := !proptools.BoolDefault(binary.binaryProperties.Dont_add_top_level_directories_to_path, true)
	dontAddEntrypointFolderToPath := proptools.BoolDefault(binary.binaryProperties.Dont_add_entrypoint_folder_to_path, true)

	binFile := registerBuildActionForParFile(ctx, embeddedLauncher, launcherPath,
		binary.getHostInterpreterName(ctx, actualVersion),
		main, binary.getStem(ctx), append(android.Paths{srcsZip}, depsSrcsZips...),
		addTopDirectoriesToPath, dontAddEntrypointFolderToPath)

	return android.OptionalPathForPath(binFile)
}

// get host interpreter name.
func (binary *binaryDecorator) getHostInterpreterName(ctx android.ModuleContext,
	actualVersion string) string {
	var interp string
	switch actualVersion {
	case pyVersion2:
		interp = "python2.7"
	case pyVersion3:
		interp = "python3"
	default:
		panic(fmt.Errorf("unknown Python actualVersion: %q for module: %q.",
			actualVersion, ctx.ModuleName()))
	}

	return interp
}

// find main program path within runfiles tree.
func (binary *binaryDecorator) getPyMainFile(ctx android.ModuleContext,
	srcsPathMappings []pathMapping) string {
	var main string
	if String(binary.binaryProperties.Main) == "" {
		main = ctx.ModuleName() + pyExt
	} else {
		main = String(binary.binaryProperties.Main)
	}

	for _, path := range srcsPathMappings {
		if main == path.src.Rel() {
			return path.dest
		}
	}
	ctx.PropertyErrorf("main", "%q is not listed in srcs.", main)

	return ""
}

func (binary *binaryDecorator) getStem(ctx android.ModuleContext) string {
	stem := ctx.ModuleName()
	if String(binary.binaryProperties.Stem) != "" {
		stem = String(binary.binaryProperties.Stem)
	}

	return stem + String(binary.binaryProperties.Suffix)
}
