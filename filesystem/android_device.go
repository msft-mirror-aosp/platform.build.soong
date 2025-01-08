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
	"strings"

	"android/soong/android"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

type PartitionNameProperties struct {
	// Name of the boot partition filesystem module
	Boot_partition_name *string
	// Name of the vendor boot partition filesystem module
	Vendor_boot_partition_name *string
	// Name of the init boot partition filesystem module
	Init_boot_partition_name *string
	// Name of the system partition filesystem module
	System_partition_name *string
	// Name of the system_ext partition filesystem module
	System_ext_partition_name *string
	// Name of the product partition filesystem module
	Product_partition_name *string
	// Name of the vendor partition filesystem module
	Vendor_partition_name *string
	// Name of the odm partition filesystem module
	Odm_partition_name *string
	// Name of the recovery partition filesystem module
	Recovery_partition_name *string
	// The vbmeta partition and its "chained" partitions
	Vbmeta_partitions []string
	// Name of the userdata partition filesystem module
	Userdata_partition_name *string
	// Name of the system_dlkm partition filesystem module
	System_dlkm_partition_name *string
	// Name of the vendor_dlkm partition filesystem module
	Vendor_dlkm_partition_name *string
	// Name of the odm_dlkm partition filesystem module
	Odm_dlkm_partition_name *string
}

type androidDevice struct {
	android.ModuleBase

	partitionProps PartitionNameProperties
}

func AndroidDeviceFactory() android.Module {
	module := &androidDevice{}
	module.AddProperties(&module.partitionProps)
	android.InitAndroidMultiTargetsArchModule(module, android.DeviceSupported, android.MultilibFirst)
	return module
}

type partitionDepTagType struct {
	blueprint.BaseDependencyTag
}

var filesystemDepTag partitionDepTagType

func (a *androidDevice) DepsMutator(ctx android.BottomUpMutatorContext) {
	addDependencyIfDefined := func(dep *string) {
		if dep != nil {
			ctx.AddDependency(ctx.Module(), filesystemDepTag, proptools.String(dep))
		}
	}

	addDependencyIfDefined(a.partitionProps.Boot_partition_name)
	addDependencyIfDefined(a.partitionProps.Init_boot_partition_name)
	addDependencyIfDefined(a.partitionProps.Vendor_boot_partition_name)
	addDependencyIfDefined(a.partitionProps.System_partition_name)
	addDependencyIfDefined(a.partitionProps.System_ext_partition_name)
	addDependencyIfDefined(a.partitionProps.Product_partition_name)
	addDependencyIfDefined(a.partitionProps.Vendor_partition_name)
	addDependencyIfDefined(a.partitionProps.Odm_partition_name)
	addDependencyIfDefined(a.partitionProps.Userdata_partition_name)
	addDependencyIfDefined(a.partitionProps.System_dlkm_partition_name)
	addDependencyIfDefined(a.partitionProps.Vendor_dlkm_partition_name)
	addDependencyIfDefined(a.partitionProps.Odm_dlkm_partition_name)
	addDependencyIfDefined(a.partitionProps.Recovery_partition_name)
	for _, vbmetaPartition := range a.partitionProps.Vbmeta_partitions {
		ctx.AddDependency(ctx.Module(), filesystemDepTag, vbmetaPartition)
	}
}

func (a *androidDevice) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	a.buildTargetFilesZip(ctx)
}

type targetFilesZipCopy struct {
	srcModule  *string
	destSubdir string
}

func (a *androidDevice) buildTargetFilesZip(ctx android.ModuleContext) {
	targetFilesDir := android.PathForModuleOut(ctx, "target_files_dir")
	targetFilesZip := android.PathForModuleOut(ctx, "target_files.zip")

	builder := android.NewRuleBuilder(pctx, ctx)
	builder.Command().Textf("rm -rf %s", targetFilesDir.String())
	builder.Command().Textf("mkdir -p %s", targetFilesDir.String())
	toCopy := []targetFilesZipCopy{
		targetFilesZipCopy{a.partitionProps.System_partition_name, "SYSTEM"},
		targetFilesZipCopy{a.partitionProps.System_ext_partition_name, "SYSTEM_EXT"},
		targetFilesZipCopy{a.partitionProps.Product_partition_name, "PRODUCT"},
		targetFilesZipCopy{a.partitionProps.Vendor_partition_name, "VENDOR"},
		targetFilesZipCopy{a.partitionProps.Odm_partition_name, "ODM"},
		targetFilesZipCopy{a.partitionProps.System_dlkm_partition_name, "SYSTEM_DLKM"},
		targetFilesZipCopy{a.partitionProps.Vendor_dlkm_partition_name, "VENDOR_DLKM"},
		targetFilesZipCopy{a.partitionProps.Odm_dlkm_partition_name, "ODM_DLKM"},
		targetFilesZipCopy{a.partitionProps.Init_boot_partition_name, "BOOT/RAMDISK"},
		targetFilesZipCopy{a.partitionProps.Init_boot_partition_name, "INIT_BOOT/RAMDISK"},
		targetFilesZipCopy{a.partitionProps.Vendor_boot_partition_name, "VENDOR_BOOT/RAMDISK"},
	}
	// TODO: Handle cases where recovery files are copied to BOOT/ or RECOVERY/
	// https://cs.android.com/android/platform/superproject/main/+/main:build/make/core/Makefile;l=6211-6219?q=core%2FMakefile&ss=android%2Fplatform%2Fsuperproject%2Fmain
	if ctx.DeviceConfig().BoardMoveRecoveryResourcesToVendorBoot() {
		toCopy = append(toCopy, targetFilesZipCopy{a.partitionProps.Recovery_partition_name, "VENDOR_BOOT/RAMDISK"})
	}

	for _, zipCopy := range toCopy {
		if zipCopy.srcModule == nil {
			continue
		}
		fsInfo := a.getFilesystemInfo(ctx, *zipCopy.srcModule)
		subdir := zipCopy.destSubdir
		rootDirString := fsInfo.RootDir.String()
		if subdir == "SYSTEM" {
			rootDirString = rootDirString + "/system"
		}
		builder.Command().Textf("mkdir -p %s/%s", targetFilesDir.String(), subdir)
		builder.Command().
			BuiltTool("acp").
			Textf("-rd %s/. %s/%s", rootDirString, targetFilesDir, subdir).
			Implicit(fsInfo.Output) // so that the staging dir is built

	}
	// Copy cmdline files of boot images
	if a.partitionProps.Vendor_boot_partition_name != nil {
		bootImg := ctx.GetDirectDepWithTag(proptools.String(a.partitionProps.Vendor_boot_partition_name), filesystemDepTag)
		bootImgInfo, _ := android.OtherModuleProvider(ctx, bootImg, BootimgInfoProvider)
		builder.Command().Textf("echo %s > %s/%s/cmdline", proptools.ShellEscape(strings.Join(bootImgInfo.Cmdline, " ")), targetFilesDir, "VENDOR_BOOT")
		builder.Command().Textf("echo %s > %s/%s/vendor_cmdline", proptools.ShellEscape(strings.Join(bootImgInfo.Cmdline, " ")), targetFilesDir, "VENDOR_BOOT")
	}
	if a.partitionProps.Boot_partition_name != nil {
		bootImg := ctx.GetDirectDepWithTag(proptools.String(a.partitionProps.Boot_partition_name), filesystemDepTag)
		bootImgInfo, _ := android.OtherModuleProvider(ctx, bootImg, BootimgInfoProvider)
		builder.Command().Textf("echo %s > %s/%s/cmdline", proptools.ShellEscape(strings.Join(bootImgInfo.Cmdline, " ")), targetFilesDir, "BOOT")
	}

	builder.Command().
		BuiltTool("soong_zip").
		Text("-d").
		FlagWithOutput("-o ", targetFilesZip).
		FlagWithArg("-C ", targetFilesDir.String()).
		FlagWithArg("-D ", targetFilesDir.String()).
		Text("-sha256")
	builder.Build("target_files_"+ctx.ModuleName(), "Build target_files.zip")
}

func (a *androidDevice) getFilesystemInfo(ctx android.ModuleContext, depName string) FilesystemInfo {
	fsMod := ctx.GetDirectDepWithTag(depName, filesystemDepTag)
	fsInfo, ok := android.OtherModuleProvider(ctx, fsMod, FilesystemProvider)
	if !ok {
		ctx.ModuleErrorf("Expected dependency %s to be a filesystem", depName)
	}
	return fsInfo
}
