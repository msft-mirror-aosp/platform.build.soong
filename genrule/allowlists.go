// Copyright 2023 Google Inc. All rights reserved.
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

package genrule

var (
	DepfileAllowList = []string{
		"depfile_allowed_for_test",
		"tflite_support_spm_config",
		"tflite_support_spm_encoder_config",
		"gen_uwb_core_proto",
		"libtextclassifier_fbgen_utils_flatbuffers_flatbuffers_test",
		"libtextclassifier_fbgen_utils_lua_utils_tests",
		"libtextclassifier_fbgen_lang_id_common_flatbuffers_model",
		"libtextclassifier_fbgen_lang_id_common_flatbuffers_embedding-network",
		"libtextclassifier_fbgen_annotator_datetime_datetime",
		"libtextclassifier_fbgen_annotator_model",
		"libtextclassifier_fbgen_annotator_experimental_experimental",
		"libtextclassifier_fbgen_annotator_entity-data",
		"libtextclassifier_fbgen_annotator_person_name_person_name_model",
		"libtextclassifier_fbgen_utils_tflite_text_encoder_config",
		"libtextclassifier_fbgen_utils_codepoint-range",
		"libtextclassifier_fbgen_utils_intents_intent-config",
		"libtextclassifier_fbgen_utils_flatbuffers_flatbuffers",
		"libtextclassifier_fbgen_utils_zlib_buffer",
		"libtextclassifier_fbgen_utils_tokenizer",
		"libtextclassifier_fbgen_utils_grammar_rules",
		"libtextclassifier_fbgen_utils_grammar_semantics_expression",
		"libtextclassifier_fbgen_utils_resources",
		"libtextclassifier_fbgen_utils_i18n_language-tag",
		"libtextclassifier_fbgen_utils_normalization",
		"libtextclassifier_fbgen_utils_container_bit-vector",
		"libtextclassifier_fbgen_actions_actions-entity-data",
		"libtextclassifier_fbgen_actions_actions_model",
		"libtextclassifier_fbgen_utils_grammar_testing_value",
	}

	SandboxingDenyModuleList = []string{
		"RsBalls-rscript",
		"CtsRsBlasTestCases-rscript",
		"pvmfw_fdt_template_rs",
		"RSTest_v14-rscript",
		"com.android.apex.test.bar_stripped",
		"com.android.apex.test.sharedlibs_secondary_generated",
		"ImageProcessingJB-rscript",
		"RSTest-rscript",
		"BluetoothGeneratedDumpsysBinarySchema_bfbs",
		"TracingVMProtoStub_h",
		"FrontendStub_h",
		"VehicleServerProtoStub_cc",
		"AudioFocusControlProtoStub_cc",
		"AudioFocusControlProtoStub_h",
		"TracingVMProtoStub_cc",
		"VehicleServerProtoStub_h",
		"hidl2aidl_translate_cpp_test_gen_headers",
		"hidl2aidl_translate_cpp_test_gen_src",
		"hidl2aidl_translate_java_test_gen_src",
		"hidl2aidl_translate_ndk_test_gen_headers",
		"hidl2aidl_translate_ndk_test_gen_src",
		"hidl_hash_test_gen",
		"nos_app_avb_service_genc++",
		"nos_app_avb_service_genc++_headers",
		"nos_app_avb_service_genc++_mock",
		"nos_app_identity_service_genc++",
		"nos_app_keymaster_service_genc++",
		"nos_generator_test_service_genc++_headers",
		"nos_generator_test_service_genc++_mock",
		"r8retrace-run-retrace",
		"ltp_config_arm",
		"ltp_config_arm_64_hwasan",
		"ltp_config_arm_lowmem",
		"ltp_config_arm_64",
		"ltp_config_riscv_64",
		"ltp_config_x86_64",
		"vm-tests-tf-lib",
		"hidl_cpp_impl_test_gen-headers",
		"Refocus-rscript",
		"RSTest_v11-rscript",
		"RSTest_v16-rscript",
		"ScriptGroupTest-rscript",
		"ImageProcessing2-rscript",
		"ImageProcessing-rscript",
		"com.android.apex.test.pony_stripped",
		"com.android.apex.test.baz_stripped",
		"com.android.apex.test.foo_stripped",
		"com.android.apex.test.sharedlibs_generated",
		"CtsRenderscriptTestCases-rscript",
		"BlueberryFacadeAndCertGeneratedStub_py",
		"BlueberryFacadeGeneratedStub_cc",
		"BlueberryFacadeGeneratedStub_h",
		"BluetoothGeneratedDumpsysDataSchema_h",
		"FrontendStub_cc",
		"OpenwrtControlServerProto_cc",
		"OpenwrtControlServerProto_h",
		"c2hal_test_genc++",
		"c2hal_test_genc++_headers",
		"hidl2aidl_test_gen_aidl",
		"hidl_error_test_gen",
		"hidl_export_test_gen-headers",
		"hidl_format_test_diff",
		"hidl_hash_version_gen",
		"libbt_topshim_facade_py_proto",
		"nos_app_identity_service_genc++_headers",
		"nos_app_identity_service_genc++_mock",
		"nos_app_keymaster_service_genc++_headers",
		"nos_app_keymaster_service_genc++_mock",
		"nos_app_weaver_service_genc++",
		"nos_app_weaver_service_genc++_headers",
		"nos_app_weaver_service_genc++_mock",
		"nos_generator_test_service_genc++",
	}

	SandboxingDenyPathList = []string{
		"art/test",
		"external/perfetto",
	}
)
