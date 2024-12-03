// Copyright (C) 2024 The Android Open Source Project
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

package fsgen

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	"android/soong/android"

	"github.com/google/blueprint/proptools"
)

func RegisterCollectFileSystemDepsMutators(ctx android.RegisterMutatorsContext) {
	ctx.BottomUp("fs_collect_deps", collectDepsMutator).MutatesGlobalState()
	ctx.BottomUp("fs_set_deps", setDepsMutator)
}

var fsGenStateOnceKey = android.NewOnceKey("FsGenState")
var fsGenRemoveOverridesOnceKey = android.NewOnceKey("FsGenRemoveOverrides")

// Map of partition module name to its partition that may be generated by Soong.
// Note that it is not guaranteed that all modules returned by this function are successfully
// created.
func getAllSoongGeneratedPartitionNames(config android.Config, partitions []string) map[string]string {
	ret := map[string]string{}
	for _, partition := range partitions {
		ret[generatedModuleNameForPartition(config, partition)] = partition
	}
	return ret
}

type depCandidateProps struct {
	Namespace string
	Multilib  string
	Arch      []android.ArchType
}

// Map of module name to depCandidateProps
type multilibDeps map[string]*depCandidateProps

// Information necessary to generate the filesystem modules, including details about their
// dependencies
type FsGenState struct {
	// List of modules in `PRODUCT_PACKAGES` and `PRODUCT_PACKAGES_DEBUG`
	depCandidates []string
	// Map of names of partition to the information of modules to be added as deps
	fsDeps map[string]*multilibDeps
	// List of name of partitions to be generated by the filesystem_creator module
	soongGeneratedPartitions []string
	// Mutex to protect the fsDeps
	fsDepsMutex sync.Mutex
	// Map of _all_ soong module names to their corresponding installation properties
	moduleToInstallationProps map[string]installationProperties
	// List of prebuilt_* modules that are autogenerated.
	generatedPrebuiltEtcModuleNames []string
	// Mapping from a path to an avb key to the name of a filegroup module that contains it
	avbKeyFilegroups map[string]string
}

type installationProperties struct {
	Required  []string
	Overrides []string
}

func defaultDepCandidateProps(config android.Config) *depCandidateProps {
	return &depCandidateProps{
		Namespace: ".",
		Arch:      []android.ArchType{config.BuildArch},
	}
}

func createFsGenState(ctx android.LoadHookContext, generatedPrebuiltEtcModuleNames []string, avbpubkeyGenerated bool) *FsGenState {
	return ctx.Config().Once(fsGenStateOnceKey, func() interface{} {
		partitionVars := ctx.Config().ProductVariables().PartitionVarsForSoongMigrationOnlyDoNotUse
		candidates := android.FirstUniqueStrings(android.Concat(partitionVars.ProductPackages, partitionVars.ProductPackagesDebug))
		candidates = android.Concat(candidates, generatedPrebuiltEtcModuleNames)

		fsGenState := FsGenState{
			depCandidates: candidates,
			fsDeps: map[string]*multilibDeps{
				// These additional deps are added according to the cuttlefish system image bp.
				"system": {
					// keep-sorted start
					"com.android.apex.cts.shim.v1_prebuilt":     defaultDepCandidateProps(ctx.Config()),
					"dex_bootjars":                              defaultDepCandidateProps(ctx.Config()),
					"framework_compatibility_matrix.device.xml": defaultDepCandidateProps(ctx.Config()),
					"init.environ.rc-soong":                     defaultDepCandidateProps(ctx.Config()),
					"libcompiler_rt":                            defaultDepCandidateProps(ctx.Config()),
					"libdmabufheap":                             defaultDepCandidateProps(ctx.Config()),
					"libgsi":                                    defaultDepCandidateProps(ctx.Config()),
					"llndk.libraries.txt":                       defaultDepCandidateProps(ctx.Config()),
					"logpersist.start":                          defaultDepCandidateProps(ctx.Config()),
					"update_engine_sideload":                    defaultDepCandidateProps(ctx.Config()),
					// keep-sorted end
				},
				"vendor": {
					"fs_config_files_vendor":                               defaultDepCandidateProps(ctx.Config()),
					"fs_config_dirs_vendor":                                defaultDepCandidateProps(ctx.Config()),
					generatedModuleName(ctx.Config(), "vendor-build.prop"): defaultDepCandidateProps(ctx.Config()),
				},
				"odm": {
					// fs_config_* files are automatically installed for all products with odm partitions.
					// https://cs.android.com/android/_/android/platform/build/+/e4849e87ab660b59a6501b3928693db065ee873b:tools/fs_config/Android.mk;l=34;drc=8d6481b92c4b4e9b9f31a61545b6862090fcc14b;bpv=1;bpt=0
					"fs_config_files_odm": defaultDepCandidateProps(ctx.Config()),
					"fs_config_dirs_odm":  defaultDepCandidateProps(ctx.Config()),
				},
				"product": {},
				"system_ext": {
					// VNDK apexes are automatically included.
					// This hardcoded list will need to be updated if `PRODUCT_EXTRA_VNDK_VERSIONS` is updated.
					// https://cs.android.com/android/_/android/platform/build/+/adba533072b00c53ac0f198c550a3cbd7a00e4cd:core/main.mk;l=984;bpv=1;bpt=0;drc=174db7b179592cf07cbfd2adb0119486fda911e7
					"com.android.vndk.v30": defaultDepCandidateProps(ctx.Config()),
					"com.android.vndk.v31": defaultDepCandidateProps(ctx.Config()),
					"com.android.vndk.v32": defaultDepCandidateProps(ctx.Config()),
					"com.android.vndk.v33": defaultDepCandidateProps(ctx.Config()),
					"com.android.vndk.v34": defaultDepCandidateProps(ctx.Config()),
				},
				"userdata": {},
				"system_dlkm": {
					// these are phony required deps of the phony fs_config_dirs_nonsystem
					"fs_config_dirs_system_dlkm":  defaultDepCandidateProps(ctx.Config()),
					"fs_config_files_system_dlkm": defaultDepCandidateProps(ctx.Config()),
					// build props are automatically added to `ALL_DEFAULT_INSTALLED_MODULES`
					"system_dlkm-build.prop": defaultDepCandidateProps(ctx.Config()),
				},
				"vendor_dlkm": {
					"fs_config_dirs_vendor_dlkm":  defaultDepCandidateProps(ctx.Config()),
					"fs_config_files_vendor_dlkm": defaultDepCandidateProps(ctx.Config()),
					"vendor_dlkm-build.prop":      defaultDepCandidateProps(ctx.Config()),
				},
				"odm_dlkm": {
					"fs_config_dirs_odm_dlkm":  defaultDepCandidateProps(ctx.Config()),
					"fs_config_files_odm_dlkm": defaultDepCandidateProps(ctx.Config()),
					"odm_dlkm-build.prop":      defaultDepCandidateProps(ctx.Config()),
				},
				"ramdisk":        {},
				"vendor_ramdisk": {},
				"recovery":       {},
			},
			fsDepsMutex:                     sync.Mutex{},
			moduleToInstallationProps:       map[string]installationProperties{},
			generatedPrebuiltEtcModuleNames: generatedPrebuiltEtcModuleNames,
			avbKeyFilegroups:                map[string]string{},
		}

		if avbpubkeyGenerated {
			(*fsGenState.fsDeps["product"])["system_other_avbpubkey"] = defaultDepCandidateProps(ctx.Config())
		}

		return &fsGenState
	}).(*FsGenState)
}

func checkDepModuleInMultipleNamespaces(mctx android.BottomUpMutatorContext, foundDeps multilibDeps, module string, partitionName string) {
	otherNamespace := mctx.Namespace().Path
	if val, found := foundDeps[module]; found && otherNamespace != "." && !android.InList(val.Namespace, []string{".", otherNamespace}) {
		mctx.ModuleErrorf("found in multiple namespaces(%s and %s) when including in %s partition", val.Namespace, otherNamespace, partitionName)
	}
}

func appendDepIfAppropriate(mctx android.BottomUpMutatorContext, deps *multilibDeps, installPartition string) {
	moduleName := mctx.ModuleName()
	checkDepModuleInMultipleNamespaces(mctx, *deps, moduleName, installPartition)
	if _, ok := (*deps)[moduleName]; ok {
		// Prefer the namespace-specific module over the platform module
		if mctx.Namespace().Path != "." {
			(*deps)[moduleName].Namespace = mctx.Namespace().Path
		}
		(*deps)[moduleName].Arch = append((*deps)[moduleName].Arch, mctx.Module().Target().Arch.ArchType)
	} else {
		multilib, _ := mctx.Module().DecodeMultilib(mctx)
		(*deps)[moduleName] = &depCandidateProps{
			Namespace: mctx.Namespace().Path,
			Multilib:  multilib,
			Arch:      []android.ArchType{mctx.Module().Target().Arch.ArchType},
		}
	}
}

func collectDepsMutator(mctx android.BottomUpMutatorContext) {
	m := mctx.Module()
	if m.Target().Os.Class != android.Device {
		return
	}
	fsGenState := mctx.Config().Get(fsGenStateOnceKey).(*FsGenState)

	fsGenState.fsDepsMutex.Lock()
	defer fsGenState.fsDepsMutex.Unlock()

	if slices.Contains(fsGenState.depCandidates, mctx.ModuleName()) {
		installPartition := m.PartitionTag(mctx.DeviceConfig())
		// Only add the module as dependency when:
		// - its enabled
		// - its namespace is included in PRODUCT_SOONG_NAMESPACES
		if m.Enabled(mctx) && m.ExportedToMake() {
			appendDepIfAppropriate(mctx, fsGenState.fsDeps[installPartition], installPartition)
		}
	}
	// store the map of module to (required,overrides) even if the module is not in PRODUCT_PACKAGES.
	// the module might be installed transitively.
	if m.Enabled(mctx) && m.ExportedToMake() {
		fsGenState.moduleToInstallationProps[m.Name()] = installationProperties{
			Required:  m.RequiredModuleNames(mctx),
			Overrides: m.Overrides(),
		}
	}
}

type depsStruct struct {
	Deps []string
}

type multilibDepsStruct struct {
	Common   depsStruct
	Lib32    depsStruct
	Lib64    depsStruct
	Both     depsStruct
	Prefer32 depsStruct
}

type packagingPropsStruct struct {
	High_priority_deps []string
	Deps               []string
	Multilib           multilibDepsStruct
}

func fullyQualifiedModuleName(moduleName, namespace string) string {
	if namespace == "." {
		return moduleName
	}
	return fmt.Sprintf("//%s:%s", namespace, moduleName)
}

func getBitness(archTypes []android.ArchType) (ret []string) {
	for _, archType := range archTypes {
		if archType.Multilib == "" {
			ret = append(ret, android.COMMON_VARIANT)
		} else {
			ret = append(ret, archType.Bitness())
		}
	}
	return ret
}

func setDepsMutator(mctx android.BottomUpMutatorContext) {
	removeOverriddenDeps(mctx)
	fsGenState := mctx.Config().Get(fsGenStateOnceKey).(*FsGenState)
	fsDeps := fsGenState.fsDeps
	soongGeneratedPartitionMap := getAllSoongGeneratedPartitionNames(mctx.Config(), fsGenState.soongGeneratedPartitions)
	m := mctx.Module()
	if partition, ok := soongGeneratedPartitionMap[m.Name()]; ok {
		depsStruct := generateDepStruct(*fsDeps[partition], fsGenState.generatedPrebuiltEtcModuleNames)
		if err := proptools.AppendMatchingProperties(m.GetProperties(), depsStruct, nil); err != nil {
			mctx.ModuleErrorf(err.Error())
		}
	}
}

// removeOverriddenDeps collects PRODUCT_PACKAGES and (transitive) required deps.
// it then removes any modules which appear in `overrides` of the above list.
func removeOverriddenDeps(mctx android.BottomUpMutatorContext) {
	mctx.Config().Once(fsGenRemoveOverridesOnceKey, func() interface{} {
		fsGenState := mctx.Config().Get(fsGenStateOnceKey).(*FsGenState)
		fsDeps := fsGenState.fsDeps
		overridden := map[string]bool{}
		allDeps := []string{}

		// Step 1: Initialization: Append PRODUCT_PACKAGES to the queue
		for _, fsDep := range fsDeps {
			for depName, _ := range *fsDep {
				allDeps = append(allDeps, depName)
			}
		}

		// Step 2: Process the queue, and add required modules to the queue.
		i := 0
		for {
			if i == len(allDeps) {
				break
			}
			depName := allDeps[i]
			for _, overrides := range fsGenState.moduleToInstallationProps[depName].Overrides {
				overridden[overrides] = true
			}
			// add required dep to the queue.
			allDeps = append(allDeps, fsGenState.moduleToInstallationProps[depName].Required...)
			i += 1
		}

		// Step 3: Delete all the overridden modules.
		for overridden, _ := range overridden {
			for partition, _ := range fsDeps {
				delete(*fsDeps[partition], overridden)
			}
		}
		return nil
	})
}

var HighPriorityDeps = []string{}

func isHighPriorityDep(depName string) bool {
	for _, highPriorityDeps := range HighPriorityDeps {
		if strings.HasPrefix(depName, highPriorityDeps) {
			return true
		}
	}
	return false
}

func generateDepStruct(deps map[string]*depCandidateProps, highPriorityDeps []string) *packagingPropsStruct {
	depsStruct := packagingPropsStruct{}
	for depName, depProps := range deps {
		bitness := getBitness(depProps.Arch)
		fullyQualifiedDepName := fullyQualifiedModuleName(depName, depProps.Namespace)
		if android.InList(depName, highPriorityDeps) {
			depsStruct.High_priority_deps = append(depsStruct.High_priority_deps, fullyQualifiedDepName)
		} else if android.InList("32", bitness) && android.InList("64", bitness) {
			// If both 32 and 64 bit variants are enabled for this module
			switch depProps.Multilib {
			case string(android.MultilibBoth):
				depsStruct.Multilib.Both.Deps = append(depsStruct.Multilib.Both.Deps, fullyQualifiedDepName)
			case string(android.MultilibCommon), string(android.MultilibFirst):
				depsStruct.Deps = append(depsStruct.Deps, fullyQualifiedDepName)
			case "32":
				depsStruct.Multilib.Lib32.Deps = append(depsStruct.Multilib.Lib32.Deps, fullyQualifiedDepName)
			case "64", "darwin_universal":
				depsStruct.Multilib.Lib64.Deps = append(depsStruct.Multilib.Lib64.Deps, fullyQualifiedDepName)
			case "prefer32", "first_prefer32":
				depsStruct.Multilib.Prefer32.Deps = append(depsStruct.Multilib.Prefer32.Deps, fullyQualifiedDepName)
			default:
				depsStruct.Multilib.Both.Deps = append(depsStruct.Multilib.Both.Deps, fullyQualifiedDepName)
			}
		} else if android.InList("64", bitness) {
			// If only 64 bit variant is enabled
			depsStruct.Multilib.Lib64.Deps = append(depsStruct.Multilib.Lib64.Deps, fullyQualifiedDepName)
		} else if android.InList("32", bitness) {
			// If only 32 bit variant is enabled
			depsStruct.Multilib.Lib32.Deps = append(depsStruct.Multilib.Lib32.Deps, fullyQualifiedDepName)
		} else {
			// If only common variant is enabled
			depsStruct.Multilib.Common.Deps = append(depsStruct.Multilib.Common.Deps, fullyQualifiedDepName)
		}
	}
	depsStruct.Deps = android.SortedUniqueStrings(depsStruct.Deps)
	depsStruct.Multilib.Lib32.Deps = android.SortedUniqueStrings(depsStruct.Multilib.Lib32.Deps)
	depsStruct.Multilib.Lib64.Deps = android.SortedUniqueStrings(depsStruct.Multilib.Lib64.Deps)
	depsStruct.Multilib.Prefer32.Deps = android.SortedUniqueStrings(depsStruct.Multilib.Prefer32.Deps)
	depsStruct.Multilib.Both.Deps = android.SortedUniqueStrings(depsStruct.Multilib.Both.Deps)
	depsStruct.Multilib.Common.Deps = android.SortedUniqueStrings(depsStruct.Multilib.Common.Deps)
	depsStruct.High_priority_deps = android.SortedUniqueStrings(depsStruct.High_priority_deps)

	return &depsStruct
}
