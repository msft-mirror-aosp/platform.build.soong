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

package fuzz

// This file contains the common code for compiling C/C++ and Rust fuzzers for Android.

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/google/blueprint/proptools"

	"android/soong/android"
)

type Lang string

const (
	Cc   Lang = "cc"
	Rust Lang = "rust"
	Java Lang = "java"
)

type Framework string

const (
	AFL              Framework = "afl"
	LibFuzzer        Framework = "libfuzzer"
	Jazzer           Framework = "jazzer"
	UnknownFramework Framework = "unknownframework"
)

var BoolDefault = proptools.BoolDefault

type FuzzModule struct {
	android.ModuleBase
	android.DefaultableModuleBase
	android.ApexModuleBase
}

type FuzzPackager struct {
	Packages                android.Paths
	FuzzTargets             map[string]bool
	SharedLibInstallStrings []string
}

type FileToZip struct {
	SourceFilePath        android.Path
	DestinationPathPrefix string
}

type ArchOs struct {
	HostOrTarget string
	Arch         string
	Dir          string
}

type PrivilegedLevel string

const (
	// Environment with the most minimal permissions.
	Constrained PrivilegedLevel = "Constrained"
	// Typical execution environment running unprivileged code.
	Unprivileged = "Unprivileged"
	// May have access to elevated permissions.
	Privileged = "Privileged"
	// Trusted computing base.
	Tcb = "TCB"
	// Bootloader chain.
	Bootloader = "Bootloader"
	// Tusted execution environment.
	Tee = "Tee"
	// Secure enclave.
	Se = "Se"
	// Other.
	Other = "Other"
)

func IsValidConfig(fuzzModule FuzzPackagedModule, moduleName string) bool {
	var config = fuzzModule.FuzzProperties.Fuzz_config
	if config != nil {
		var level = PrivilegedLevel(config.Privilege_level)
		if level != "" {
			switch level {
			case Constrained, Unprivileged, Privileged, Tcb, Bootloader, Tee, Se, Other:
				return true
			}
			panic(fmt.Errorf("Invalid privileged level in fuzz config in %s", moduleName))
		}
		return true
	} else {
		return false
	}
}

type FuzzConfig struct {
	// Email address of people to CC on bugs or contact about this fuzz target.
	Cc []string `json:"cc,omitempty"`
	// A brief description of what the fuzzed code does.
	Description string `json:"description,omitempty"`
	// Can this code be triggered remotely or only locally.
	Remotely_accessible *bool `json:"remotely_accessible,omitempty"`
	// Is the fuzzed code host only, i.e. test frameworks or support utilities.
	Host_only *bool `json:"host_only,omitempty"`
	// Can third party/untrusted apps supply data to fuzzed code.
	Untrusted_data *bool `json:"untrusted_data,omitempty"`
	// Is the code being fuzzed in a privileged, constrained or any other
	// context from:
	// https://source.android.com/security/overview/updates-resources#context_types.
	Privilege_level PrivilegedLevel `json:"privilege_level,omitempty"`
	// Can the fuzzed code isolated or can be called by multiple users/processes.
	Isolated *bool `json:"users_isolation,omitempty"`
	// When code was relaeased or will be released.
	Production_date string `json:"production_date,omitempty"`
	// Prevents critical service functionality like phone calls, bluetooth, etc.
	Critical *bool `json:"critical,omitempty"`
	// Specify whether to enable continuous fuzzing on devices. Defaults to true.
	Fuzz_on_haiku_device *bool `json:"fuzz_on_haiku_device,omitempty"`
	// Specify whether to enable continuous fuzzing on host. Defaults to true.
	Fuzz_on_haiku_host *bool `json:"fuzz_on_haiku_host,omitempty"`
	// Component in Google's bug tracking system that bugs should be filed to.
	Componentid *int64 `json:"componentid,omitempty"`
	// Hotlists in Google's bug tracking system that bugs should be marked with.
	Hotlists []string `json:"hotlists,omitempty"`
	// Specify whether this fuzz target was submitted by a researcher. Defaults
	// to false.
	Researcher_submitted *bool `json:"researcher_submitted,omitempty"`
	// Specify who should be acknowledged for CVEs in the Android Security
	// Bulletin.
	Acknowledgement []string `json:"acknowledgement,omitempty"`
	// Additional options to be passed to libfuzzer when run in Haiku.
	Libfuzzer_options []string `json:"libfuzzer_options,omitempty"`
	// Additional options to be passed to HWASAN when running on-device in Haiku.
	Hwasan_options []string `json:"hwasan_options,omitempty"`
	// Additional options to be passed to HWASAN when running on host in Haiku.
	Asan_options []string `json:"asan_options,omitempty"`
	// If there's a Java fuzzer with JNI, a different version of Jazzer would
	// need to be added to the fuzzer package than one without JNI
	IsJni *bool `json:"is_jni,omitempty"`
	// List of modules for monitoring coverage drops in directories (e.g. "libicu")
	Target_modules []string `json:"target_modules,omitempty"`
}

type FuzzFrameworks struct {
	Afl       *bool
	Libfuzzer *bool
	Jazzer    *bool
}

type FuzzProperties struct {
	// Optional list of seed files to be installed to the fuzz target's output
	// directory.
	Corpus []string `android:"path"`
	// Optional list of data files to be installed to the fuzz target's output
	// directory. Directory structure relative to the module is preserved.
	Data []string `android:"path"`
	// Optional dictionary to be installed to the fuzz target's output directory.
	Dictionary *string `android:"path"`
	// Define the fuzzing frameworks this fuzz target can be built for. If
	// empty then the fuzz target will be available to be  built for all fuzz
	// frameworks available
	Fuzzing_frameworks *FuzzFrameworks
	// Config for running the target on fuzzing infrastructure.
	Fuzz_config *FuzzConfig
}

type FuzzPackagedModule struct {
	FuzzProperties        FuzzProperties
	Dictionary            android.Path
	Corpus                android.Paths
	CorpusIntermediateDir android.Path
	Config                android.Path
	Data                  android.Paths
	DataIntermediateDir   android.Path
}

func GetFramework(ctx android.LoadHookContext, lang Lang) Framework {
	framework := ctx.Config().Getenv("FUZZ_FRAMEWORK")

	if lang == Cc {
		switch strings.ToLower(framework) {
		case "":
			return LibFuzzer
		case "libfuzzer":
			return LibFuzzer
		case "afl":
			return AFL
		}
	} else if lang == Rust {
		return LibFuzzer
	} else if lang == Java {
		return Jazzer
	}

	ctx.ModuleErrorf(fmt.Sprintf("%s is not a valid fuzzing framework for %s", framework, lang))
	return UnknownFramework
}

func IsValidFrameworkForModule(targetFramework Framework, lang Lang, moduleFrameworks *FuzzFrameworks) bool {
	if targetFramework == UnknownFramework {
		return false
	}

	if moduleFrameworks == nil {
		return true
	}

	switch targetFramework {
	case LibFuzzer:
		return proptools.BoolDefault(moduleFrameworks.Libfuzzer, true)
	case AFL:
		return proptools.BoolDefault(moduleFrameworks.Afl, true)
	case Jazzer:
		return proptools.BoolDefault(moduleFrameworks.Jazzer, true)
	default:
		panic("%s is not supported as a fuzz framework")
	}
}

func IsValid(fuzzModule FuzzModule) bool {
	// Discard ramdisk + vendor_ramdisk + recovery modules, they're duplicates of
	// fuzz targets we're going to package anyway.
	if !fuzzModule.Enabled() || fuzzModule.InRamdisk() || fuzzModule.InVendorRamdisk() || fuzzModule.InRecovery() {
		return false
	}

	// Discard modules that are in an unavailable namespace.
	if !fuzzModule.ExportedToMake() {
		return false
	}

	return true
}

func (s *FuzzPackager) PackageArtifacts(ctx android.SingletonContext, module android.Module, fuzzModule FuzzPackagedModule, archDir android.OutputPath, builder *android.RuleBuilder) []FileToZip {
	// Package the corpora into a zipfile.
	var files []FileToZip
	if fuzzModule.Corpus != nil {
		corpusZip := archDir.Join(ctx, module.Name()+"_seed_corpus.zip")
		command := builder.Command().BuiltTool("soong_zip").
			Flag("-j").
			FlagWithOutput("-o ", corpusZip)
		rspFile := corpusZip.ReplaceExtension(ctx, "rsp")
		command.FlagWithRspFileInputList("-r ", rspFile, fuzzModule.Corpus)
		files = append(files, FileToZip{corpusZip, ""})
	}

	// Package the data into a zipfile.
	if fuzzModule.Data != nil {
		dataZip := archDir.Join(ctx, module.Name()+"_data.zip")
		command := builder.Command().BuiltTool("soong_zip").
			FlagWithOutput("-o ", dataZip)
		for _, f := range fuzzModule.Data {
			intermediateDir := strings.TrimSuffix(f.String(), f.Rel())
			command.FlagWithArg("-C ", intermediateDir)
			command.FlagWithInput("-f ", f)
		}
		files = append(files, FileToZip{dataZip, ""})
	}

	// The dictionary.
	if fuzzModule.Dictionary != nil {
		files = append(files, FileToZip{fuzzModule.Dictionary, ""})
	}

	// Additional fuzz config.
	if fuzzModule.Config != nil && IsValidConfig(fuzzModule, module.Name()) {
		files = append(files, FileToZip{fuzzModule.Config, ""})
	}

	return files
}

func (s *FuzzPackager) BuildZipFile(ctx android.SingletonContext, module android.Module, fuzzModule FuzzPackagedModule, files []FileToZip, builder *android.RuleBuilder, archDir android.OutputPath, archString string, hostOrTargetString string, archOs ArchOs, archDirs map[ArchOs][]FileToZip) ([]FileToZip, bool) {
	fuzzZip := archDir.Join(ctx, module.Name()+".zip")

	command := builder.Command().BuiltTool("soong_zip").
		Flag("-j").
		FlagWithOutput("-o ", fuzzZip)

	for _, file := range files {
		if file.DestinationPathPrefix != "" {
			command.FlagWithArg("-P ", file.DestinationPathPrefix)
		} else {
			command.Flag("-P ''")
		}
		command.FlagWithInput("-f ", file.SourceFilePath)
	}

	builder.Build("create-"+fuzzZip.String(),
		"Package "+module.Name()+" for "+archString+"-"+hostOrTargetString)

	// Don't add modules to 'make haiku-rust' that are set to not be
	// exported to the fuzzing infrastructure.
	if config := fuzzModule.FuzzProperties.Fuzz_config; config != nil {
		if strings.Contains(hostOrTargetString, "host") && !BoolDefault(config.Fuzz_on_haiku_host, true) {
			return archDirs[archOs], false
		} else if !BoolDefault(config.Fuzz_on_haiku_device, true) {
			return archDirs[archOs], false
		}
	}

	s.FuzzTargets[module.Name()] = true
	archDirs[archOs] = append(archDirs[archOs], FileToZip{fuzzZip, ""})

	return archDirs[archOs], true
}

func (f *FuzzConfig) String() string {
	b, err := json.Marshal(f)
	if err != nil {
		panic(err)
	}

	return string(b)
}

func (s *FuzzPackager) CreateFuzzPackage(ctx android.SingletonContext, archDirs map[ArchOs][]FileToZip, fuzzType Lang, pctx android.PackageContext) {
	var archOsList []ArchOs
	for archOs := range archDirs {
		archOsList = append(archOsList, archOs)
	}
	sort.Slice(archOsList, func(i, j int) bool { return archOsList[i].Dir < archOsList[j].Dir })

	for _, archOs := range archOsList {
		filesToZip := archDirs[archOs]
		arch := archOs.Arch
		hostOrTarget := archOs.HostOrTarget
		builder := android.NewRuleBuilder(pctx, ctx)
		zipFileName := "fuzz-" + hostOrTarget + "-" + arch + ".zip"
		if fuzzType == Rust {
			zipFileName = "fuzz-rust-" + hostOrTarget + "-" + arch + ".zip"
		}
		if fuzzType == Java {
			zipFileName = "fuzz-java-" + hostOrTarget + "-" + arch + ".zip"
		}

		outputFile := android.PathForOutput(ctx, zipFileName)

		s.Packages = append(s.Packages, outputFile)

		command := builder.Command().BuiltTool("soong_zip").
			Flag("-j").
			FlagWithOutput("-o ", outputFile).
			Flag("-L 0") // No need to try and re-compress the zipfiles.

		for _, fileToZip := range filesToZip {
			if fileToZip.DestinationPathPrefix != "" {
				command.FlagWithArg("-P ", fileToZip.DestinationPathPrefix)
			} else {
				command.Flag("-P ''")
			}
			command.FlagWithInput("-f ", fileToZip.SourceFilePath)

		}
		builder.Build("create-fuzz-package-"+arch+"-"+hostOrTarget,
			"Create fuzz target packages for "+arch+"-"+hostOrTarget)
	}
}

func (s *FuzzPackager) PreallocateSlice(ctx android.MakeVarsContext, targets string) {
	fuzzTargets := make([]string, 0, len(s.FuzzTargets))
	for target, _ := range s.FuzzTargets {
		fuzzTargets = append(fuzzTargets, target)
	}

	sort.Strings(fuzzTargets)
	ctx.Strict(targets, strings.Join(fuzzTargets, " "))
}
