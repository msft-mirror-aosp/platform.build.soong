// Copyright 2020 The Android Open Source Project
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package cc

// This file contains image variant related things, including image mutator functions, utility
// functions to determine where a module is installed, etc.

import (
	"strings"

	"android/soong/android"

	"github.com/google/blueprint/proptools"
)

var _ android.ImageInterface = (*Module)(nil)

type ImageVariantType string

const (
	coreImageVariant          ImageVariantType = "core"
	vendorImageVariant        ImageVariantType = "vendor"
	productImageVariant       ImageVariantType = "product"
	ramdiskImageVariant       ImageVariantType = "ramdisk"
	vendorRamdiskImageVariant ImageVariantType = "vendor_ramdisk"
	recoveryImageVariant      ImageVariantType = "recovery"
	hostImageVariant          ImageVariantType = "host"
)

const (
	// VendorVariationPrefix is the variant prefix used for /vendor code that compiles
	// against the VNDK.
	VendorVariationPrefix = "vendor."

	// ProductVariationPrefix is the variant prefix used for /product code that compiles
	// against the VNDK.
	ProductVariationPrefix = "product."
)

func (ctx *moduleContextImpl) inProduct() bool {
	return ctx.mod.InProduct()
}

func (ctx *moduleContextImpl) inVendor() bool {
	return ctx.mod.InVendor()
}

func (ctx *moduleContextImpl) inRamdisk() bool {
	return ctx.mod.InRamdisk()
}

func (ctx *moduleContextImpl) inVendorRamdisk() bool {
	return ctx.mod.InVendorRamdisk()
}

func (ctx *moduleContextImpl) inRecovery() bool {
	return ctx.mod.InRecovery()
}

func (c *Module) InstallInProduct() bool {
	// Additionally check if this module is inProduct() that means it is a "product" variant of a
	// module. As well as product specific modules, product variants must be installed to /product.
	return c.InProduct()
}

func (c *Module) InstallInVendor() bool {
	// Additionally check if this module is inVendor() that means it is a "vendor" variant of a
	// module. As well as SoC specific modules, vendor variants must be installed to /vendor
	// unless they have "odm_available: true".
	return c.HasVendorVariant() && c.InVendor() && !c.VendorVariantToOdm()
}

func (c *Module) InstallInOdm() bool {
	// Some vendor variants want to be installed to /odm by setting "odm_available: true".
	return c.InVendor() && c.VendorVariantToOdm()
}

// Returns true when this module is configured to have core and vendor variants.
func (c *Module) HasVendorVariant() bool {
	return Bool(c.VendorProperties.Vendor_available) || Bool(c.VendorProperties.Odm_available)
}

// Returns true when this module creates a vendor variant and wants to install the vendor variant
// to the odm partition.
func (c *Module) VendorVariantToOdm() bool {
	return Bool(c.VendorProperties.Odm_available)
}

// Returns true when this module is configured to have core and product variants.
func (c *Module) HasProductVariant() bool {
	return Bool(c.VendorProperties.Product_available)
}

// Returns true when this module is configured to have core and either product or vendor variants.
func (c *Module) HasNonSystemVariants() bool {
	return c.HasVendorVariant() || c.HasProductVariant()
}

// Returns true if the module is "product" variant. Usually these modules are installed in /product
func (c *Module) InProduct() bool {
	return c.Properties.ImageVariation == android.ProductVariation
}

// Returns true if the module is "vendor" variant. Usually these modules are installed in /vendor
func (c *Module) InVendor() bool {
	return c.Properties.ImageVariation == android.VendorVariation
}

// Returns true if the module is "vendor" or "product" variant. This replaces previous UseVndk usages
// which were misused to check if the module variant is vendor or product.
func (c *Module) InVendorOrProduct() bool {
	return c.InVendor() || c.InProduct()
}

func (c *Module) InRamdisk() bool {
	return c.ModuleBase.InRamdisk() || c.ModuleBase.InstallInRamdisk()
}

func (c *Module) InVendorRamdisk() bool {
	return c.ModuleBase.InVendorRamdisk() || c.ModuleBase.InstallInVendorRamdisk()
}

func (c *Module) InRecovery() bool {
	return c.ModuleBase.InRecovery() || c.ModuleBase.InstallInRecovery()
}

func (c *Module) OnlyInRamdisk() bool {
	return c.ModuleBase.InstallInRamdisk()
}

func (c *Module) OnlyInVendorRamdisk() bool {
	return c.ModuleBase.InstallInVendorRamdisk()
}

func (c *Module) OnlyInRecovery() bool {
	return c.ModuleBase.InstallInRecovery()
}

// ImageMutatableModule provides a common image mutation interface for  LinkableInterface modules.
type ImageMutatableModule interface {
	android.Module
	LinkableInterface

	// AndroidModuleBase returns the android.ModuleBase for this module
	AndroidModuleBase() *android.ModuleBase

	// VendorAvailable returns true if this module is available on the vendor image.
	VendorAvailable() bool

	// OdmAvailable returns true if this module is available on the odm image.
	OdmAvailable() bool

	// ProductAvailable returns true if this module is available on the product image.
	ProductAvailable() bool

	// RamdiskAvailable returns true if this module is available on the ramdisk image.
	RamdiskAvailable() bool

	// RecoveryAvailable returns true if this module is available on the recovery image.
	RecoveryAvailable() bool

	// VendorRamdiskAvailable returns true if this module is available on the vendor ramdisk image.
	VendorRamdiskAvailable() bool

	// IsSnapshotPrebuilt returns true if this module is a snapshot prebuilt.
	IsSnapshotPrebuilt() bool

	// SnapshotVersion returns the snapshot version for this module.
	SnapshotVersion(mctx android.BaseModuleContext) string

	// SdkVersion returns the SDK version for this module.
	SdkVersion() string

	// ExtraVariants returns the list of extra variants this module requires.
	ExtraVariants() []string

	// AppendExtraVariant returns an extra variant to the list of extra variants this module requires.
	AppendExtraVariant(extraVariant string)

	// SetRamdiskVariantNeeded sets whether the Ramdisk Variant is needed.
	SetRamdiskVariantNeeded(b bool)

	// SetVendorRamdiskVariantNeeded sets whether the Vendor Ramdisk Variant is needed.
	SetVendorRamdiskVariantNeeded(b bool)

	// SetRecoveryVariantNeeded sets whether the Recovery Variant is needed.
	SetRecoveryVariantNeeded(b bool)

	// SetCoreVariantNeeded sets whether the Core Variant is needed.
	SetCoreVariantNeeded(b bool)

	// SetProductVariantNeeded sets whether the Product Variant is needed.
	SetProductVariantNeeded(b bool)

	// SetVendorVariantNeeded sets whether the Vendor Variant is needed.
	SetVendorVariantNeeded(b bool)
}

var _ ImageMutatableModule = (*Module)(nil)

func (m *Module) ImageMutatorBegin(mctx android.BaseModuleContext) {
	MutateImage(mctx, m)
}

func (m *Module) VendorAvailable() bool {
	return Bool(m.VendorProperties.Vendor_available)
}

func (m *Module) OdmAvailable() bool {
	return Bool(m.VendorProperties.Odm_available)
}

func (m *Module) ProductAvailable() bool {
	return Bool(m.VendorProperties.Product_available)
}

func (m *Module) RamdiskAvailable() bool {
	return Bool(m.Properties.Ramdisk_available)
}

func (m *Module) VendorRamdiskAvailable() bool {
	return Bool(m.Properties.Vendor_ramdisk_available)
}

func (m *Module) AndroidModuleBase() *android.ModuleBase {
	return &m.ModuleBase
}

func (m *Module) RecoveryAvailable() bool {
	return Bool(m.Properties.Recovery_available)
}

func (m *Module) ExtraVariants() []string {
	return m.Properties.ExtraVersionedImageVariations
}

func (m *Module) AppendExtraVariant(extraVariant string) {
	m.Properties.ExtraVersionedImageVariations = append(m.Properties.ExtraVersionedImageVariations, extraVariant)
}

func (m *Module) SetRamdiskVariantNeeded(b bool) {
	m.Properties.RamdiskVariantNeeded = b
}

func (m *Module) SetVendorRamdiskVariantNeeded(b bool) {
	m.Properties.VendorRamdiskVariantNeeded = b
}

func (m *Module) SetRecoveryVariantNeeded(b bool) {
	m.Properties.RecoveryVariantNeeded = b
}

func (m *Module) SetCoreVariantNeeded(b bool) {
	m.Properties.CoreVariantNeeded = b
}

func (m *Module) SetProductVariantNeeded(b bool) {
	m.Properties.ProductVariantNeeded = b
}

func (m *Module) SetVendorVariantNeeded(b bool) {
	m.Properties.VendorVariantNeeded = b
}

func (m *Module) SnapshotVersion(mctx android.BaseModuleContext) string {
	if snapshot, ok := m.linker.(SnapshotInterface); ok {
		return snapshot.Version()
	} else {
		mctx.ModuleErrorf("version is unknown for snapshot prebuilt")
		// Should we be panicking here instead?
		return ""
	}
}

func (m *Module) KernelHeadersDecorator() bool {
	if _, ok := m.linker.(*kernelHeadersDecorator); ok {
		return true
	}
	return false
}

// MutateImage handles common image mutations for ImageMutatableModule interfaces.
func MutateImage(mctx android.BaseModuleContext, m ImageMutatableModule) {
	// Validation check
	vendorSpecific := mctx.SocSpecific() || mctx.DeviceSpecific()
	productSpecific := mctx.ProductSpecific()

	if m.VendorAvailable() {
		if vendorSpecific {
			mctx.PropertyErrorf("vendor_available",
				"doesn't make sense at the same time as `vendor: true`, `proprietary: true`, or `device_specific: true`")
		}
		if m.OdmAvailable() {
			mctx.PropertyErrorf("vendor_available",
				"doesn't make sense at the same time as `odm_available: true`")
		}
	}

	if m.OdmAvailable() {
		if vendorSpecific {
			mctx.PropertyErrorf("odm_available",
				"doesn't make sense at the same time as `vendor: true`, `proprietary: true`, or `device_specific: true`")
		}
	}

	if m.ProductAvailable() {
		if productSpecific {
			mctx.PropertyErrorf("product_available",
				"doesn't make sense at the same time as `product_specific: true`")
		}
		if vendorSpecific {
			mctx.PropertyErrorf("product_available",
				"cannot provide product variant from a vendor module. Please use `product_specific: true` with `vendor_available: true`")
		}
	}

	var vendorVariantNeeded bool = false
	var productVariantNeeded bool = false
	var coreVariantNeeded bool = false
	var ramdiskVariantNeeded bool = false
	var vendorRamdiskVariantNeeded bool = false
	var recoveryVariantNeeded bool = false

	if m.NeedsLlndkVariants() {
		// This is an LLNDK library.  The implementation of the library will be on /system,
		// and vendor and product variants will be created with LLNDK stubs.
		// The LLNDK libraries need vendor variants even if there is no VNDK.
		coreVariantNeeded = true
		vendorVariantNeeded = true
		productVariantNeeded = true

	} else if m.NeedsVendorPublicLibraryVariants() {
		// A vendor public library has the implementation on /vendor, with stub variants
		// for system and product.
		coreVariantNeeded = true
		vendorVariantNeeded = true
		productVariantNeeded = true
	} else if m.IsSnapshotPrebuilt() {
		// Make vendor variants only for the versions in BOARD_VNDK_VERSION and
		// PRODUCT_EXTRA_VNDK_VERSIONS.
		if m.InstallInRecovery() {
			recoveryVariantNeeded = true
		} else {
			m.AppendExtraVariant(VendorVariationPrefix + m.SnapshotVersion(mctx))
		}
	} else if m.HasNonSystemVariants() {
		// This will be available to /system unless it is product_specific
		// which will be handled later.
		coreVariantNeeded = true

		// We assume that modules under proprietary paths are compatible for
		// BOARD_VNDK_VERSION. The other modules are regarded as AOSP, or
		// PLATFORM_VNDK_VERSION.
		if m.HasVendorVariant() {
			vendorVariantNeeded = true
		}

		// product_available modules are available to /product.
		if m.HasProductVariant() {
			productVariantNeeded = true
		}
	} else if vendorSpecific && m.SdkVersion() == "" {
		// This will be available in /vendor (or /odm) only
		vendorVariantNeeded = true
	} else {
		// This is either in /system (or similar: /data), or is a
		// module built with the NDK. Modules built with the NDK
		// will be restricted using the existing link type checks.
		coreVariantNeeded = true
	}

	if coreVariantNeeded && productSpecific && m.SdkVersion() == "" {
		// The module has "product_specific: true" that does not create core variant.
		coreVariantNeeded = false
		productVariantNeeded = true
	}

	if m.RamdiskAvailable() {
		ramdiskVariantNeeded = true
	}

	if m.AndroidModuleBase().InstallInRamdisk() {
		ramdiskVariantNeeded = true
		coreVariantNeeded = false
	}

	if m.VendorRamdiskAvailable() {
		vendorRamdiskVariantNeeded = true
	}

	if m.AndroidModuleBase().InstallInVendorRamdisk() {
		vendorRamdiskVariantNeeded = true
		coreVariantNeeded = false
	}

	if m.RecoveryAvailable() {
		recoveryVariantNeeded = true
	}

	if m.AndroidModuleBase().InstallInRecovery() {
		recoveryVariantNeeded = true
		coreVariantNeeded = false
	}

	m.SetRamdiskVariantNeeded(ramdiskVariantNeeded)
	m.SetVendorRamdiskVariantNeeded(vendorRamdiskVariantNeeded)
	m.SetRecoveryVariantNeeded(recoveryVariantNeeded)
	m.SetCoreVariantNeeded(coreVariantNeeded)
	m.SetProductVariantNeeded(productVariantNeeded)
	m.SetVendorVariantNeeded(vendorVariantNeeded)

	// Disable the module if no variants are needed.
	if !ramdiskVariantNeeded &&
		!recoveryVariantNeeded &&
		!coreVariantNeeded &&
		!productVariantNeeded &&
		!vendorVariantNeeded &&
		len(m.ExtraVariants()) == 0 {
		m.Disable()
	}
}

func (c *Module) VendorVariantNeeded(ctx android.BaseModuleContext) bool {
	return c.Properties.VendorVariantNeeded
}

func (c *Module) ProductVariantNeeded(ctx android.BaseModuleContext) bool {
	return c.Properties.ProductVariantNeeded
}

func (c *Module) CoreVariantNeeded(ctx android.BaseModuleContext) bool {
	return c.Properties.CoreVariantNeeded
}

func (c *Module) RamdiskVariantNeeded(ctx android.BaseModuleContext) bool {
	return c.Properties.RamdiskVariantNeeded
}

func (c *Module) VendorRamdiskVariantNeeded(ctx android.BaseModuleContext) bool {
	return c.Properties.VendorRamdiskVariantNeeded
}

func (c *Module) DebugRamdiskVariantNeeded(ctx android.BaseModuleContext) bool {
	return false
}

func (c *Module) RecoveryVariantNeeded(ctx android.BaseModuleContext) bool {
	return c.Properties.RecoveryVariantNeeded
}

func (c *Module) ExtraImageVariations(ctx android.BaseModuleContext) []string {
	return c.Properties.ExtraVersionedImageVariations
}

func squashVendorSrcs(m *Module) {
	if lib, ok := m.compiler.(*libraryDecorator); ok {
		lib.baseCompiler.Properties.Srcs.AppendSimpleValue(lib.baseCompiler.Properties.Target.Vendor.Srcs)
		lib.baseCompiler.Properties.Exclude_srcs.AppendSimpleValue(lib.baseCompiler.Properties.Target.Vendor.Exclude_srcs)

		lib.baseCompiler.Properties.Exclude_generated_sources = append(lib.baseCompiler.Properties.Exclude_generated_sources,
			lib.baseCompiler.Properties.Target.Vendor.Exclude_generated_sources...)

		if lib.Properties.Target.Vendor.No_stubs {
			proptools.Clear(&lib.Properties.Stubs)
		}
	}
}

func squashProductSrcs(m *Module) {
	if lib, ok := m.compiler.(*libraryDecorator); ok {
		lib.baseCompiler.Properties.Srcs.AppendSimpleValue(lib.baseCompiler.Properties.Target.Product.Srcs)
		lib.baseCompiler.Properties.Exclude_srcs.AppendSimpleValue(lib.baseCompiler.Properties.Target.Product.Exclude_srcs)

		lib.baseCompiler.Properties.Exclude_generated_sources = append(lib.baseCompiler.Properties.Exclude_generated_sources,
			lib.baseCompiler.Properties.Target.Product.Exclude_generated_sources...)

		if lib.Properties.Target.Product.No_stubs {
			proptools.Clear(&lib.Properties.Stubs)
		}
	}
}

func squashRecoverySrcs(m *Module) {
	if lib, ok := m.compiler.(*libraryDecorator); ok {
		lib.baseCompiler.Properties.Srcs.AppendSimpleValue(lib.baseCompiler.Properties.Target.Recovery.Srcs)
		lib.baseCompiler.Properties.Exclude_srcs.AppendSimpleValue(lib.baseCompiler.Properties.Target.Recovery.Exclude_srcs)

		lib.baseCompiler.Properties.Exclude_generated_sources = append(lib.baseCompiler.Properties.Exclude_generated_sources,
			lib.baseCompiler.Properties.Target.Recovery.Exclude_generated_sources...)
	}
}

func squashVendorRamdiskSrcs(m *Module) {
	if lib, ok := m.compiler.(*libraryDecorator); ok {
		lib.baseCompiler.Properties.Exclude_srcs.AppendSimpleValue(lib.baseCompiler.Properties.Target.Vendor_ramdisk.Exclude_srcs)
	}
}

func squashRamdiskSrcs(m *Module) {
	if lib, ok := m.compiler.(*libraryDecorator); ok {
		lib.baseCompiler.Properties.Exclude_srcs.AppendSimpleValue(lib.baseCompiler.Properties.Target.Ramdisk.Exclude_srcs)
	}
}

func (c *Module) SetImageVariation(ctx android.BaseModuleContext, variant string) {
	if variant == android.RamdiskVariation {
		c.MakeAsPlatform()
		squashRamdiskSrcs(c)
	} else if variant == android.VendorRamdiskVariation {
		c.MakeAsPlatform()
		squashVendorRamdiskSrcs(c)
	} else if variant == android.RecoveryVariation {
		c.MakeAsPlatform()
		squashRecoverySrcs(c)
	} else if strings.HasPrefix(variant, android.VendorVariation) {
		c.Properties.ImageVariation = android.VendorVariation

		if strings.HasPrefix(variant, VendorVariationPrefix) {
			c.Properties.VndkVersion = strings.TrimPrefix(variant, VendorVariationPrefix)
		}
		squashVendorSrcs(c)
	} else if strings.HasPrefix(variant, android.ProductVariation) {
		c.Properties.ImageVariation = android.ProductVariation
		if strings.HasPrefix(variant, ProductVariationPrefix) {
			c.Properties.VndkVersion = strings.TrimPrefix(variant, ProductVariationPrefix)
		}
		squashProductSrcs(c)
	}

	if c.NeedsVendorPublicLibraryVariants() &&
		(variant == android.CoreVariation || strings.HasPrefix(variant, ProductVariationPrefix)) {
		c.VendorProperties.IsVendorPublicLibrary = true
	}
}
