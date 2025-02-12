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
	"strings"
	"sync/atomic"

	"android/soong/android"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

type PartitionNameProperties struct {
	// Name of the super partition filesystem module
	Super_partition_name *string
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

type DeviceProperties struct {
	// Path to the prebuilt bootloader that would be copied to PRODUCT_OUT
	Bootloader *string `android:"path"`
	// Path to android-info.txt file containing board specific info.
	Android_info *string `android:"path"`
	// If this is the "main" android_device target for the build, i.e. the one that gets built
	// when running a plain `m` command. Currently, this is the autogenerated android_device module
	// in soong-only builds, but in the future when we check in android_device modules, the main
	// one will be determined based on the lunch product. TODO: Figure out how to make this
	// blueprint:"mutated" and still set it from filesystem_creator
	Main_device *bool

	Ab_ota_updater *bool
}

type androidDevice struct {
	android.ModuleBase

	partitionProps PartitionNameProperties

	deviceProps DeviceProperties

	allImagesZip android.Path
}

func AndroidDeviceFactory() android.Module {
	module := &androidDevice{}
	module.AddProperties(&module.partitionProps, &module.deviceProps)
	android.InitAndroidMultiTargetsArchModule(module, android.DeviceSupported, android.MultilibFirst)
	return module
}

var numMainAndroidDevicesOnceKey android.OnceKey = android.NewOnceKey("num_auto_generated_anroid_devices")

type partitionDepTagType struct {
	blueprint.BaseDependencyTag
}

type superPartitionDepTagType struct {
	blueprint.BaseDependencyTag
}
type targetFilesMetadataDepTagType struct {
	blueprint.BaseDependencyTag
}

var superPartitionDepTag superPartitionDepTagType
var filesystemDepTag partitionDepTagType
var targetFilesMetadataDepTag targetFilesMetadataDepTagType

func (a *androidDevice) DepsMutator(ctx android.BottomUpMutatorContext) {
	addDependencyIfDefined := func(dep *string) {
		if dep != nil {
			ctx.AddDependency(ctx.Module(), filesystemDepTag, proptools.String(dep))
		}
	}

	if a.partitionProps.Super_partition_name != nil {
		ctx.AddDependency(ctx.Module(), superPartitionDepTag, *a.partitionProps.Super_partition_name)
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
	a.addDepsForTargetFilesMetadata(ctx)
}

func (a *androidDevice) addDepsForTargetFilesMetadata(ctx android.BottomUpMutatorContext) {
	ctx.AddFarVariationDependencies(ctx.Config().BuildOSTarget.Variations(), targetFilesMetadataDepTag, "liblz4") // host variant
}

func (a *androidDevice) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	if proptools.Bool(a.deviceProps.Main_device) {
		numMainAndroidDevices := ctx.Config().Once(numMainAndroidDevicesOnceKey, func() interface{} {
			return &atomic.Int32{}
		}).(*atomic.Int32)
		total := numMainAndroidDevices.Add(1)
		if total > 1 {
			// There should only be 1 main android_device module. That one will be
			// made the default thing to build in soong-only builds.
			ctx.ModuleErrorf("There cannot be more than 1 main android_device module")
		}
	}

	a.buildTargetFilesZip(ctx)
	var deps []android.Path
	if proptools.String(a.partitionProps.Super_partition_name) != "" {
		superImage := ctx.GetDirectDepProxyWithTag(*a.partitionProps.Super_partition_name, superPartitionDepTag)
		if info, ok := android.OtherModuleProvider(ctx, superImage, SuperImageProvider); ok {
			assertUnset := func(prop *string, propName string) {
				if prop != nil && *prop != "" {
					ctx.PropertyErrorf(propName, "Cannot be set because it's already part of the super image")
				}
			}
			for _, subPartitionType := range android.SortedKeys(info.SubImageInfo) {
				switch subPartitionType {
				case "system":
					assertUnset(a.partitionProps.System_partition_name, "system_partition_name")
				case "system_ext":
					assertUnset(a.partitionProps.System_ext_partition_name, "system_ext_partition_name")
				case "system_dlkm":
					assertUnset(a.partitionProps.System_dlkm_partition_name, "system_dlkm_partition_name")
				case "system_other":
					// TODO
				case "product":
					assertUnset(a.partitionProps.Product_partition_name, "product_partition_name")
				case "vendor":
					assertUnset(a.partitionProps.Vendor_partition_name, "vendor_partition_name")
				case "vendor_dlkm":
					assertUnset(a.partitionProps.Vendor_dlkm_partition_name, "vendor_dlkm_partition_name")
				case "odm":
					assertUnset(a.partitionProps.Odm_partition_name, "odm_partition_name")
				case "odm_dlkm":
					assertUnset(a.partitionProps.Odm_dlkm_partition_name, "odm_dlkm_partition_name")
				default:
					ctx.ModuleErrorf("Unsupported sub-partition of super partition: %q", subPartitionType)
				}
			}

			deps = append(deps, info.SuperImage)
		} else {
			ctx.ModuleErrorf("Expected super image dep to provide SuperImageProvider")
		}
	}
	ctx.VisitDirectDepsProxyWithTag(filesystemDepTag, func(m android.ModuleProxy) {
		imageOutput, ok := android.OtherModuleProvider(ctx, m, android.OutputFilesProvider)
		if !ok {
			ctx.ModuleErrorf("Partition module %s doesn't set OutputfilesProvider", m.Name())
		}
		if len(imageOutput.DefaultOutputFiles) != 1 {
			ctx.ModuleErrorf("Partition module %s should provide exact 1 output file", m.Name())
		}
		deps = append(deps, imageOutput.DefaultOutputFiles[0])
	})

	allImagesZip := android.PathForModuleOut(ctx, "all_images.zip")
	allImagesZipBuilder := android.NewRuleBuilder(pctx, ctx)
	cmd := allImagesZipBuilder.Command().BuiltTool("soong_zip").Flag("--sort_entries")
	for _, dep := range deps {
		cmd.FlagWithArg("-e ", dep.Base())
		cmd.FlagWithInput("-f ", dep)
	}
	cmd.FlagWithOutput("-o ", allImagesZip)
	allImagesZipBuilder.Build("soong_all_images_zip", "all_images.zip")
	a.allImagesZip = allImagesZip

	allImagesStamp := android.PathForModuleOut(ctx, "all_images_stamp")
	var validations android.Paths
	if !ctx.Config().KatiEnabled() && proptools.Bool(a.deviceProps.Main_device) {
		// In soong-only builds, build this module by default.
		// This is the analogue to this make code:
		// https://cs.android.com/android/platform/superproject/main/+/main:build/make/core/main.mk;l=1396;drc=6595459cdd8164a6008335f6372c9f97b9094060
		ctx.Phony("droidcore-unbundled", allImagesStamp)

		deps = append(deps, a.copyFilesToProductOutForSoongOnly(ctx))
	}

	ctx.Build(pctx, android.BuildParams{
		Rule:        android.Touch,
		Output:      allImagesStamp,
		Implicits:   deps,
		Validations: validations,
	})

	// Checkbuilding it causes soong to make a phony, so you can say `m <module name>`
	ctx.CheckbuildFile(allImagesStamp)

	a.setVbmetaPhonyTargets(ctx)

	a.distFiles(ctx)
}

func (a *androidDevice) distFiles(ctx android.ModuleContext) {
	if !ctx.Config().KatiEnabled() {
		if proptools.Bool(a.deviceProps.Main_device) {
			fsInfoMap := a.getFsInfos(ctx)
			for _, partition := range android.SortedKeys(fsInfoMap) {
				fsInfo := fsInfoMap[partition]
				if fsInfo.InstalledFiles.Json != nil {
					ctx.DistForGoal("droidcore-unbundled", fsInfo.InstalledFiles.Json)
				}
				if fsInfo.InstalledFiles.Txt != nil {
					ctx.DistForGoal("droidcore-unbundled", fsInfo.InstalledFiles.Txt)
				}
			}
		}
	}

}

func (a *androidDevice) MakeVars(ctx android.MakeVarsModuleContext) {
	if proptools.Bool(a.deviceProps.Main_device) {
		ctx.StrictRaw("SOONG_ONLY_ALL_IMAGES_ZIP", a.allImagesZip.String())
	}
}

// Helper structs for target_files.zip creation
type targetFilesZipCopy struct {
	srcModule  *string
	destSubdir string
}

type targetFilesystemZipCopy struct {
	fsInfo     FilesystemInfo
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

	filesystemsToCopy := []targetFilesystemZipCopy{}
	for _, zipCopy := range toCopy {
		if zipCopy.srcModule == nil {
			continue
		}
		filesystemsToCopy = append(
			filesystemsToCopy,
			targetFilesystemZipCopy{a.getFilesystemInfo(ctx, *zipCopy.srcModule), zipCopy.destSubdir},
		)
	}
	// Get additional filesystems from super_partition dependency
	if a.partitionProps.Super_partition_name != nil {
		superPartition := ctx.GetDirectDepProxyWithTag(*a.partitionProps.Super_partition_name, superPartitionDepTag)
		if info, ok := android.OtherModuleProvider(ctx, superPartition, SuperImageProvider); ok {
			for _, partition := range android.SortedStringKeys(info.SubImageInfo) {
				filesystemsToCopy = append(
					filesystemsToCopy,
					targetFilesystemZipCopy{info.SubImageInfo[partition], strings.ToUpper(partition)},
				)
			}
		} else {
			ctx.ModuleErrorf("Super partition %s does set SuperImageProvider\n", superPartition.Name())
		}
	}

	for _, toCopy := range filesystemsToCopy {
		rootDirString := toCopy.fsInfo.RootDir.String()
		if toCopy.destSubdir == "SYSTEM" {
			rootDirString = rootDirString + "/system"
		}
		builder.Command().Textf("mkdir -p %s/%s", targetFilesDir.String(), toCopy.destSubdir)
		builder.Command().
			BuiltTool("acp").
			Textf("-rd %s/. %s/%s", rootDirString, targetFilesDir, toCopy.destSubdir).
			Implicit(toCopy.fsInfo.Output) // so that the staging dir is built

		if toCopy.destSubdir == "SYSTEM" {
			// Create the ROOT partition in target_files.zip
			builder.Command().Textf("rsync --links --exclude=system/* %s/ -r %s/ROOT", toCopy.fsInfo.RootDir, targetFilesDir.String())
		}
	}
	// Copy cmdline, kernel etc. files of boot images
	if a.partitionProps.Vendor_boot_partition_name != nil {
		bootImg := ctx.GetDirectDepProxyWithTag(proptools.String(a.partitionProps.Vendor_boot_partition_name), filesystemDepTag)
		bootImgInfo, _ := android.OtherModuleProvider(ctx, bootImg, BootimgInfoProvider)
		builder.Command().Textf("echo %s > %s/VENDOR_BOOT/cmdline", proptools.ShellEscape(strings.Join(bootImgInfo.Cmdline, " ")), targetFilesDir)
		builder.Command().Textf("echo %s > %s/VENDOR_BOOT/vendor_cmdline", proptools.ShellEscape(strings.Join(bootImgInfo.Cmdline, " ")), targetFilesDir)
		if bootImgInfo.Dtb != nil {
			builder.Command().Textf("cp ").Input(bootImgInfo.Dtb).Textf(" %s/VENDOR_BOOT/dtb", targetFilesDir)
		}
		if bootImgInfo.Bootconfig != nil {
			builder.Command().Textf("cp ").Input(bootImgInfo.Bootconfig).Textf(" %s/VENDOR_BOOT/vendor_bootconfig", targetFilesDir)
		}
	}
	if a.partitionProps.Boot_partition_name != nil {
		bootImg := ctx.GetDirectDepProxyWithTag(proptools.String(a.partitionProps.Boot_partition_name), filesystemDepTag)
		bootImgInfo, _ := android.OtherModuleProvider(ctx, bootImg, BootimgInfoProvider)
		builder.Command().Textf("echo %s > %s/BOOT/cmdline", proptools.ShellEscape(strings.Join(bootImgInfo.Cmdline, " ")), targetFilesDir)
		if bootImgInfo.Dtb != nil {
			builder.Command().Textf("cp ").Input(bootImgInfo.Dtb).Textf(" %s/BOOT/dtb", targetFilesDir)
		}
		if bootImgInfo.Kernel != nil {
			builder.Command().Textf("cp ").Input(bootImgInfo.Kernel).Textf(" %s/BOOT/kernel", targetFilesDir)
			// Even though kernel is not used to build vendor_boot, copy the kernel to VENDOR_BOOT to match the behavior of make packaging.
			builder.Command().Textf("cp ").Input(bootImgInfo.Kernel).Textf(" %s/VENDOR_BOOT/kernel", targetFilesDir)
		}
		if bootImgInfo.Bootconfig != nil {
			builder.Command().Textf("cp ").Input(bootImgInfo.Bootconfig).Textf(" %s/BOOT/bootconfig", targetFilesDir)
		}
	}

	if a.deviceProps.Android_info != nil {
		builder.Command().Textf("mkdir -p %s/OTA", targetFilesDir)
		builder.Command().Textf("cp ").Input(android.PathForModuleSrc(ctx, *a.deviceProps.Android_info)).Textf(" %s/OTA/android-info.txt", targetFilesDir)
	}

	a.copyImagesToTargetZip(ctx, builder, targetFilesDir)
	a.copyMetadataToTargetZip(ctx, builder, targetFilesDir)

	builder.Command().
		BuiltTool("soong_zip").
		Text("-d").
		FlagWithOutput("-o ", targetFilesZip).
		FlagWithArg("-C ", targetFilesDir.String()).
		FlagWithArg("-D ", targetFilesDir.String()).
		Text("-sha256")
	builder.Build("target_files_"+ctx.ModuleName(), "Build target_files.zip")
}

func (a *androidDevice) copyImagesToTargetZip(ctx android.ModuleContext, builder *android.RuleBuilder, targetFilesDir android.WritablePath) {
	// Create an IMAGES/ subdirectory
	builder.Command().Textf("mkdir -p %s/IMAGES", targetFilesDir.String())
	if a.deviceProps.Bootloader != nil {
		builder.Command().Textf("cp ").Input(android.PathForModuleSrc(ctx, proptools.String(a.deviceProps.Bootloader))).Textf(" %s/IMAGES/bootloader", targetFilesDir.String())
	}
	// Copy the filesystem ,boot and vbmeta img files to IMAGES/
	ctx.VisitDirectDepsProxyWithTag(filesystemDepTag, func(child android.ModuleProxy) {
		if strings.Contains(child.Name(), "recovery") {
			return // skip recovery.img to match the make packaging behavior
		}
		if info, ok := android.OtherModuleProvider(ctx, child, BootimgInfoProvider); ok {
			// Check Boot img first so that the boot.img is copied and not its dep ramdisk.img
			builder.Command().Textf("cp ").Input(info.Output).Textf(" %s/IMAGES/", targetFilesDir.String())
		} else if info, ok := android.OtherModuleProvider(ctx, child, FilesystemProvider); ok {
			builder.Command().Textf("cp ").Input(info.Output).Textf(" %s/IMAGES/", targetFilesDir.String())
		} else if info, ok := android.OtherModuleProvider(ctx, child, vbmetaPartitionProvider); ok {
			builder.Command().Textf("cp ").Input(info.Output).Textf(" %s/IMAGES/", targetFilesDir.String())
		} else {
			ctx.ModuleErrorf("Module %s does not provide an .img file output for target_files.zip", child.Name())
		}
	})

	if a.partitionProps.Super_partition_name != nil {
		superPartition := ctx.GetDirectDepProxyWithTag(*a.partitionProps.Super_partition_name, superPartitionDepTag)
		if info, ok := android.OtherModuleProvider(ctx, superPartition, SuperImageProvider); ok {
			for _, partition := range android.SortedKeys(info.SubImageInfo) {
				if info.SubImageInfo[partition].OutputHermetic != nil {
					builder.Command().Textf("cp ").Input(info.SubImageInfo[partition].OutputHermetic).Textf(" %s/IMAGES/", targetFilesDir.String())
				}
				if info.SubImageInfo[partition].MapFile != nil {
					builder.Command().Textf("cp ").Input(info.SubImageInfo[partition].MapFile).Textf(" %s/IMAGES/", targetFilesDir.String())
				}
			}
		} else {
			ctx.ModuleErrorf("Super partition %s does set SuperImageProvider\n", superPartition.Name())
		}
	}
}

func (a *androidDevice) copyMetadataToTargetZip(ctx android.ModuleContext, builder *android.RuleBuilder, targetFilesDir android.WritablePath) {
	// Create a META/ subdirectory
	builder.Command().Textf("mkdir -p %s/META", targetFilesDir.String())
	if proptools.Bool(a.deviceProps.Ab_ota_updater) {
		ctx.VisitDirectDepsProxyWithTag(targetFilesMetadataDepTag, func(child android.ModuleProxy) {
			info, _ := android.OtherModuleProvider(ctx, child, android.OutputFilesProvider)
			builder.Command().Textf("cp").Inputs(info.DefaultOutputFiles).Textf(" %s/META/", targetFilesDir.String())
		})
	}
	builder.Command().Textf("cp").Input(android.PathForSource(ctx, "external/zucchini/version_info.h")).Textf(" %s/META/zucchini_config.txt", targetFilesDir.String())
	builder.Command().Textf("cp").Input(android.PathForSource(ctx, "system/update_engine/update_engine.conf")).Textf(" %s/META/update_engine_config.txt", targetFilesDir.String())
}

func (a *androidDevice) getFilesystemInfo(ctx android.ModuleContext, depName string) FilesystemInfo {
	fsMod := ctx.GetDirectDepProxyWithTag(depName, filesystemDepTag)
	fsInfo, ok := android.OtherModuleProvider(ctx, fsMod, FilesystemProvider)
	if !ok {
		ctx.ModuleErrorf("Expected dependency %s to be a filesystem", depName)
	}
	return fsInfo
}

func (a *androidDevice) setVbmetaPhonyTargets(ctx android.ModuleContext) {
	if !proptools.Bool(a.deviceProps.Main_device) {
		return
	}

	if !ctx.Config().KatiEnabled() {
		for _, vbmetaPartitionName := range a.partitionProps.Vbmeta_partitions {
			img := ctx.GetDirectDepProxyWithTag(vbmetaPartitionName, filesystemDepTag)
			if provider, ok := android.OtherModuleProvider(ctx, img, vbmetaPartitionProvider); ok {
				// make generates `vbmetasystemimage` phony target instead of `vbmeta_systemimage` phony target.
				partitionName := strings.ReplaceAll(provider.Name, "_", "")
				ctx.Phony(fmt.Sprintf("%simage", partitionName), provider.Output)
			}
		}
	}
}
