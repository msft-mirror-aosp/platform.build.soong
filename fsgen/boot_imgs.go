package fsgen

import (
	"android/soong/android"
	"android/soong/filesystem"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/google/blueprint/proptools"
)

func createBootImage(ctx android.LoadHookContext) bool {
	partitionVariables := ctx.Config().ProductVariables().PartitionVarsForSoongMigrationOnlyDoNotUse

	if partitionVariables.TargetKernelPath == "" {
		// There are potentially code paths that don't set TARGET_KERNEL_PATH
		return false
	}

	kernelDir := filepath.Dir(partitionVariables.TargetKernelPath)
	kernelBase := filepath.Base(partitionVariables.TargetKernelPath)
	kernelFilegroupName := generatedModuleName(ctx.Config(), "kernel")

	ctx.CreateModuleInDirectory(
		android.FileGroupFactory,
		kernelDir,
		&struct {
			Name       *string
			Srcs       []string
			Visibility []string
		}{
			Name:       proptools.StringPtr(kernelFilegroupName),
			Srcs:       []string{kernelBase},
			Visibility: []string{"//visibility:public"},
		},
	)

	bootImageName := generatedModuleNameForPartition(ctx.Config(), "boot")

	ctx.CreateModule(
		filesystem.BootimgFactory,
		&filesystem.BootimgProperties{
			Kernel_prebuilt: proptools.StringPtr(":" + kernelFilegroupName),
			Header_version:  proptools.StringPtr(partitionVariables.BoardBootHeaderVersion),
		},
		&struct {
			Name *string
		}{
			Name: proptools.StringPtr(bootImageName),
		},
	)
	return true
}

func createVendorBootImage(ctx android.LoadHookContext) bool {
	partitionVariables := ctx.Config().ProductVariables().PartitionVarsForSoongMigrationOnlyDoNotUse

	bootImageName := generatedModuleNameForPartition(ctx.Config(), "vendor_boot")

	ctx.CreateModule(
		filesystem.BootimgFactory,
		&filesystem.BootimgProperties{
			Boot_image_type: proptools.StringPtr("vendor_boot"),
			Ramdisk_module:  proptools.StringPtr(generatedModuleNameForPartition(ctx.Config(), "vendor_ramdisk")),
			Header_version:  proptools.StringPtr(partitionVariables.BoardBootHeaderVersion),
		},
		&struct {
			Name *string
		}{
			Name: proptools.StringPtr(bootImageName),
		},
	)
	return true
}

func createInitBootImage(ctx android.LoadHookContext) bool {
	partitionVariables := ctx.Config().ProductVariables().PartitionVarsForSoongMigrationOnlyDoNotUse

	bootImageName := generatedModuleNameForPartition(ctx.Config(), "init_boot")

	ctx.CreateModule(
		filesystem.BootimgFactory,
		&filesystem.BootimgProperties{
			Boot_image_type: proptools.StringPtr("init_boot"),
			Ramdisk_module:  proptools.StringPtr(generatedModuleNameForPartition(ctx.Config(), "ramdisk")),
			Header_version:  proptools.StringPtr(partitionVariables.BoardBootHeaderVersion),
		},
		&struct {
			Name *string
		}{
			Name: proptools.StringPtr(bootImageName),
		},
	)
	return true
}

// Returns the equivalent of the BUILDING_BOOT_IMAGE variable in make. Derived from this logic:
// https://cs.android.com/android/platform/superproject/main/+/main:build/make/core/board_config.mk;l=458;drc=5b55f926830963c02ab1d2d91e46442f04ba3af0
func buildingBootImage(partitionVars android.PartitionVariables) bool {
	if partitionVars.BoardUsesRecoveryAsBoot {
		return false
	}

	if partitionVars.ProductBuildBootImage {
		return true
	}

	if len(partitionVars.BoardPrebuiltBootimage) > 0 {
		return false
	}

	if len(partitionVars.BoardBootimagePartitionSize) > 0 {
		return true
	}

	// TODO: return true if BOARD_KERNEL_BINARIES is set and has a *_BOOTIMAGE_PARTITION_SIZE
	// variable. However, I don't think BOARD_KERNEL_BINARIES is ever set in practice.

	return false
}

// Returns the equivalent of the BUILDING_VENDOR_BOOT_IMAGE variable in make. Derived from this logic:
// https://cs.android.com/android/platform/superproject/main/+/main:build/make/core/board_config.mk;l=518;drc=5b55f926830963c02ab1d2d91e46442f04ba3af0
func buildingVendorBootImage(partitionVars android.PartitionVariables) bool {
	if v, exists := boardBootHeaderVersion(partitionVars); exists && v >= 3 {
		x := partitionVars.ProductBuildVendorBootImage
		if x == "" || x == "true" {
			return true
		}
	}

	return false
}

// Derived from: https://cs.android.com/android/platform/superproject/main/+/main:build/make/core/board_config.mk;l=480;drc=5b55f926830963c02ab1d2d91e46442f04ba3af0
func buildingInitBootImage(partitionVars android.PartitionVariables) bool {
	if !partitionVars.ProductBuildInitBootImage {
		if partitionVars.BoardUsesRecoveryAsBoot || len(partitionVars.BoardPrebuiltInitBootimage) > 0 {
			return false
		} else if len(partitionVars.BoardInitBootimagePartitionSize) > 0 {
			return true
		}
	} else {
		if partitionVars.BoardUsesRecoveryAsBoot {
			panic("PRODUCT_BUILD_INIT_BOOT_IMAGE is true, but so is BOARD_USES_RECOVERY_AS_BOOT. Use only one option.")
		}
		return true
	}
	return false
}

func boardBootHeaderVersion(partitionVars android.PartitionVariables) (int, bool) {
	if len(partitionVars.BoardBootHeaderVersion) == 0 {
		return 0, false
	}
	v, err := strconv.ParseInt(partitionVars.BoardBootHeaderVersion, 10, 32)
	if err != nil {
		panic(fmt.Sprintf("BOARD_BOOT_HEADER_VERSION must be an int, got: %q", partitionVars.BoardBootHeaderVersion))
	}
	return int(v), true
}
