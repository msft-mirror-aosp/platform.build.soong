// Copyright 2024 Google Inc. All rights reserved.
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

package android

import (
	"reflect"
	"slices"

	"github.com/google/blueprint"
)

type StubsAvailableModule interface {
	IsStubsModule() bool
}

// Returns true if the dependency module is a stubs module
var depIsStubsModule = func(_ ModuleContext, _, dep Module) bool {
	if stubsModule, ok := dep.(StubsAvailableModule); ok {
		return stubsModule.IsStubsModule()
	}
	return false
}

// Labels of exception functions, which are used to determine special dependencies that allow
// otherwise restricted inter-container dependencies
type exceptionHandleFuncLabel int

const (
	checkStubs exceptionHandleFuncLabel = iota
)

// Functions cannot be used as a value passed in providers, because functions are not
// hashable. As a workaround, the exceptionHandleFunc enum values are passed using providers,
// and the corresponding functions are called from this map.
var exceptionHandleFunctionsTable = map[exceptionHandleFuncLabel]func(ModuleContext, Module, Module) bool{
	checkStubs: depIsStubsModule,
}

type InstallableModule interface {
	EnforceApiContainerChecks() bool
}

type restriction struct {
	// container of the dependency
	dependency *container

	// Error message to be emitted to the user when the dependency meets this restriction
	errorMessage string

	// List of labels of allowed exception functions that allows bypassing this restriction.
	// If any of the functions mapped to each labels returns true, this dependency would be
	// considered allowed and an error will not be thrown.
	allowedExceptions []exceptionHandleFuncLabel
}
type container struct {
	// The name of the container i.e. partition, api domain
	name string

	// Map of dependency restricted containers.
	restricted []restriction
}

var (
	VendorContainer = &container{
		name:       VendorVariation,
		restricted: nil,
	}
	SystemContainer = &container{
		name: "system",
		restricted: []restriction{
			{
				dependency: VendorContainer,
				errorMessage: "Module belonging to the system partition other than HALs is " +
					"not allowed to depend on the vendor partition module, in order to support " +
					"independent development/update cycles and to support the Generic System " +
					"Image. Try depending on HALs, VNDK or AIDL instead.",
				allowedExceptions: []exceptionHandleFuncLabel{},
			},
		},
	}
	ProductContainer = &container{
		name: ProductVariation,
		restricted: []restriction{
			{
				dependency: VendorContainer,
				errorMessage: "Module belonging to the product partition is not allowed to " +
					"depend on the vendor partition module, as this may lead to security " +
					"vulnerabilities. Try depending on the HALs or utilize AIDL instead.",
				allowedExceptions: []exceptionHandleFuncLabel{},
			},
		},
	}
	ApexContainer = initializeApexContainer()
	CtsContainer  = &container{
		name: "cts",
		restricted: []restriction{
			{
				dependency: SystemContainer,
				errorMessage: "CTS module should not depend on the modules belonging to the " +
					"system partition, including \"framework\". Depending on the system " +
					"partition may lead to disclosure of implementation details and regression " +
					"due to API changes across platform versions. Try depending on the stubs instead.",
				allowedExceptions: []exceptionHandleFuncLabel{checkStubs},
			},
		},
	}
)

func initializeApexContainer() *container {
	apexContainer := &container{
		name: "apex",
		restricted: []restriction{
			{
				dependency: SystemContainer,
				errorMessage: "Module belonging to Apex(es) is not allowed to depend on the " +
					"modules belonging to the system partition. Either statically depend on the " +
					"module or convert the depending module to java_sdk_library and depend on " +
					"the stubs.",
				allowedExceptions: []exceptionHandleFuncLabel{checkStubs},
			},
		},
	}

	apexContainer.restricted = append(apexContainer.restricted, restriction{
		dependency: apexContainer,
		errorMessage: "Module belonging to Apex(es) is not allowed to depend on the " +
			"modules belonging to other Apex(es). Either include the depending " +
			"module in the Apex or convert the depending module to java_sdk_library " +
			"and depend on its stubs.",
		allowedExceptions: []exceptionHandleFuncLabel{checkStubs},
	})

	return apexContainer
}

type ContainersInfo struct {
	belongingContainers []*container

	belongingApexes []ApexInfo
}

func (c *ContainersInfo) BelongingContainers() []*container {
	return c.belongingContainers
}

var ContainersInfoProvider = blueprint.NewProvider[ContainersInfo]()

// Determines if the module can be installed in the system partition or not.
// Logic is identical to that of modulePartition(...) defined in paths.go
func installInSystemPartition(ctx ModuleContext) bool {
	module := ctx.Module()
	return !module.InstallInTestcases() &&
		!module.InstallInData() &&
		!module.InstallInRamdisk() &&
		!module.InstallInVendorRamdisk() &&
		!module.InstallInDebugRamdisk() &&
		!module.InstallInRecovery() &&
		!module.InstallInVendor() &&
		!module.InstallInOdm() &&
		!module.InstallInProduct() &&
		determineModuleKind(module.base(), ctx.blueprintBaseModuleContext()) == platformModule
}

func generateContainerInfo(ctx ModuleContext) ContainersInfo {
	inSystem := installInSystemPartition(ctx)
	inProduct := ctx.Module().InstallInProduct()
	inVendor := ctx.Module().InstallInVendor()
	inCts := false
	inApex := false

	if m, ok := ctx.Module().(ImageInterface); ok {
		inProduct = inProduct || m.ProductVariantNeeded(ctx)
		inVendor = inVendor || m.VendorVariantNeeded(ctx)
	}

	props := ctx.Module().GetProperties()
	for _, prop := range props {
		val := reflect.ValueOf(prop).Elem()
		if val.Kind() == reflect.Struct {
			testSuites := val.FieldByName("Test_suites")
			if testSuites.IsValid() && testSuites.Kind() == reflect.Slice && slices.Contains(testSuites.Interface().([]string), "cts") {
				inCts = true
			}
		}
	}

	var belongingApexes []ApexInfo
	if apexInfo, ok := ModuleProvider(ctx, AllApexInfoProvider); ok {
		belongingApexes = apexInfo.ApexInfos
		inApex = true
	}

	containers := []*container{}
	if inSystem {
		containers = append(containers, SystemContainer)
	}
	if inProduct {
		containers = append(containers, ProductContainer)
	}
	if inVendor {
		containers = append(containers, VendorContainer)
	}
	if inCts {
		containers = append(containers, CtsContainer)
	}
	if inApex {
		containers = append(containers, ApexContainer)
	}

	return ContainersInfo{
		belongingContainers: containers,
		belongingApexes:     belongingApexes,
	}
}

func setContainerInfo(ctx ModuleContext) {
	if _, ok := ctx.Module().(InstallableModule); ok {
		containersInfo := generateContainerInfo(ctx)
		SetProvider(ctx, ContainersInfoProvider, containersInfo)
	}
}
