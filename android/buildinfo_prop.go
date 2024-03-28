// Copyright 2022 Google Inc. All rights reserved.
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
	"fmt"
	"strings"

	"github.com/google/blueprint/proptools"
)

func init() {
	ctx := InitRegistrationContext
	ctx.RegisterParallelSingletonModuleType("buildinfo_prop", buildinfoPropFactory)
}

type buildinfoPropProperties struct {
	// Whether this module is directly installable to one of the partitions. Default: true.
	Installable *bool
}

type buildinfoPropModule struct {
	SingletonModuleBase

	properties buildinfoPropProperties

	outputFilePath OutputPath
	installPath    InstallPath
}

var _ OutputFileProducer = (*buildinfoPropModule)(nil)

func (p *buildinfoPropModule) installable() bool {
	return proptools.BoolDefault(p.properties.Installable, true)
}

// OutputFileProducer
func (p *buildinfoPropModule) OutputFiles(tag string) (Paths, error) {
	if tag != "" {
		return nil, fmt.Errorf("unsupported tag %q", tag)
	}
	return Paths{p.outputFilePath}, nil
}

func getBuildVariant(config Config) string {
	if config.Eng() {
		return "eng"
	} else if config.Debuggable() {
		return "userdebug"
	} else {
		return "user"
	}
}

func getBuildFlavor(config Config) string {
	buildFlavor := config.DeviceProduct() + "-" + getBuildVariant(config)
	if InList("address", config.SanitizeDevice()) && !strings.Contains(buildFlavor, "_asan") {
		buildFlavor += "_asan"
	}
	return buildFlavor
}

func shouldAddBuildThumbprint(config Config) bool {
	knownOemProperties := []string{
		"ro.product.brand",
		"ro.product.name",
		"ro.product.device",
	}

	for _, knownProp := range knownOemProperties {
		if InList(knownProp, config.OemProperties()) {
			return true
		}
	}
	return false
}

func (p *buildinfoPropModule) GenerateAndroidBuildActions(ctx ModuleContext) {
	p.outputFilePath = PathForModuleOut(ctx, p.Name()).OutputPath
	if !ctx.Config().KatiEnabled() {
		WriteFileRule(ctx, p.outputFilePath, "# no buildinfo.prop if kati is disabled")
		return
	}

	rule := NewRuleBuilder(pctx, ctx)

	config := ctx.Config()
	buildVariant := getBuildVariant(config)
	buildFlavor := getBuildFlavor(config)

	cmd := rule.Command().BuiltTool("buildinfo")

	if config.BoardUseVbmetaDigestInFingerprint() {
		cmd.Flag("--use-vbmeta-digest-in-fingerprint")
	}

	cmd.FlagWithArg("--build-flavor=", buildFlavor)
	cmd.FlagWithInput("--build-hostname-file=", config.BuildHostnameFile(ctx))
	cmd.FlagWithArg("--build-id=", config.BuildId())
	cmd.FlagWithArg("--build-keys=", config.BuildKeys())

	// shouldn't depend on BuildNumberFile and BuildThumbprintFile to prevent from rebuilding
	// on every incremental build.
	cmd.FlagWithArg("--build-number-file=", config.BuildNumberFile(ctx).String())
	if shouldAddBuildThumbprint(config) {
		cmd.FlagWithArg("--build-thumbprint-file=", config.BuildThumbprintFile(ctx).String())
	}

	cmd.FlagWithArg("--build-type=", config.BuildType())
	cmd.FlagWithArg("--build-username=", config.Getenv("BUILD_USERNAME"))
	cmd.FlagWithArg("--build-variant=", buildVariant)
	cmd.FlagForEachArg("--cpu-abis=", config.DeviceAbi())

	// shouldn't depend on BUILD_DATETIME_FILE to prevent from rebuilding on every incremental
	// build.
	cmd.FlagWithArg("--date-file=", ctx.Config().Getenv("BUILD_DATETIME_FILE"))

	if len(config.ProductLocales()) > 0 {
		cmd.FlagWithArg("--default-locale=", config.ProductLocales()[0])
	}

	cmd.FlagForEachArg("--default-wifi-channels=", config.ProductDefaultWifiChannels())
	cmd.FlagWithArg("--device=", config.DeviceName())
	if config.DisplayBuildNumber() {
		cmd.Flag("--display-build-number")
	}

	cmd.FlagWithArg("--platform-base-os=", config.PlatformBaseOS())
	cmd.FlagWithArg("--platform-display-version=", config.PlatformDisplayVersionName())
	cmd.FlagWithArg("--platform-min-supported-target-sdk-version=", config.PlatformMinSupportedTargetSdkVersion())
	cmd.FlagWithInput("--platform-preview-sdk-fingerprint-file=", ApiFingerprintPath(ctx))
	cmd.FlagWithArg("--platform-preview-sdk-version=", config.PlatformPreviewSdkVersion())
	cmd.FlagWithArg("--platform-sdk-version=", config.PlatformSdkVersion().String())
	cmd.FlagWithArg("--platform-security-patch=", config.PlatformSecurityPatch())
	cmd.FlagWithArg("--platform-version=", config.PlatformVersionName())
	cmd.FlagWithArg("--platform-version-codename=", config.PlatformSdkCodename())
	cmd.FlagForEachArg("--platform-version-all-codenames=", config.PlatformVersionActiveCodenames())
	cmd.FlagWithArg("--platform-version-known-codenames=", config.PlatformVersionKnownCodenames())
	cmd.FlagWithArg("--platform-version-last-stable=", config.PlatformVersionLastStable())
	cmd.FlagWithArg("--product=", config.DeviceProduct())

	cmd.FlagWithOutput("--out=", p.outputFilePath)

	rule.Build(ctx.ModuleName(), "generating buildinfo props")

	if !p.installable() {
		p.SkipInstall()
	}

	p.installPath = PathForModuleInstall(ctx)
	ctx.InstallFile(p.installPath, p.Name(), p.outputFilePath)
}

func (f *buildinfoPropModule) GenerateSingletonBuildActions(ctx SingletonContext) {
	// does nothing; buildinfo_prop is a singeton because two buildinfo modules don't make sense.
}

func (p *buildinfoPropModule) AndroidMkEntries() []AndroidMkEntries {
	return []AndroidMkEntries{AndroidMkEntries{
		Class:      "ETC",
		OutputFile: OptionalPathForPath(p.outputFilePath),
		ExtraEntries: []AndroidMkExtraEntriesFunc{
			func(ctx AndroidMkExtraEntriesContext, entries *AndroidMkEntries) {
				entries.SetString("LOCAL_MODULE_PATH", p.installPath.String())
				entries.SetString("LOCAL_INSTALLED_MODULE_STEM", p.outputFilePath.Base())
				entries.SetBoolIfTrue("LOCAL_UNINSTALLABLE_MODULE", !p.installable())
			},
		},
	}}
}

// buildinfo_prop module generates a build.prop file, which contains a set of common
// system/build.prop properties, such as ro.build.version.*.  Not all properties are implemented;
// currently this module is only for microdroid.
func buildinfoPropFactory() SingletonModule {
	module := &buildinfoPropModule{}
	module.AddProperties(&module.properties)
	InitAndroidModule(module)
	return module
}
