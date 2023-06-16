// Copyright 2022 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        v3.21.12
// source: bazel_metrics.proto

package bazel_metrics_proto

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type BazelMetrics struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	PhaseTimings []*PhaseTiming `protobuf:"bytes,1,rep,name=phase_timings,json=phaseTimings,proto3" json:"phase_timings,omitempty"`
	Total        *int64         `protobuf:"varint,2,opt,name=total,proto3,oneof" json:"total,omitempty"`
	ExitCode     *int32         `protobuf:"varint,3,opt,name=exit_code,json=exitCode,proto3,oneof" json:"exit_code,omitempty"`
	SpongeId     *string        `protobuf:"bytes,4,opt,name=sponge_id,json=spongeId,proto3,oneof" json:"sponge_id,omitempty"`
}

func (x *BazelMetrics) Reset() {
	*x = BazelMetrics{}
	if protoimpl.UnsafeEnabled {
		mi := &file_bazel_metrics_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BazelMetrics) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BazelMetrics) ProtoMessage() {}

func (x *BazelMetrics) ProtoReflect() protoreflect.Message {
	mi := &file_bazel_metrics_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BazelMetrics.ProtoReflect.Descriptor instead.
func (*BazelMetrics) Descriptor() ([]byte, []int) {
	return file_bazel_metrics_proto_rawDescGZIP(), []int{0}
}

func (x *BazelMetrics) GetPhaseTimings() []*PhaseTiming {
	if x != nil {
		return x.PhaseTimings
	}
	return nil
}

func (x *BazelMetrics) GetTotal() int64 {
	if x != nil && x.Total != nil {
		return *x.Total
	}
	return 0
}

func (x *BazelMetrics) GetExitCode() int32 {
	if x != nil && x.ExitCode != nil {
		return *x.ExitCode
	}
	return 0
}

func (x *BazelMetrics) GetSpongeId() string {
	if x != nil && x.SpongeId != nil {
		return *x.SpongeId
	}
	return ""
}

type PhaseTiming struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// E.g. "execution", "analysis", "launch"
	PhaseName     *string `protobuf:"bytes,1,opt,name=phase_name,json=phaseName,proto3,oneof" json:"phase_name,omitempty"`
	DurationNanos *int64  `protobuf:"varint,2,opt,name=duration_nanos,json=durationNanos,proto3,oneof" json:"duration_nanos,omitempty"`
	// What portion of the build time this phase took, with ten-thousandths precision.
	// E.g., 1111 = 11.11%, 111 = 1.11%
	PortionOfBuildTime *int32 `protobuf:"varint,3,opt,name=portion_of_build_time,json=portionOfBuildTime,proto3,oneof" json:"portion_of_build_time,omitempty"`
}

func (x *PhaseTiming) Reset() {
	*x = PhaseTiming{}
	if protoimpl.UnsafeEnabled {
		mi := &file_bazel_metrics_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PhaseTiming) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PhaseTiming) ProtoMessage() {}

func (x *PhaseTiming) ProtoReflect() protoreflect.Message {
	mi := &file_bazel_metrics_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PhaseTiming.ProtoReflect.Descriptor instead.
func (*PhaseTiming) Descriptor() ([]byte, []int) {
	return file_bazel_metrics_proto_rawDescGZIP(), []int{1}
}

func (x *PhaseTiming) GetPhaseName() string {
	if x != nil && x.PhaseName != nil {
		return *x.PhaseName
	}
	return ""
}

func (x *PhaseTiming) GetDurationNanos() int64 {
	if x != nil && x.DurationNanos != nil {
		return *x.DurationNanos
	}
	return 0
}

func (x *PhaseTiming) GetPortionOfBuildTime() int32 {
	if x != nil && x.PortionOfBuildTime != nil {
		return *x.PortionOfBuildTime
	}
	return 0
}

var File_bazel_metrics_proto protoreflect.FileDescriptor

var file_bazel_metrics_proto_rawDesc = []byte{
	0x0a, 0x13, 0x62, 0x61, 0x7a, 0x65, 0x6c, 0x5f, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x19, 0x73, 0x6f, 0x6f, 0x6e, 0x67, 0x5f, 0x62, 0x75, 0x69,
	0x6c, 0x64, 0x5f, 0x62, 0x61, 0x7a, 0x65, 0x6c, 0x5f, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73,
	0x22, 0xe0, 0x01, 0x0a, 0x0c, 0x42, 0x61, 0x7a, 0x65, 0x6c, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63,
	0x73, 0x12, 0x4b, 0x0a, 0x0d, 0x70, 0x68, 0x61, 0x73, 0x65, 0x5f, 0x74, 0x69, 0x6d, 0x69, 0x6e,
	0x67, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x26, 0x2e, 0x73, 0x6f, 0x6f, 0x6e, 0x67,
	0x5f, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x5f, 0x62, 0x61, 0x7a, 0x65, 0x6c, 0x5f, 0x6d, 0x65, 0x74,
	0x72, 0x69, 0x63, 0x73, 0x2e, 0x50, 0x68, 0x61, 0x73, 0x65, 0x54, 0x69, 0x6d, 0x69, 0x6e, 0x67,
	0x52, 0x0c, 0x70, 0x68, 0x61, 0x73, 0x65, 0x54, 0x69, 0x6d, 0x69, 0x6e, 0x67, 0x73, 0x12, 0x19,
	0x0a, 0x05, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x48, 0x00, 0x52,
	0x05, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x88, 0x01, 0x01, 0x12, 0x20, 0x0a, 0x09, 0x65, 0x78, 0x69,
	0x74, 0x5f, 0x63, 0x6f, 0x64, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x48, 0x01, 0x52, 0x08,
	0x65, 0x78, 0x69, 0x74, 0x43, 0x6f, 0x64, 0x65, 0x88, 0x01, 0x01, 0x12, 0x20, 0x0a, 0x09, 0x73,
	0x70, 0x6f, 0x6e, 0x67, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x48, 0x02,
	0x52, 0x08, 0x73, 0x70, 0x6f, 0x6e, 0x67, 0x65, 0x49, 0x64, 0x88, 0x01, 0x01, 0x42, 0x08, 0x0a,
	0x06, 0x5f, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x42, 0x0c, 0x0a, 0x0a, 0x5f, 0x65, 0x78, 0x69, 0x74,
	0x5f, 0x63, 0x6f, 0x64, 0x65, 0x42, 0x0c, 0x0a, 0x0a, 0x5f, 0x73, 0x70, 0x6f, 0x6e, 0x67, 0x65,
	0x5f, 0x69, 0x64, 0x22, 0xd1, 0x01, 0x0a, 0x0b, 0x50, 0x68, 0x61, 0x73, 0x65, 0x54, 0x69, 0x6d,
	0x69, 0x6e, 0x67, 0x12, 0x22, 0x0a, 0x0a, 0x70, 0x68, 0x61, 0x73, 0x65, 0x5f, 0x6e, 0x61, 0x6d,
	0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x09, 0x70, 0x68, 0x61, 0x73, 0x65,
	0x4e, 0x61, 0x6d, 0x65, 0x88, 0x01, 0x01, 0x12, 0x2a, 0x0a, 0x0e, 0x64, 0x75, 0x72, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x5f, 0x6e, 0x61, 0x6e, 0x6f, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x48,
	0x01, 0x52, 0x0d, 0x64, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x4e, 0x61, 0x6e, 0x6f, 0x73,
	0x88, 0x01, 0x01, 0x12, 0x36, 0x0a, 0x15, 0x70, 0x6f, 0x72, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x6f,
	0x66, 0x5f, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x05, 0x48, 0x02, 0x52, 0x12, 0x70, 0x6f, 0x72, 0x74, 0x69, 0x6f, 0x6e, 0x4f, 0x66, 0x42,
	0x75, 0x69, 0x6c, 0x64, 0x54, 0x69, 0x6d, 0x65, 0x88, 0x01, 0x01, 0x42, 0x0d, 0x0a, 0x0b, 0x5f,
	0x70, 0x68, 0x61, 0x73, 0x65, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x42, 0x11, 0x0a, 0x0f, 0x5f, 0x64,
	0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x6e, 0x61, 0x6e, 0x6f, 0x73, 0x42, 0x18, 0x0a,
	0x16, 0x5f, 0x70, 0x6f, 0x72, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x6f, 0x66, 0x5f, 0x62, 0x75, 0x69,
	0x6c, 0x64, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x42, 0x2e, 0x5a, 0x2c, 0x61, 0x6e, 0x64, 0x72, 0x6f,
	0x69, 0x64, 0x2f, 0x73, 0x6f, 0x6f, 0x6e, 0x67, 0x2f, 0x75, 0x69, 0x2f, 0x6d, 0x65, 0x74, 0x72,
	0x69, 0x63, 0x73, 0x2f, 0x62, 0x61, 0x7a, 0x65, 0x6c, 0x5f, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63,
	0x73, 0x5f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_bazel_metrics_proto_rawDescOnce sync.Once
	file_bazel_metrics_proto_rawDescData = file_bazel_metrics_proto_rawDesc
)

func file_bazel_metrics_proto_rawDescGZIP() []byte {
	file_bazel_metrics_proto_rawDescOnce.Do(func() {
		file_bazel_metrics_proto_rawDescData = protoimpl.X.CompressGZIP(file_bazel_metrics_proto_rawDescData)
	})
	return file_bazel_metrics_proto_rawDescData
}

var file_bazel_metrics_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_bazel_metrics_proto_goTypes = []interface{}{
	(*BazelMetrics)(nil), // 0: soong_build_bazel_metrics.BazelMetrics
	(*PhaseTiming)(nil),  // 1: soong_build_bazel_metrics.PhaseTiming
}
var file_bazel_metrics_proto_depIdxs = []int32{
	1, // 0: soong_build_bazel_metrics.BazelMetrics.phase_timings:type_name -> soong_build_bazel_metrics.PhaseTiming
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_bazel_metrics_proto_init() }
func file_bazel_metrics_proto_init() {
	if File_bazel_metrics_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_bazel_metrics_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BazelMetrics); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_bazel_metrics_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PhaseTiming); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_bazel_metrics_proto_msgTypes[0].OneofWrappers = []interface{}{}
	file_bazel_metrics_proto_msgTypes[1].OneofWrappers = []interface{}{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_bazel_metrics_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_bazel_metrics_proto_goTypes,
		DependencyIndexes: file_bazel_metrics_proto_depIdxs,
		MessageInfos:      file_bazel_metrics_proto_msgTypes,
	}.Build()
	File_bazel_metrics_proto = out.File
	file_bazel_metrics_proto_rawDesc = nil
	file_bazel_metrics_proto_goTypes = nil
	file_bazel_metrics_proto_depIdxs = nil
}
