// Copyright (C) 2020 The Android Open Source Project
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

package linkerconfig

import (
	"sort"
	"strings"

	"github.com/google/blueprint/proptools"

	"android/soong/android"
	"android/soong/cc"
	"android/soong/etc"
)

var (
	pctx = android.NewPackageContext("android/soong/linkerconfig")
)

func init() {
	pctx.HostBinToolVariable("conv_linker_config", "conv_linker_config")
	registerLinkerConfigBuildComponent(android.InitRegistrationContext)
}

func registerLinkerConfigBuildComponent(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("linker_config", LinkerConfigFactory)
}

type linkerConfigProperties struct {
	// source linker configuration property file
	Src *string `android:"path"`

	// If set to true, allow module to be installed to one of the partitions.
	// Default value is true.
	// Installable should be marked as false for APEX configuration to avoid
	// conflicts of configuration on /system/etc directory.
	Installable *bool
}

type linkerConfig struct {
	android.ModuleBase
	properties linkerConfigProperties

	outputFilePath android.OutputPath
	installDirPath android.InstallPath
}

// Implement PrebuiltEtcModule interface to fit in APEX prebuilt list.
var _ etc.PrebuiltEtcModule = (*linkerConfig)(nil)

func (l *linkerConfig) BaseDir() string {
	return "etc"
}

func (l *linkerConfig) SubDir() string {
	return ""
}

func (l *linkerConfig) OutputFile() android.OutputPath {
	return l.outputFilePath
}

func (l *linkerConfig) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	input := android.PathForModuleSrc(ctx, android.String(l.properties.Src))
	output := android.PathForModuleOut(ctx, "linker.config.pb").OutputPath

	builder := android.NewRuleBuilder(pctx, ctx)
	BuildLinkerConfig(ctx, builder, input, nil, nil, output)
	builder.Build("conv_linker_config", "Generate linker config protobuf "+output.String())

	l.outputFilePath = output
	l.installDirPath = android.PathForModuleInstall(ctx, "etc")
	if !proptools.BoolDefault(l.properties.Installable, true) {
		l.SkipInstall()
	}
	ctx.InstallFile(l.installDirPath, l.outputFilePath.Base(), l.outputFilePath)

	ctx.SetOutputFiles(android.Paths{l.outputFilePath}, "")
}

func BuildLinkerConfig(ctx android.ModuleContext, builder *android.RuleBuilder,
	input android.Path, provideModules []android.Module, requireModules []android.Module, output android.OutputPath) {

	// First, convert the input json to protobuf format
	interimOutput := android.PathForModuleOut(ctx, "temp.pb")
	builder.Command().
		BuiltTool("conv_linker_config").
		Flag("proto").
		FlagWithInput("-s ", input).
		FlagWithOutput("-o ", interimOutput)

	// Secondly, if there's provideLibs gathered from provideModules, append them
	var provideLibs []string
	for _, m := range provideModules {
		if c, ok := m.(*cc.Module); ok && cc.IsStubTarget(c) {
			for _, ps := range c.PackagingSpecs() {
				provideLibs = append(provideLibs, ps.FileName())
			}
		}
	}
	provideLibs = android.FirstUniqueStrings(provideLibs)
	sort.Strings(provideLibs)

	var requireLibs []string
	for _, m := range requireModules {
		if c, ok := m.(*cc.Module); ok && c.HasStubsVariants() && !c.Host() {
			requireLibs = append(requireLibs, c.ImplementationModuleName(ctx)+".so")
		}
	}

	requireLibs = android.FirstUniqueStrings(requireLibs)
	sort.Strings(requireLibs)

	if len(provideLibs) > 0 {
		prevOutput := interimOutput
		interimOutput = android.PathForModuleOut(ctx, "temp_provideLibs.pb")
		builder.Command().
			BuiltTool("conv_linker_config").
			Flag("append").
			FlagWithInput("-s ", prevOutput).
			FlagWithOutput("-o ", interimOutput).
			FlagWithArg("--key ", "provideLibs").
			FlagWithArg("--value ", proptools.ShellEscapeIncludingSpaces(strings.Join(provideLibs, " ")))
		builder.Temporary(prevOutput)
	}
	if len(requireLibs) > 0 {
		prevOutput := interimOutput
		interimOutput = android.PathForModuleOut(ctx, "temp_requireLibs.pb")
		builder.Command().
			BuiltTool("conv_linker_config").
			Flag("append").
			FlagWithInput("-s ", prevOutput).
			FlagWithOutput("-o ", interimOutput).
			FlagWithArg("--key ", "requireLibs").
			FlagWithArg("--value ", proptools.ShellEscapeIncludingSpaces(strings.Join(requireLibs, " ")))
		builder.Temporary(prevOutput)
	}

	// cp to the final output
	builder.Command().Text("cp").Input(interimOutput).Output(output)

	builder.Temporary(interimOutput)
	builder.DeleteTemporaryFiles()
}

// linker_config generates protobuf file from json file. This protobuf file will be used from
// linkerconfig while generating ld.config.txt. Format of this file can be found from
// https://android.googlesource.com/platform/system/linkerconfig/+/main/README.md
func LinkerConfigFactory() android.Module {
	m := &linkerConfig{}
	m.AddProperties(&m.properties)
	android.InitAndroidArchModule(m, android.HostAndDeviceSupported, android.MultilibFirst)
	return m
}

func (l *linkerConfig) AndroidMkEntries() []android.AndroidMkEntries {
	installable := proptools.BoolDefault(l.properties.Installable, true)
	return []android.AndroidMkEntries{android.AndroidMkEntries{
		Class:      "ETC",
		OutputFile: android.OptionalPathForPath(l.outputFilePath),
		ExtraEntries: []android.AndroidMkExtraEntriesFunc{
			func(ctx android.AndroidMkExtraEntriesContext, entries *android.AndroidMkEntries) {
				entries.SetString("LOCAL_MODULE_PATH", l.installDirPath.String())
				entries.SetString("LOCAL_INSTALLED_MODULE_STEM", l.outputFilePath.Base())
				entries.SetBoolIfTrue("LOCAL_UNINSTALLABLE_MODULE", !installable)
			},
		},
	}}
}
