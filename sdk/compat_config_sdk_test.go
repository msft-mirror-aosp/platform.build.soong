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

package sdk

import (
	"testing"

	"android/soong/android"
	"android/soong/java"
)

func testSnapshotWithCompatConfig(t *testing.T, sdk string) {
	result := android.GroupFixturePreparers(
		prepareForSdkTestWithJava,
		java.PrepareForTestWithPlatformCompatConfig,
		prepareForSdkTestWithApex,
	).RunTestWithBp(t, sdk+`
		platform_compat_config {
			name: "myconfig",
		}
	`)

	CheckSnapshot(t, result, "mysdk", "",
		checkAndroidBpContents(`
// This is auto-generated. DO NOT EDIT.

prebuilt_platform_compat_config {
    name: "myconfig",
    prefer: false,
    visibility: ["//visibility:public"],
    metadata: "compat_configs/myconfig/myconfig_meta.xml",
}
`),
		checkAllCopyRules(`
.intermediates/myconfig/android_common/myconfig_meta.xml -> compat_configs/myconfig/myconfig_meta.xml
`),
		snapshotTestChecker(checkSnapshotWithoutSource,
			func(t *testing.T, result *android.TestResult) {
				// Make sure that the snapshot metadata is collated by the platform compat config singleton.
				java.CheckMergedCompatConfigInputs(t, result, "snapshot module", "snapshot/compat_configs/myconfig/myconfig_meta.xml")
			}),

		snapshotTestChecker(checkSnapshotWithSourcePreferred,
			func(t *testing.T, result *android.TestResult) {
				// Make sure that the snapshot metadata is collated by the platform compat config singleton.
				java.CheckMergedCompatConfigInputs(t, result, "snapshot module",
					"out/soong/.intermediates/myconfig/android_common/myconfig_meta.xml",
				)
			}),

		snapshotTestChecker(checkSnapshotPreferredWithSource,
			func(t *testing.T, result *android.TestResult) {
				// Make sure that the snapshot metadata is collated by the platform compat config singleton.
				java.CheckMergedCompatConfigInputs(t, result, "snapshot module",
					"snapshot/compat_configs/myconfig/myconfig_meta.xml",
				)
			}),
	)
}

func TestSnapshotWithCompatConfig(t *testing.T) {
	testSnapshotWithCompatConfig(t, `
		sdk {
			name: "mysdk",
			compat_configs: ["myconfig"],
		}
`)
}

func TestSnapshotWithCompatConfig_Apex(t *testing.T) {
	testSnapshotWithCompatConfig(t, `
		apex {
			name: "myapex",
			key: "myapex.key",
			min_sdk_version: "2",
			compat_configs: ["myconfig"],
		}

		sdk {
			name: "mysdk",
			apexes: ["myapex"],
		}
`)
}
