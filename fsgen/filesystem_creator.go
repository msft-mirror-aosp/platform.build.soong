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
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"

	"android/soong/android"
	"android/soong/filesystem"

	"github.com/google/blueprint"
	"github.com/google/blueprint/parser"
	"github.com/google/blueprint/proptools"
)

var pctx = android.NewPackageContext("android/soong/fsgen")

func init() {
	registerBuildComponents(android.InitRegistrationContext)
}

func registerBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("soong_filesystem_creator", filesystemCreatorFactory)
	ctx.PreDepsMutators(RegisterCollectFileSystemDepsMutators)
}

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
type multilibDeps *map[string]*depCandidateProps

// Information necessary to generate the filesystem modules, including details about their
// dependencies
type FsGenState struct {
	// List of modules in `PRODUCT_PACKAGES` and `PRODUCT_PACKAGES_DEBUG`
	depCandidates []string
	// Map of names of partition to the information of modules to be added as deps
	fsDeps map[string]multilibDeps
	// List of name of partitions to be generated by the filesystem_creator module
	soongGeneratedPartitions []string
	// Mutex to protect the fsDeps
	fsDepsMutex sync.Mutex
	// Map of _all_ soong module names to their corresponding installation properties
	moduleToInstallationProps map[string]installationProperties
}

type installationProperties struct {
	Required  []string
	Overrides []string
}

func newMultilibDeps() multilibDeps {
	return &map[string]*depCandidateProps{}
}

func defaultDepCandidateProps(config android.Config) *depCandidateProps {
	return &depCandidateProps{
		Namespace: ".",
		Arch:      []android.ArchType{config.BuildArch},
	}
}

func createFsGenState(ctx android.LoadHookContext) *FsGenState {
	return ctx.Config().Once(fsGenStateOnceKey, func() interface{} {
		partitionVars := ctx.Config().ProductVariables().PartitionVarsForSoongMigrationOnlyDoNotUse
		candidates := android.FirstUniqueStrings(android.Concat(partitionVars.ProductPackages, partitionVars.ProductPackagesDebug))

		generatedPartitions := []string{"system"}
		if ctx.DeviceConfig().SystemExtPath() == "system_ext" {
			generatedPartitions = append(generatedPartitions, "system_ext")
		}
		if ctx.DeviceConfig().BuildingVendorImage() && ctx.DeviceConfig().VendorPath() == "vendor" {
			generatedPartitions = append(generatedPartitions, "vendor")
		}
		if ctx.DeviceConfig().BuildingProductImage() && ctx.DeviceConfig().ProductPath() == "product" {
			generatedPartitions = append(generatedPartitions, "product")
		}

		return &FsGenState{
			depCandidates: candidates,
			fsDeps: map[string]multilibDeps{
				// These additional deps are added according to the cuttlefish system image bp.
				"system": &map[string]*depCandidateProps{
					"com.android.apex.cts.shim.v1_prebuilt":     defaultDepCandidateProps(ctx.Config()),
					"dex_bootjars":                              defaultDepCandidateProps(ctx.Config()),
					"framework_compatibility_matrix.device.xml": defaultDepCandidateProps(ctx.Config()),
					"idc_data":                     defaultDepCandidateProps(ctx.Config()),
					"init.environ.rc-soong":        defaultDepCandidateProps(ctx.Config()),
					"keychars_data":                defaultDepCandidateProps(ctx.Config()),
					"keylayout_data":               defaultDepCandidateProps(ctx.Config()),
					"libclang_rt.asan":             defaultDepCandidateProps(ctx.Config()),
					"libcompiler_rt":               defaultDepCandidateProps(ctx.Config()),
					"libdmabufheap":                defaultDepCandidateProps(ctx.Config()),
					"libgsi":                       defaultDepCandidateProps(ctx.Config()),
					"llndk.libraries.txt":          defaultDepCandidateProps(ctx.Config()),
					"logpersist.start":             defaultDepCandidateProps(ctx.Config()),
					"preloaded-classes":            defaultDepCandidateProps(ctx.Config()),
					"public.libraries.android.txt": defaultDepCandidateProps(ctx.Config()),
					"update_engine_sideload":       defaultDepCandidateProps(ctx.Config()),
				},
				"vendor": &map[string]*depCandidateProps{
					"fs_config_files_vendor":                               defaultDepCandidateProps(ctx.Config()),
					"fs_config_dirs_vendor":                                defaultDepCandidateProps(ctx.Config()),
					generatedModuleName(ctx.Config(), "vendor-build.prop"): defaultDepCandidateProps(ctx.Config()),
				},
				"odm":     newMultilibDeps(),
				"product": newMultilibDeps(),
				"system_ext": &map[string]*depCandidateProps{
					// VNDK apexes are automatically included.
					// This hardcoded list will need to be updated if `PRODUCT_EXTRA_VNDK_VERSIONS` is updated.
					// https://cs.android.com/android/_/android/platform/build/+/adba533072b00c53ac0f198c550a3cbd7a00e4cd:core/main.mk;l=984;bpv=1;bpt=0;drc=174db7b179592cf07cbfd2adb0119486fda911e7
					"com.android.vndk.v30": defaultDepCandidateProps(ctx.Config()),
					"com.android.vndk.v31": defaultDepCandidateProps(ctx.Config()),
					"com.android.vndk.v32": defaultDepCandidateProps(ctx.Config()),
					"com.android.vndk.v33": defaultDepCandidateProps(ctx.Config()),
					"com.android.vndk.v34": defaultDepCandidateProps(ctx.Config()),
				},
			},
			soongGeneratedPartitions:  generatedPartitions,
			fsDepsMutex:               sync.Mutex{},
			moduleToInstallationProps: map[string]installationProperties{},
		}
	}).(*FsGenState)
}

func checkDepModuleInMultipleNamespaces(mctx android.BottomUpMutatorContext, foundDeps map[string]*depCandidateProps, module string, partitionName string) {
	otherNamespace := mctx.Namespace().Path
	if val, found := foundDeps[module]; found && otherNamespace != "." && !android.InList(val.Namespace, []string{".", otherNamespace}) {
		mctx.ModuleErrorf("found in multiple namespaces(%s and %s) when including in %s partition", val.Namespace, otherNamespace, partitionName)
	}
}

func appendDepIfAppropriate(mctx android.BottomUpMutatorContext, deps *map[string]*depCandidateProps, installPartition string) {
	checkDepModuleInMultipleNamespaces(mctx, *deps, mctx.Module().Name(), installPartition)
	if _, ok := (*deps)[mctx.Module().Name()]; ok {
		// Prefer the namespace-specific module over the platform module
		if mctx.Namespace().Path != "." {
			(*deps)[mctx.Module().Name()].Namespace = mctx.Namespace().Path
		}
		(*deps)[mctx.Module().Name()].Arch = append((*deps)[mctx.Module().Name()].Arch, mctx.Module().Target().Arch.ArchType)
	} else {
		multilib, _ := mctx.Module().DecodeMultilib(mctx)
		(*deps)[mctx.Module().Name()] = &depCandidateProps{
			Namespace: mctx.Namespace().Path,
			Multilib:  multilib,
			Arch:      []android.ArchType{mctx.Module().Target().Arch.ArchType},
		}
	}
}

func collectDepsMutator(mctx android.BottomUpMutatorContext) {
	fsGenState := mctx.Config().Get(fsGenStateOnceKey).(*FsGenState)

	m := mctx.Module()
	if m.Target().Os.Class == android.Device && slices.Contains(fsGenState.depCandidates, m.Name()) {
		installPartition := m.PartitionTag(mctx.DeviceConfig())
		fsGenState.fsDepsMutex.Lock()
		// Only add the module as dependency when:
		// - its enabled
		// - its namespace is included in PRODUCT_SOONG_NAMESPACES
		if m.Enabled(mctx) && m.ExportedToMake() {
			appendDepIfAppropriate(mctx, fsGenState.fsDeps[installPartition], installPartition)
		}
		fsGenState.fsDepsMutex.Unlock()
	}
	// store the map of module to (required,overrides) even if the module is not in PRODUCT_PACKAGES.
	// the module might be installed transitively.
	if m.Target().Os.Class == android.Device && m.Enabled(mctx) && m.ExportedToMake() {
		fsGenState.fsDepsMutex.Lock()
		fsGenState.moduleToInstallationProps[m.Name()] = installationProperties{
			Required:  m.RequiredModuleNames(mctx),
			Overrides: m.Overrides(),
		}
		fsGenState.fsDepsMutex.Unlock()
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
		depsStruct := generateDepStruct(*fsDeps[partition])
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

func generateDepStruct(deps map[string]*depCandidateProps) *packagingPropsStruct {
	depsStruct := packagingPropsStruct{}
	for depName, depProps := range deps {
		bitness := getBitness(depProps.Arch)
		fullyQualifiedDepName := fullyQualifiedModuleName(depName, depProps.Namespace)
		if android.InList(depName, HighPriorityDeps) {
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

	return &depsStruct
}

type filesystemCreatorProps struct {
	Generated_partition_types   []string `blueprint:"mutated"`
	Unsupported_partition_types []string `blueprint:"mutated"`
}

type filesystemCreator struct {
	android.ModuleBase

	properties filesystemCreatorProps
}

func filesystemCreatorFactory() android.Module {
	module := &filesystemCreator{}

	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibCommon)
	module.AddProperties(&module.properties)
	android.AddLoadHook(module, func(ctx android.LoadHookContext) {
		createFsGenState(ctx)
		module.createInternalModules(ctx)
	})

	return module
}

func (f *filesystemCreator) createInternalModules(ctx android.LoadHookContext) {
	soongGeneratedPartitions := &ctx.Config().Get(fsGenStateOnceKey).(*FsGenState).soongGeneratedPartitions
	for _, partitionType := range *soongGeneratedPartitions {
		if f.createPartition(ctx, partitionType) {
			f.properties.Generated_partition_types = append(f.properties.Generated_partition_types, partitionType)
		} else {
			f.properties.Unsupported_partition_types = append(f.properties.Unsupported_partition_types, partitionType)
			_, *soongGeneratedPartitions = android.RemoveFromList(partitionType, *soongGeneratedPartitions)
		}
	}
	f.createDeviceModule(ctx)
}

func generatedModuleName(cfg android.Config, suffix string) string {
	prefix := "soong"
	if cfg.HasDeviceProduct() {
		prefix = cfg.DeviceProduct()
	}
	return fmt.Sprintf("%s_generated_%s", prefix, suffix)
}

func generatedModuleNameForPartition(cfg android.Config, partitionType string) string {
	return generatedModuleName(cfg, fmt.Sprintf("%s_image", partitionType))
}

func (f *filesystemCreator) createDeviceModule(ctx android.LoadHookContext) {
	baseProps := &struct {
		Name *string
	}{
		Name: proptools.StringPtr(generatedModuleName(ctx.Config(), "device")),
	}

	// Currently, only the system and system_ext partition module is created.
	partitionProps := &filesystem.PartitionNameProperties{}
	if android.InList("system", f.properties.Generated_partition_types) {
		partitionProps.System_partition_name = proptools.StringPtr(generatedModuleNameForPartition(ctx.Config(), "system"))
	}
	if android.InList("system_ext", f.properties.Generated_partition_types) {
		partitionProps.System_ext_partition_name = proptools.StringPtr(generatedModuleNameForPartition(ctx.Config(), "system_ext"))
	}
	if android.InList("vendor", f.properties.Generated_partition_types) {
		partitionProps.Vendor_partition_name = proptools.StringPtr(generatedModuleNameForPartition(ctx.Config(), "vendor"))
	}
	if android.InList("product", f.properties.Generated_partition_types) {
		partitionProps.Product_partition_name = proptools.StringPtr(generatedModuleNameForPartition(ctx.Config(), "product"))
	}

	ctx.CreateModule(filesystem.AndroidDeviceFactory, baseProps, partitionProps)
}

func partitionSpecificFsProps(fsProps *filesystem.FilesystemProperties, partitionType string) {
	switch partitionType {
	case "system":
		fsProps.Build_logtags = proptools.BoolPtr(true)
		// https://source.corp.google.com/h/googleplex-android/platform/build//639d79f5012a6542ab1f733b0697db45761ab0f3:core/packaging/flags.mk;l=21;drc=5ba8a8b77507f93aa48cc61c5ba3f31a4d0cbf37;bpv=1;bpt=0
		fsProps.Gen_aconfig_flags_pb = proptools.BoolPtr(true)
		// Identical to that of the generic_system_image
		fsProps.Fsverity.Inputs = []string{
			"etc/boot-image.prof",
			"etc/dirty-image-objects",
			"etc/preloaded-classes",
			"etc/classpaths/*.pb",
			"framework/*",
			"framework/*/*",     // framework/{arch}
			"framework/oat/*/*", // framework/oat/{arch}
		}
		fsProps.Fsverity.Libs = []string{":framework-res{.export-package.apk}"}
	case "system_ext":
		fsProps.Fsverity.Inputs = []string{
			"framework/*",
			"framework/*/*",     // framework/{arch}
			"framework/oat/*/*", // framework/oat/{arch}
		}
		fsProps.Fsverity.Libs = []string{":framework-res{.export-package.apk}"}
	case "product":
		fsProps.Gen_aconfig_flags_pb = proptools.BoolPtr(true)
	case "vendor":
		fsProps.Gen_aconfig_flags_pb = proptools.BoolPtr(true)
		fsProps.Symlinks = []filesystem.SymlinkDefinition{
			filesystem.SymlinkDefinition{
				Target: proptools.StringPtr("/odm"),
				Name:   proptools.StringPtr("vendor/odm"),
			},
			filesystem.SymlinkDefinition{
				Target: proptools.StringPtr("/vendor_dlkm/lib/modules"),
				Name:   proptools.StringPtr("vendor/lib/modules"),
			},
		}
		fsProps.Base_dir = proptools.StringPtr("vendor")
	}
}

// Creates a soong module to build the given partition. Returns false if we can't support building
// it.
func (f *filesystemCreator) createPartition(ctx android.LoadHookContext, partitionType string) bool {
	baseProps := generateBaseProps(proptools.StringPtr(generatedModuleNameForPartition(ctx.Config(), partitionType)))

	fsProps, supported := generateFsProps(ctx, partitionType)
	if !supported {
		return false
	}

	if partitionType == "vendor" || partitionType == "product" {
		fsProps.Linker_config_srcs = f.createLinkerConfigSourceFilegroups(ctx, partitionType)
	}

	var module android.Module
	if partitionType == "system" {
		module = ctx.CreateModule(filesystem.SystemImageFactory, baseProps, fsProps)
	} else {
		// Explicitly set the partition.
		fsProps.Partition_type = proptools.StringPtr(partitionType)
		module = ctx.CreateModule(filesystem.FilesystemFactory, baseProps, fsProps)
	}
	module.HideFromMake()
	if partitionType == "vendor" {
		// Create a build prop for vendor
		vendorBuildProps := &struct {
			Name           *string
			Vendor         *bool
			Stem           *string
			Product_config *string
		}{
			Name:           proptools.StringPtr(generatedModuleName(ctx.Config(), "vendor-build.prop")),
			Vendor:         proptools.BoolPtr(true),
			Stem:           proptools.StringPtr("build.prop"),
			Product_config: proptools.StringPtr(":product_config"),
		}
		vendorBuildProp := ctx.CreateModule(
			android.BuildPropFactory,
			vendorBuildProps,
		)
		vendorBuildProp.HideFromMake()
	}
	return true
}

// createLinkerConfigSourceFilegroups creates filegroup modules to generate linker.config.pb for the following partitions
// 1. vendor: Using PRODUCT_VENDOR_LINKER_CONFIG_FRAGMENTS (space separated file list)
// 1. product: Using PRODUCT_PRODUCT_LINKER_CONFIG_FRAGMENTS (space separated file list)
// It creates a filegroup for each file in the fragment list
// The filegroup modules are then added to `linker_config_srcs` of the autogenerated vendor `android_filesystem`.
func (f *filesystemCreator) createLinkerConfigSourceFilegroups(ctx android.LoadHookContext, partitionType string) []string {
	ret := []string{}
	partitionVars := ctx.Config().ProductVariables().PartitionVarsForSoongMigrationOnlyDoNotUse
	var linkerConfigSrcs []string
	if partitionType == "vendor" {
		linkerConfigSrcs = android.FirstUniqueStrings(partitionVars.VendorLinkerConfigSrcs)
	} else if partitionType == "product" {
		linkerConfigSrcs = android.FirstUniqueStrings(partitionVars.ProductLinkerConfigSrcs)
	} else {
		ctx.ModuleErrorf("linker.config.pb is only supported for vendor and product partitions. For system partition, use `android_system_image`")
	}

	if len(linkerConfigSrcs) > 0 {
		// Create a filegroup, and add `:<filegroup_name>` to ret.
		for index, linkerConfigSrc := range linkerConfigSrcs {
			dir := filepath.Dir(linkerConfigSrc)
			base := filepath.Base(linkerConfigSrc)
			fgName := generatedModuleName(ctx.Config(), fmt.Sprintf("%s-linker-config-src%s", partitionType, strconv.Itoa(index)))
			srcs := []string{base}
			fgProps := &struct {
				Name *string
				Srcs proptools.Configurable[[]string]
			}{
				Name: proptools.StringPtr(fgName),
				Srcs: proptools.NewSimpleConfigurable(srcs),
			}
			ctx.CreateModuleInDirectory(
				android.FileGroupFactory,
				dir,
				fgProps,
			)
			ret = append(ret, ":"+fgName)
		}
	}
	return ret
}

type filesystemBaseProperty struct {
	Name             *string
	Compile_multilib *string
}

func generateBaseProps(namePtr *string) *filesystemBaseProperty {
	return &filesystemBaseProperty{
		Name:             namePtr,
		Compile_multilib: proptools.StringPtr("both"),
	}
}

func generateFsProps(ctx android.EarlyModuleContext, partitionType string) (*filesystem.FilesystemProperties, bool) {
	fsProps := &filesystem.FilesystemProperties{}

	partitionVars := ctx.Config().ProductVariables().PartitionVarsForSoongMigrationOnlyDoNotUse
	specificPartitionVars := partitionVars.PartitionQualifiedVariables[partitionType]

	// BOARD_SYSTEMIMAGE_FILE_SYSTEM_TYPE
	fsType := specificPartitionVars.BoardFileSystemType
	if fsType == "" {
		fsType = "ext4" //default
	}
	fsProps.Type = proptools.StringPtr(fsType)
	if filesystem.GetFsTypeFromString(ctx, *fsProps.Type).IsUnknown() {
		// Currently the android_filesystem module type only supports a handful of FS types like ext4, erofs
		return nil, false
	}

	// Don't build this module on checkbuilds, the soong-built partitions are still in-progress
	// and sometimes don't build.
	fsProps.Unchecked_module = proptools.BoolPtr(true)

	// BOARD_AVB_ENABLE
	fsProps.Use_avb = proptools.BoolPtr(partitionVars.BoardAvbEnable)
	// BOARD_AVB_KEY_PATH
	fsProps.Avb_private_key = proptools.StringPtr(specificPartitionVars.BoardAvbKeyPath)
	// BOARD_AVB_ALGORITHM
	fsProps.Avb_algorithm = proptools.StringPtr(specificPartitionVars.BoardAvbAlgorithm)
	// BOARD_AVB_SYSTEM_ROLLBACK_INDEX
	if rollbackIndex, err := strconv.ParseInt(specificPartitionVars.BoardAvbRollbackIndex, 10, 64); err == nil {
		fsProps.Rollback_index = proptools.Int64Ptr(rollbackIndex)
	}

	fsProps.Partition_name = proptools.StringPtr(partitionType)

	fsProps.Base_dir = proptools.StringPtr(partitionType)

	fsProps.Is_auto_generated = proptools.BoolPtr(true)

	partitionSpecificFsProps(fsProps, partitionType)

	// system_image properties that are not set:
	// - filesystemProperties.Avb_hash_algorithm
	// - filesystemProperties.File_contexts
	// - filesystemProperties.Dirs
	// - filesystemProperties.Symlinks
	// - filesystemProperties.Fake_timestamp
	// - filesystemProperties.Uuid
	// - filesystemProperties.Mount_point
	// - filesystemProperties.Include_make_built_files
	// - filesystemProperties.Build_logtags
	// - systemImageProperties.Linker_config_src

	return fsProps, true
}

func (f *filesystemCreator) createDiffTest(ctx android.ModuleContext, partitionType string) android.Path {
	partitionModuleName := generatedModuleNameForPartition(ctx.Config(), partitionType)
	systemImage := ctx.GetDirectDepWithTag(partitionModuleName, generatedFilesystemDepTag)
	filesystemInfo, ok := android.OtherModuleProvider(ctx, systemImage, filesystem.FilesystemProvider)
	if !ok {
		ctx.ModuleErrorf("Expected module %s to provide FileysystemInfo", partitionModuleName)
	}
	makeFileList := android.PathForArbitraryOutput(ctx, fmt.Sprintf("target/product/%s/obj/PACKAGING/%s_intermediates/file_list.txt", ctx.Config().DeviceName(), partitionType))
	// For now, don't allowlist anything. The test will fail, but that's fine in the current
	// early stages where we're just figuring out what we need
	emptyAllowlistFile := android.PathForModuleOut(ctx, fmt.Sprintf("allowlist_%s.txt", partitionModuleName))
	android.WriteFileRule(ctx, emptyAllowlistFile, "")
	diffTestResultFile := android.PathForModuleOut(ctx, fmt.Sprintf("diff_test_%s.txt", partitionModuleName))

	builder := android.NewRuleBuilder(pctx, ctx)
	builder.Command().BuiltTool("file_list_diff").
		Input(makeFileList).
		Input(filesystemInfo.FileListFile).
		Text(partitionModuleName).
		FlagWithInput("--allowlists ", emptyAllowlistFile)
	builder.Command().Text("touch").Output(diffTestResultFile)
	builder.Build(partitionModuleName+" diff test", partitionModuleName+" diff test")
	return diffTestResultFile
}

func createFailingCommand(ctx android.ModuleContext, message string) android.Path {
	hasher := sha256.New()
	hasher.Write([]byte(message))
	filename := fmt.Sprintf("failing_command_%x.txt", hasher.Sum(nil))
	file := android.PathForModuleOut(ctx, filename)
	builder := android.NewRuleBuilder(pctx, ctx)
	builder.Command().Textf("echo %s", proptools.NinjaAndShellEscape(message))
	builder.Command().Text("exit 1 #").Output(file)
	builder.Build("failing command "+filename, "failing command "+filename)
	return file
}

type systemImageDepTagType struct {
	blueprint.BaseDependencyTag
}

var generatedFilesystemDepTag systemImageDepTagType

func (f *filesystemCreator) DepsMutator(ctx android.BottomUpMutatorContext) {
	for _, partitionType := range f.properties.Generated_partition_types {
		ctx.AddDependency(ctx.Module(), generatedFilesystemDepTag, generatedModuleNameForPartition(ctx.Config(), partitionType))
	}
}

func (f *filesystemCreator) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	if ctx.ModuleDir() != "build/soong/fsgen" {
		ctx.ModuleErrorf("There can only be one soong_filesystem_creator in build/soong/fsgen")
	}
	f.HideFromMake()

	var content strings.Builder
	generatedBp := android.PathForModuleOut(ctx, "soong_generated_product_config.bp")
	for _, partition := range ctx.Config().Get(fsGenStateOnceKey).(*FsGenState).soongGeneratedPartitions {
		content.WriteString(generateBpContent(ctx, partition))
		content.WriteString("\n")
	}
	android.WriteFileRule(ctx, generatedBp, content.String())

	ctx.Phony("product_config_to_bp", generatedBp)

	var diffTestFiles []android.Path
	for _, partitionType := range f.properties.Generated_partition_types {
		diffTestFile := f.createDiffTest(ctx, partitionType)
		diffTestFiles = append(diffTestFiles, diffTestFile)
		ctx.Phony(fmt.Sprintf("soong_generated_%s_filesystem_test", partitionType), diffTestFile)
	}
	for _, partitionType := range f.properties.Unsupported_partition_types {
		diffTestFile := createFailingCommand(ctx, fmt.Sprintf("Couldn't build %s partition", partitionType))
		diffTestFiles = append(diffTestFiles, diffTestFile)
		ctx.Phony(fmt.Sprintf("soong_generated_%s_filesystem_test", partitionType), diffTestFile)
	}
	ctx.Phony("soong_generated_filesystem_tests", diffTestFiles...)
}

func generateBpContent(ctx android.EarlyModuleContext, partitionType string) string {
	fsProps, fsTypeSupported := generateFsProps(ctx, partitionType)
	if !fsTypeSupported {
		return ""
	}
	if partitionType == "vendor" {
		return "" // TODO: Handle struct props
	}

	baseProps := generateBaseProps(proptools.StringPtr(generatedModuleNameForPartition(ctx.Config(), partitionType)))
	deps := ctx.Config().Get(fsGenStateOnceKey).(*FsGenState).fsDeps[partitionType]
	depProps := generateDepStruct(*deps)

	result, err := proptools.RepackProperties([]interface{}{baseProps, fsProps, depProps})
	if err != nil {
		ctx.ModuleErrorf(err.Error())
	}

	moduleType := "android_filesystem"
	if partitionType == "system" {
		moduleType = "android_system_image"
	}

	file := &parser.File{
		Defs: []parser.Definition{
			&parser.Module{
				Type: moduleType,
				Map:  *result,
			},
		},
	}
	bytes, err := parser.Print(file)
	if err != nil {
		ctx.ModuleErrorf(err.Error())
	}
	return strings.TrimSpace(string(bytes))
}
