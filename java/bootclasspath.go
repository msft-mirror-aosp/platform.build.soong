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

package java

import (
	"android/soong/android"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

// Contains code that is common to both platform_bootclasspath and bootclasspath_fragment.

// addDependencyOntoApexVariants adds dependencies onto the appropriate apex specific variants of
// the module as specified in the ApexVariantReference list.
func addDependencyOntoApexVariants(ctx android.BottomUpMutatorContext, propertyName string, refs []ApexVariantReference, tagType bootclasspathDependencyTagType) {
	for i, ref := range refs {
		apex := proptools.StringDefault(ref.Apex, "platform")

		if ref.Module == nil {
			ctx.PropertyErrorf(propertyName, "missing module name at position %d", i)
			continue
		}
		name := proptools.String(ref.Module)

		addDependencyOntoApexModulePair(ctx, apex, name, tagType)
	}
}

// addDependencyOntoApexModulePair adds a dependency onto the specified APEX specific variant or the
// specified module.
//
// If apex="platform" or "system_ext" then this adds a dependency onto the platform variant of the
// module. This adds dependencies onto the prebuilt and source modules with the specified name,
// depending on which ones are available. Visiting must use isActiveModule to select the preferred
// module when both source and prebuilt modules are available.
//
// Use gatherApexModulePairDepsWithTag to retrieve the dependencies.
func addDependencyOntoApexModulePair(ctx android.BottomUpMutatorContext, apex string, name string, tagType bootclasspathDependencyTagType) {
	tag := bootclasspathDependencyTag{
		typ: tagType,
	}
	if !android.IsConfiguredJarForPlatform(apex) {
		tag.apex = apex
	}

	target := ctx.Module().Target()

	ctx.AddFarVariationDependencies(target.Variations(), tag, name)
}

// gatherApexModulePairDepsWithTag returns the list of dependencies with the supplied tag that was
// added by addDependencyOntoApexModulePair.
func gatherApexModulePairDepsWithTag(ctx android.BaseModuleContext, tagType bootclasspathDependencyTagType) []android.Module {
	var modules []android.Module
	ctx.VisitDirectDeps(func(module android.Module) {
		t := ctx.OtherModuleDependencyTag(module)
		if bcpTag, ok := t.(bootclasspathDependencyTag); ok && bcpTag.typ == tagType {
			modules = append(modules, module)
		}
	})
	return modules
}

// ApexVariantReference specifies a particular apex variant of a module.
type ApexVariantReference struct {
	android.BpPrintableBase

	// The name of the module apex variant, i.e. the apex containing the module variant.
	//
	// If this is not specified then it defaults to "platform" which will cause a dependency to be
	// added to the module's platform variant.
	//
	// A value of system_ext should be used for any module that will be part of the system_ext
	// partition.
	Apex *string

	// The name of the module.
	Module *string
}

// BootclasspathFragmentsDepsProperties contains properties related to dependencies onto fragments.
type BootclasspathFragmentsDepsProperties struct {
	// The names of the bootclasspath_fragment modules that form part of this module.
	Fragments []ApexVariantReference
}

// addDependenciesOntoFragments adds dependencies to the fragments specified in this properties
// structure.
func (p *BootclasspathFragmentsDepsProperties) addDependenciesOntoFragments(ctx android.BottomUpMutatorContext) {
	addDependencyOntoApexVariants(ctx, "fragments", p.Fragments, fragment)
}

// bootclasspathDependencyTag defines dependencies from/to bootclasspath_fragment,
// prebuilt_bootclasspath_fragment and platform_bootclasspath onto either source or prebuilt
// modules.
type bootclasspathDependencyTag struct {
	blueprint.BaseDependencyTag

	typ bootclasspathDependencyTagType

	apex string
}

type bootclasspathDependencyTagType int

const (
	// The tag used for dependencies onto bootclasspath_fragments.
	fragment bootclasspathDependencyTagType = iota
	// The tag used for dependencies onto platform_bootclasspath.
	platform
	dexpreoptBootJar
	artBootJar
	platformBootJar
	apexBootJar
)

func (t bootclasspathDependencyTag) ExcludeFromVisibilityEnforcement() {
}

func (t bootclasspathDependencyTag) ApexTransition() string {
	return t.apex
}

// Dependencies that use the bootclasspathDependencyTag instances are only added after all the
// visibility checking has been done so this has no functional effect. However, it does make it
// clear that visibility is not being enforced on these tags.
var _ android.ExcludeFromVisibilityEnforcementTag = bootclasspathDependencyTag{}

// BootclasspathNestedAPIProperties defines properties related to the API provided by parts of the
// bootclasspath that are nested within the main BootclasspathAPIProperties.
type BootclasspathNestedAPIProperties struct {
	// java_library or preferably, java_sdk_library modules providing stub classes that define the
	// APIs provided by this bootclasspath_fragment.
	Stub_libs proptools.Configurable[[]string]
}

// BootclasspathAPIProperties defines properties for defining the API provided by parts of the
// bootclasspath.
type BootclasspathAPIProperties struct {
	// Api properties provide information about the APIs provided by the bootclasspath_fragment.
	// Properties in this section apply to public, system and test api scopes. They DO NOT apply to
	// core_platform as that is a special, ART specific scope, that does not follow the pattern and so
	// has its own section. It is in the process of being deprecated and replaced by the system scope
	// but this will remain for the foreseeable future to maintain backwards compatibility.
	//
	// Every bootclasspath_fragment must specify at least one stubs_lib in this section and must
	// specify stubs for all the APIs provided by its contents. Failure to do so will lead to those
	// methods being inaccessible to other parts of Android, including but not limited to
	// applications.
	Api BootclasspathNestedAPIProperties

	// Properties related to the core platform API surface.
	//
	// This must only be used by the following modules:
	// * ART
	// * Conscrypt
	// * I18N
	//
	// The bootclasspath_fragments for each of the above modules must specify at least one stubs_lib
	// and must specify stubs for all the APIs provided by its contents. Failure to do so will lead to
	// those methods being inaccessible to the other modules in the list.
	Core_platform_api BootclasspathNestedAPIProperties
}

// apiScopeToStubLibs calculates the stub library modules for each relevant *HiddenAPIScope from the
// Stub_libs properties.
func (p BootclasspathAPIProperties) apiScopeToStubLibs(ctx android.BaseModuleContext) map[*HiddenAPIScope][]string {
	m := map[*HiddenAPIScope][]string{}
	for _, apiScope := range hiddenAPISdkLibrarySupportedScopes {
		m[apiScope] = p.Api.Stub_libs.GetOrDefault(ctx, nil)
	}
	m[CorePlatformHiddenAPIScope] = p.Core_platform_api.Stub_libs.GetOrDefault(ctx, nil)
	return m
}
