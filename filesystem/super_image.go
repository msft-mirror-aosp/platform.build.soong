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

package filesystem

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"android/soong/android"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

func init() {
	android.RegisterModuleType("super_image", SuperImageFactory)
}

type superImage struct {
	android.ModuleBase

	properties     SuperImageProperties
	partitionProps SuperImagePartitionNameProperties

	installDir android.InstallPath
}

type SuperImageProperties struct {
	// the size of the super partition
	Size *int64
	// the block device where metadata for dynamic partitions is stored
	Metadata_device *string
	// the super partition block device list
	Block_devices []string
	// whether A/B updater is used
	Ab_update *bool
	// whether dynamic partitions is enabled on devices that were launched without this support
	Retrofit *bool
	// whether virtual A/B seamless update is enabled
	Virtual_ab *bool
	// whether retrofitting virtual A/B seamless update is enabled
	Virtual_ab_retrofit *bool
	// whether the output is a sparse image
	Sparse *bool
	// information about how partitions within the super partition are grouped together
	Partition_groups []PartitionGroupsInfo
	// whether dynamic partitions is used
	Use_dynamic_partitions *bool
}

type PartitionGroupsInfo struct {
	Name          string
	GroupSize     string
	PartitionList []string
}

type SuperImagePartitionNameProperties struct {
	// Name of the System partition filesystem module
	System_partition *string
	// Name of the System_ext partition filesystem module
	System_ext_partition *string
	// Name of the System_dlkm partition filesystem module
	System_dlkm_partition *string
	// Name of the System_other partition filesystem module
	System_other_partition *string
	// Name of the Product partition filesystem module
	Product_partition *string
	// Name of the Vendor partition filesystem module
	Vendor_partition *string
	// Name of the Vendor_dlkm partition filesystem module
	Vendor_dlkm_partition *string
	// Name of the Odm partition filesystem module
	Odm_partition *string
	// Name of the Odm_dlkm partition filesystem module
	Odm_dlkm_partition *string
}

type SuperImageInfo struct {
	// The built super.img file, which contains the sub-partitions
	SuperImage android.Path

	// Mapping from the sub-partition type to its re-exported FileSystemInfo providers from the
	// sub-partitions.
	SubImageInfo map[string]FilesystemInfo
}

var SuperImageProvider = blueprint.NewProvider[SuperImageInfo]()

func SuperImageFactory() android.Module {
	module := &superImage{}
	module.AddProperties(&module.properties, &module.partitionProps)
	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibCommon)
	return module
}

type superImageDepTagType struct {
	blueprint.BaseDependencyTag
}

var subImageDepTag superImageDepTagType

func (s *superImage) DepsMutator(ctx android.BottomUpMutatorContext) {
	addDependencyIfDefined := func(dep *string) {
		if dep != nil {
			ctx.AddDependency(ctx.Module(), subImageDepTag, proptools.String(dep))
		}
	}

	addDependencyIfDefined(s.partitionProps.System_partition)
	addDependencyIfDefined(s.partitionProps.System_ext_partition)
	addDependencyIfDefined(s.partitionProps.System_dlkm_partition)
	addDependencyIfDefined(s.partitionProps.System_other_partition)
	addDependencyIfDefined(s.partitionProps.Product_partition)
	addDependencyIfDefined(s.partitionProps.Vendor_partition)
	addDependencyIfDefined(s.partitionProps.Vendor_dlkm_partition)
	addDependencyIfDefined(s.partitionProps.Odm_partition)
	addDependencyIfDefined(s.partitionProps.Odm_dlkm_partition)
}

func (s *superImage) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	miscInfo, deps, subImageInfos := s.buildMiscInfo(ctx)
	builder := android.NewRuleBuilder(pctx, ctx)
	output := android.PathForModuleOut(ctx, s.installFileName())
	lpMake := ctx.Config().HostToolPath(ctx, "lpmake")
	lpMakeDir := filepath.Dir(lpMake.String())
	deps = append(deps, lpMake)
	builder.Command().Textf("PATH=%s:\\$PATH", lpMakeDir).
		BuiltTool("build_super_image").
		Text("-v").
		Input(miscInfo).
		Implicits(deps).
		Output(output)
	builder.Build("build_super_image", fmt.Sprintf("Creating super image %s", s.BaseModuleName()))
	android.SetProvider(ctx, SuperImageProvider, SuperImageInfo{
		SuperImage:   output,
		SubImageInfo: subImageInfos,
	})
	ctx.SetOutputFiles([]android.Path{output}, "")
	ctx.CheckbuildFile(output)
}

func (s *superImage) installFileName() string {
	return s.BaseModuleName() + ".img"
}

func (s *superImage) buildMiscInfo(ctx android.ModuleContext) (android.Path, android.Paths, map[string]FilesystemInfo) {
	var miscInfoString strings.Builder
	addStr := func(name string, value string) {
		miscInfoString.WriteString(name)
		miscInfoString.WriteRune('=')
		miscInfoString.WriteString(value)
		miscInfoString.WriteRune('\n')
	}

	addStr("use_dynamic_partitions", strconv.FormatBool(proptools.Bool(s.properties.Use_dynamic_partitions)))
	addStr("dynamic_partition_retrofit", strconv.FormatBool(proptools.Bool(s.properties.Retrofit)))
	addStr("lpmake", "lpmake")
	addStr("super_metadata_device", proptools.String(s.properties.Metadata_device))
	if len(s.properties.Block_devices) > 0 {
		addStr("super_block_devices", strings.Join(s.properties.Block_devices, " "))
	}
	addStr("super_super_device_size", strconv.Itoa(proptools.Int(s.properties.Size)))
	var groups, partitionList []string
	for _, groupInfo := range s.properties.Partition_groups {
		groups = append(groups, groupInfo.Name)
		partitionList = append(partitionList, groupInfo.PartitionList...)
		addStr("super_"+groupInfo.Name+"_group_size", groupInfo.GroupSize)
		addStr("super_"+groupInfo.Name+"_partition_list", strings.Join(groupInfo.PartitionList, " "))
	}
	initialPartitionListLen := len(partitionList)
	partitionList = android.SortedUniqueStrings(partitionList)
	if len(partitionList) != initialPartitionListLen {
		ctx.ModuleErrorf("Duplicate partitions found in the partition_groups property")
	}
	addStr("super_partition_groups", strings.Join(groups, " "))
	addStr("dynamic_partition_list", strings.Join(partitionList, " "))

	addStr("virtual_ab", strconv.FormatBool(proptools.Bool(s.properties.Virtual_ab)))
	addStr("virtual_ab_retrofit", strconv.FormatBool(proptools.Bool(s.properties.Virtual_ab_retrofit)))
	addStr("ab_update", strconv.FormatBool(proptools.Bool(s.properties.Ab_update)))
	addStr("build_non_sparse_super_partition", strconv.FormatBool(!proptools.Bool(s.properties.Sparse)))

	subImageInfo := make(map[string]FilesystemInfo)
	var deps android.Paths

	missingPartitionErrorMessage := ""
	handleSubPartition := func(partitionType string, name *string) {
		if proptools.String(name) == "" {
			missingPartitionErrorMessage += fmt.Sprintf("%s image listed in partition groups, but its module was not specified. ", partitionType)
			return
		}
		mod := ctx.GetDirectDepWithTag(*name, subImageDepTag)
		if mod == nil {
			ctx.ModuleErrorf("Could not get dep %q", *name)
			return
		}
		info, ok := android.OtherModuleProvider(ctx, mod, FilesystemProvider)
		if !ok {
			ctx.ModuleErrorf("Expected dep %q to provide FilesystemInfo", *name)
			return
		}
		addStr(partitionType+"_image", info.Output.String())
		deps = append(deps, info.Output)
		if _, ok := subImageInfo[partitionType]; ok {
			ctx.ModuleErrorf("Already set subimageInfo for %q", partitionType)
		}
		subImageInfo[partitionType] = info
	}

	// Build partitionToImagePath, because system partition may need system_other
	// partition image path
	for _, p := range partitionList {
		switch p {
		case "system":
			handleSubPartition("system", s.partitionProps.System_partition)
			// TODO: add system_other to deps after it can be generated
			//getFsInfo("system_other", s.partitionProps.System_other_partition, &subImageInfo.System_other)
		case "system_dlkm":
			handleSubPartition("system_dlkm", s.partitionProps.System_dlkm_partition)
		case "system_ext":
			handleSubPartition("system_ext", s.partitionProps.System_ext_partition)
		case "product":
			handleSubPartition("product", s.partitionProps.Product_partition)
		case "vendor":
			handleSubPartition("vendor", s.partitionProps.Vendor_partition)
		case "vendor_dlkm":
			handleSubPartition("vendor_dlkm", s.partitionProps.Vendor_dlkm_partition)
		case "odm":
			handleSubPartition("odm", s.partitionProps.Odm_partition)
		case "odm_dlkm":
			handleSubPartition("odm_dlkm", s.partitionProps.Odm_dlkm_partition)
		default:
			ctx.ModuleErrorf("partition %q is not a super image supported partition", p)
		}
	}

	// Delay the error message until execution time because on aosp-main-future-without-vendor,
	// BUILDING_VENDOR_IMAGE is false so we don't get the vendor image, but it's still listed in
	// BOARD_GOOGLE_DYNAMIC_PARTITIONS_PARTITION_LIST.
	missingPartitionErrorMessageFile := android.PathForModuleOut(ctx, "missing_partition_error.txt")
	if missingPartitionErrorMessage != "" {
		ctx.Build(pctx, android.BuildParams{
			Rule:   android.ErrorRule,
			Output: missingPartitionErrorMessageFile,
			Args: map[string]string{
				"error": missingPartitionErrorMessage,
			},
		})
	} else {
		ctx.Build(pctx, android.BuildParams{
			Rule:   android.Touch,
			Output: missingPartitionErrorMessageFile,
		})
	}

	miscInfo := android.PathForModuleOut(ctx, "misc_info.txt")
	android.WriteFileRule(ctx, miscInfo, miscInfoString.String(), missingPartitionErrorMessageFile)
	return miscInfo, deps, subImageInfo
}
