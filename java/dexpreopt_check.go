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
	"strings"

	"github.com/google/blueprint"

	"android/soong/android"
	"android/soong/dexpreopt"

	"github.com/google/blueprint/pathtools"
)

func init() {
	RegisterDexpreoptCheckBuildComponents(android.InitRegistrationContext)
}

func RegisterDexpreoptCheckBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterParallelSingletonModuleType("dexpreopt_systemserver_check", dexpreoptSystemserverCheckFactory)
}

// A build-time check to verify if all compilation artifacts of system server jars are installed
// into the system image. When the check fails, it means that dexpreopting is not working for some
// system server jars and needs to be fixed.
// This singleton module generates a list of the paths to the artifacts based on
// PRODUCT_SYSTEM_SERVER_JARS and PRODUCT_APEX_SYSTEM_SERVER_JARS, and passes it to Make via a
// variable. Make will then do the actual check.
// Currently, it only checks artifacts of modules defined in Soong. Artifacts of modules defined in
// Makefile are generated by a script generated by dexpreopt_gen, and their existence is unknown to
// Make and Ninja.
type dexpreoptSystemserverCheck struct {
	android.SingletonModuleBase

	// The install paths to the compilation artifacts.
	artifacts []string
}

func dexpreoptSystemserverCheckFactory() android.SingletonModule {
	m := &dexpreoptSystemserverCheck{}
	android.InitAndroidArchModule(m, android.DeviceSupported, android.MultilibCommon)
	return m
}

func getInstallPath(ctx android.ModuleContext, location string) android.InstallPath {
	return android.PathForModuleInPartitionInstall(
		ctx, "", strings.TrimPrefix(location, "/"))
}

type systemServerDependencyTag struct {
	blueprint.BaseDependencyTag
}

// systemServerJarDepTag willl be used for validation. Skip visiblility.
func (b systemServerDependencyTag) ExcludeFromVisibilityEnforcement() {
}

var (
	// dep tag for platform and apex system server jars
	systemServerJarDepTag = systemServerDependencyTag{}
)

var _ android.ExcludeFromVisibilityEnforcementTag = systemServerJarDepTag

// Add a depenendency on the system server jars. The dexpreopt files of those will be emitted to make.
// The kati packaging system will verify that those files appear in installed files.
// Adding the dependency allows the singleton module to determine whether an apex system server jar is system_ext specific.
func (m *dexpreoptSystemserverCheck) DepsMutator(ctx android.BottomUpMutatorContext) {
	global := dexpreopt.GetGlobalConfig(ctx)
	targets := ctx.Config().Targets[android.Android]

	// The check should be skipped on unbundled builds because system server jars are not preopted on
	// unbundled builds since the artifacts are installed into the system image, not the APEXes.
	if global.DisablePreopt || global.OnlyPreoptArtBootImage || len(targets) == 0 || ctx.Config().UnbundledBuild() {
		return
	}

	ctx.AddDependency(ctx.Module(), systemServerJarDepTag, global.AllSystemServerJars(ctx).CopyOfJars()...)
}

func (m *dexpreoptSystemserverCheck) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	global := dexpreopt.GetGlobalConfig(ctx)
	targets := ctx.Config().Targets[android.Android]

	ctx.VisitDirectDepsWithTag(systemServerJarDepTag, func(systemServerJar android.Module) {
		partition := "system"
		if systemServerJar.InstallInSystemExt() && ctx.Config().InstallApexSystemServerDexpreoptSamePartition() {
			partition = ctx.DeviceConfig().SystemExtPath() // system_ext
		}
		dexLocation := dexpreopt.GetSystemServerDexLocation(ctx, global, systemServerJar.Name())
		odexLocation := dexpreopt.ToOdexPath(dexLocation, targets[0].Arch.ArchType, partition)
		odexPath := getInstallPath(ctx, odexLocation)
		vdexPath := getInstallPath(ctx, pathtools.ReplaceExtension(odexLocation, "vdex"))
		m.artifacts = append(m.artifacts, odexPath.String(), vdexPath.String())
	})
}

func (m *dexpreoptSystemserverCheck) GenerateSingletonBuildActions(ctx android.SingletonContext) {
}

func (m *dexpreoptSystemserverCheck) MakeVars(ctx android.MakeVarsContext) {
	ctx.Strict("DEXPREOPT_SYSTEMSERVER_ARTIFACTS", strings.Join(m.artifacts, " "))
}
