// Copyright 2021 Google Inc. All rights reserved.
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

package cc

import (
	"fmt"
	"strings"

	"android/soong/android"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

// TODO(b/267229066): Remove globalAfdoProfileProjects after implementing bp2build converter for fdo_profile
var (
	globalAfdoProfileProjects = []string{
		"vendor/google_data/pgo_profile/sampling/",
		"toolchain/pgo-profiles/sampling/",
	}
)

var afdoProfileProjectsConfigKey = android.NewOnceKey("AfdoProfileProjects")

const afdoCFlagsFormat = "-funique-internal-linkage-names -fprofile-sample-accurate -fprofile-sample-use=%s"

func recordMissingAfdoProfileFile(ctx android.BaseModuleContext, missing string) {
	getNamedMapForConfig(ctx.Config(), modulesMissingProfileFileKey).Store(missing, true)
}

type afdoRdep struct {
	VariationName *string
	ProfilePath   *string
}

type AfdoProperties struct {
	// Afdo allows developers self-service enroll for
	// automatic feedback-directed optimization using profile data.
	Afdo bool

	FdoProfilePath *string `blueprint:"mutated"`

	AfdoRDeps []afdoRdep `blueprint:"mutated"`
}

type afdo struct {
	Properties AfdoProperties
}

func (afdo *afdo) props() []interface{} {
	return []interface{}{&afdo.Properties}
}

// afdoEnabled returns true for binaries and shared libraries
// that set afdo prop to True and there is a profile available
func (afdo *afdo) afdoEnabled() bool {
	return afdo != nil && afdo.Properties.Afdo && afdo.Properties.FdoProfilePath != nil
}

func (afdo *afdo) flags(ctx ModuleContext, flags Flags) Flags {
	if path := afdo.Properties.FdoProfilePath; path != nil {
		// The flags are prepended to allow overriding.
		profileUseFlag := fmt.Sprintf(afdoCFlagsFormat, *path)
		flags.Local.CFlags = append([]string{profileUseFlag}, flags.Local.CFlags...)
		flags.Local.LdFlags = append([]string{profileUseFlag, "-Wl,-mllvm,-no-warn-sample-unused=true"}, flags.Local.LdFlags...)

		// Update CFlagsDeps and LdFlagsDeps so the module is rebuilt
		// if profileFile gets updated
		pathForSrc := android.PathForSource(ctx, *path)
		flags.CFlagsDeps = append(flags.CFlagsDeps, pathForSrc)
		flags.LdFlagsDeps = append(flags.LdFlagsDeps, pathForSrc)
	}

	return flags
}

func (afdo *afdo) addDep(ctx BaseModuleContext, actx android.BottomUpMutatorContext) {
	if ctx.Host() {
		return
	}

	if ctx.static() && !ctx.staticBinary() {
		return
	}

	if c, ok := ctx.Module().(*Module); ok && c.Enabled() {
		if fdoProfileName, err := actx.DeviceConfig().AfdoProfile(actx.ModuleName()); fdoProfileName != nil && err == nil {
			actx.AddFarVariationDependencies(
				[]blueprint.Variation{
					{Mutator: "arch", Variation: actx.Target().ArchVariation()},
					{Mutator: "os", Variation: "android"},
				},
				FdoProfileTag,
				[]string{*fdoProfileName}...,
			)
		}
	}
}

// FdoProfileMutator reads the FdoProfileProvider from a direct dep with FdoProfileTag
// assigns FdoProfileInfo.Path to the FdoProfilePath mutated property
func (c *Module) fdoProfileMutator(ctx android.BottomUpMutatorContext) {
	if !c.Enabled() {
		return
	}

	ctx.VisitDirectDepsWithTag(FdoProfileTag, func(m android.Module) {
		if ctx.OtherModuleHasProvider(m, FdoProfileProvider) {
			info := ctx.OtherModuleProvider(m, FdoProfileProvider).(FdoProfileInfo)
			c.afdo.Properties.FdoProfilePath = proptools.StringPtr(info.Path.String())
		}
	})
}

var _ FdoProfileMutatorInterface = (*Module)(nil)

// Propagate afdo requirements down from binaries and shared libraries
func afdoDepsMutator(mctx android.TopDownMutatorContext) {
	if m, ok := mctx.Module().(*Module); ok && m.afdo.afdoEnabled() {
		if path := m.afdo.Properties.FdoProfilePath; path != nil {
			mctx.WalkDeps(func(dep android.Module, parent android.Module) bool {
				tag := mctx.OtherModuleDependencyTag(dep)
				libTag, isLibTag := tag.(libraryDependencyTag)

				// Do not recurse down non-static dependencies
				if isLibTag {
					if !libTag.static() {
						return false
					}
				} else {
					if tag != objDepTag && tag != reuseObjTag {
						return false
					}
				}

				if dep, ok := dep.(*Module); ok {
					dep.afdo.Properties.AfdoRDeps = append(
						dep.afdo.Properties.AfdoRDeps,
						afdoRdep{
							VariationName: proptools.StringPtr(encodeTarget(m.Name())),
							ProfilePath:   path,
						},
					)
				}

				return true
			})
		}
	}
}

// Create afdo variants for modules that need them
func afdoMutator(mctx android.BottomUpMutatorContext) {
	if m, ok := mctx.Module().(*Module); ok && m.afdo != nil {
		if !m.static() && m.afdo.Properties.Afdo && m.afdo.Properties.FdoProfilePath != nil {
			mctx.SetDependencyVariation(encodeTarget(m.Name()))
			return
		}

		variationNames := []string{""}

		variantNameToProfilePath := make(map[string]*string)

		for _, afdoRDep := range m.afdo.Properties.AfdoRDeps {
			variantName := *afdoRDep.VariationName
			// An rdep can be set twice in AfdoRDeps because there can be
			// more than one path from an afdo-enabled module to
			// a static dep such as
			// afdo_enabled_foo -> static_bar ----> static_baz
			//                   \                      ^
			//                    ----------------------|
			// We only need to create one variant per unique rdep
			if variantNameToProfilePath[variantName] == nil {
				variationNames = append(variationNames, variantName)
				variantNameToProfilePath[variantName] = afdoRDep.ProfilePath
			}
		}

		if len(variationNames) > 1 {
			modules := mctx.CreateVariations(variationNames...)
			for i, name := range variationNames {
				if name == "" {
					continue
				}
				variation := modules[i].(*Module)
				variation.Properties.PreventInstall = true
				variation.Properties.HideFromMake = true
				variation.afdo.Properties.FdoProfilePath = variantNameToProfilePath[name]
			}
		}
	}
}

// Encode target name to variation name.
func encodeTarget(target string) string {
	if target == "" {
		return ""
	}
	return "afdo-" + target
}

// Decode target name from variation name.
func decodeTarget(variation string) string {
	if variation == "" {
		return ""
	}
	return strings.TrimPrefix(variation, "afdo-")
}
