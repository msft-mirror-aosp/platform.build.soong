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
	"path/filepath"
	"strings"

	"android/soong/android"

	"github.com/google/blueprint/proptools"
)

type fsverityProperties struct {
	// Patterns of files for fsverity metadata generation.  For each matched file, a .fsv_meta file
	// will be generated and included to the filesystem image.
	// etc/security/fsverity/BuildManifest.apk will also be generated which contains information
	// about generated .fsv_meta files.
	Inputs proptools.Configurable[[]string]

	// APK libraries to link against, for etc/security/fsverity/BuildManifest.apk
	Libs proptools.Configurable[[]string] `android:"path"`
}

func (f *filesystem) writeManifestGeneratorListFile(ctx android.ModuleContext, outputPath android.WritablePath, matchedSpecs []android.PackagingSpec, rebasedDir android.OutputPath) {
	var buf strings.Builder
	for _, spec := range matchedSpecs {
		buf.WriteString(rebasedDir.Join(ctx, spec.RelPathInPackage()).String())
		buf.WriteRune('\n')
	}
	android.WriteFileRuleVerbatim(ctx, outputPath, buf.String())
}

func (f *filesystem) buildFsverityMetadataFiles(
	ctx android.ModuleContext,
	builder *android.RuleBuilder,
	specs map[string]android.PackagingSpec,
	rootDir android.OutputPath,
	rebasedDir android.OutputPath,
	fullInstallPaths *[]FullInstallPathInfo,
) {
	match := func(path string) bool {
		for _, pattern := range f.properties.Fsverity.Inputs.GetOrDefault(ctx, nil) {
			if matched, err := filepath.Match(pattern, path); matched {
				return true
			} else if err != nil {
				ctx.PropertyErrorf("fsverity.inputs", "bad pattern %q", pattern)
				return false
			}
		}
		return false
	}

	var matchedSpecs []android.PackagingSpec
	for _, relPath := range android.SortedKeys(specs) {
		if match(relPath) {
			matchedSpecs = append(matchedSpecs, specs[relPath])
		}
	}

	if len(matchedSpecs) == 0 {
		return
	}

	fsverityPath := ctx.Config().HostToolPath(ctx, "fsverity")

	// STEP 1: generate .fsv_meta
	var sb strings.Builder
	sb.WriteString("set -e\n")
	for _, spec := range matchedSpecs {
		// srcPath is copied by CopySpecsToDir()
		srcPath := rebasedDir.Join(ctx, spec.RelPathInPackage())
		destPath := rebasedDir.Join(ctx, spec.RelPathInPackage()+".fsv_meta")
		builder.Command().
			BuiltTool("fsverity_metadata_generator").
			FlagWithInput("--fsverity-path ", fsverityPath).
			FlagWithArg("--signature ", "none").
			FlagWithArg("--hash-alg ", "sha256").
			FlagWithOutput("--output ", destPath).
			Text(srcPath.String())
		f.appendToEntry(ctx, destPath)
		*fullInstallPaths = append(*fullInstallPaths, FullInstallPathInfo{
			SourcePath:      destPath,
			FullInstallPath: android.PathForModuleInPartitionInstall(ctx, f.PartitionType(), spec.RelPathInPackage()+".fsv_meta"),
		})
	}

	fsVerityBaseDir := rootDir.String()
	if f.PartitionType() == "system_ext" {
		// Use the equivalent of $PRODUCT_OUT as the base dir.
		// This ensures that the paths in build_manifest.pb contain on-device paths
		// e.g. system_ext/framework/javalib.jar
		// and not framework/javalib.jar.
		//
		// Although base-dir is outside the rootdir provided for packaging, this action
		// is hermetic since it uses `manifestGeneratorListPath` to filter the files to be written to build_manifest.pb
		fsVerityBaseDir = filepath.Dir(rootDir.String())
	}

	// STEP 2: generate signed BuildManifest.apk
	// STEP 2-1: generate build_manifest.pb
	manifestGeneratorListPath := android.PathForModuleOut(ctx, "fsverity_manifest.list")
	f.writeManifestGeneratorListFile(ctx, manifestGeneratorListPath, matchedSpecs, rebasedDir)
	assetsPath := android.PathForModuleOut(ctx, "fsverity_manifest/assets")
	manifestPbPath := assetsPath.Join(ctx, "build_manifest.pb")
	builder.Command().Text("rm -rf " + assetsPath.String())
	builder.Command().Text("mkdir -p " + assetsPath.String())
	builder.Command().
		BuiltTool("fsverity_manifest_generator").
		FlagWithInput("--fsverity-path ", fsverityPath).
		FlagWithArg("--base-dir ", fsVerityBaseDir).
		FlagWithArg("--output ", manifestPbPath.String()).
		FlagWithInput("@", manifestGeneratorListPath)

	f.appendToEntry(ctx, manifestPbPath)
	f.appendToEntry(ctx, manifestGeneratorListPath)

	// STEP 2-2: generate BuildManifest.apk (unsigned)
	apkNameSuffix := ""
	if f.PartitionType() == "system_ext" {
		//https://source.corp.google.com/h/googleplex-android/platform/build/+/e392d2b486c2d4187b20a72b1c67cc737ecbcca5:core/Makefile;l=3410;drc=ea8f34bc1d6e63656b4ec32f2391e9d54b3ebb6b;bpv=1;bpt=0
		apkNameSuffix = "SystemExt"
	}
	apkPath := rebasedDir.Join(ctx, "etc", "security", "fsverity", fmt.Sprintf("BuildManifest%s.apk", apkNameSuffix))
	idsigPath := rebasedDir.Join(ctx, "etc", "security", "fsverity", fmt.Sprintf("BuildManifest%s.apk.idsig", apkNameSuffix))
	manifestTemplatePath := android.PathForSource(ctx, "system/security/fsverity/AndroidManifest.xml")
	libs := android.PathsForModuleSrc(ctx, f.properties.Fsverity.Libs.GetOrDefault(ctx, nil))

	minSdkVersion := ctx.Config().PlatformSdkCodename()
	if minSdkVersion == "REL" {
		minSdkVersion = ctx.Config().PlatformSdkVersion().String()
	}

	unsignedApkCommand := builder.Command().
		Textf("mkdir -p %s && ", filepath.Dir(apkPath.String())).
		BuiltTool("aapt2").
		Text("link").
		FlagWithOutput("-o ", apkPath).
		FlagWithArg("-A ", assetsPath.String())
	for _, lib := range libs {
		unsignedApkCommand.FlagWithInput("-I ", lib)
	}
	unsignedApkCommand.
		FlagWithArg("--min-sdk-version ", minSdkVersion).
		FlagWithArg("--version-code ", ctx.Config().PlatformSdkVersion().String()).
		FlagWithArg("--version-name ", ctx.Config().AppsDefaultVersionName()).
		FlagWithInput("--manifest ", manifestTemplatePath).
		Text(" --rename-manifest-package com.android.security.fsverity_metadata." + f.partitionName())
	*fullInstallPaths = append(*fullInstallPaths, FullInstallPathInfo{
		SourcePath:      apkPath,
		FullInstallPath: android.PathForModuleInPartitionInstall(ctx, f.PartitionType(), fmt.Sprintf("etc/security/fsverity/BuildManifest%s.apk", apkNameSuffix)),
	})

	f.appendToEntry(ctx, apkPath)

	// STEP 2-3: sign BuildManifest.apk
	pemPath, keyPath := ctx.Config().DefaultAppCertificate(ctx)
	builder.Command().
		BuiltTool("apksigner").
		Text("sign").
		FlagWithArg("--in ", apkPath.String()).
		FlagWithInput("--cert ", pemPath).
		FlagWithInput("--key ", keyPath).
		ImplicitOutput(idsigPath)
	*fullInstallPaths = append(*fullInstallPaths, FullInstallPathInfo{
		SourcePath:      idsigPath,
		FullInstallPath: android.PathForModuleInPartitionInstall(ctx, f.PartitionType(), fmt.Sprintf("etc/security/fsverity/BuildManifest%s.apk.idsig", apkNameSuffix)),
	})

	f.appendToEntry(ctx, idsigPath)
}
