//
// Copyright (C) 2025 The Android Open-Source Project
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
// 	protoc-gen-go v1.33.0
// 	protoc        v3.21.12
// source: release_configs_contributions.proto

package release_config_proto

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

type ReleaseConfigContributionsArtifact struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The name of the release config.
	Name *string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	// The release config contribution directories that may contribute to this
	// release config.
	ContributingDirectories []string `protobuf:"bytes,2,rep,name=contributing_directories,json=contributingDirectories" json:"contributing_directories,omitempty"`
}

func (x *ReleaseConfigContributionsArtifact) Reset() {
	*x = ReleaseConfigContributionsArtifact{}
	if protoimpl.UnsafeEnabled {
		mi := &file_release_configs_contributions_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReleaseConfigContributionsArtifact) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReleaseConfigContributionsArtifact) ProtoMessage() {}

func (x *ReleaseConfigContributionsArtifact) ProtoReflect() protoreflect.Message {
	mi := &file_release_configs_contributions_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReleaseConfigContributionsArtifact.ProtoReflect.Descriptor instead.
func (*ReleaseConfigContributionsArtifact) Descriptor() ([]byte, []int) {
	return file_release_configs_contributions_proto_rawDescGZIP(), []int{0}
}

func (x *ReleaseConfigContributionsArtifact) GetName() string {
	if x != nil && x.Name != nil {
		return *x.Name
	}
	return ""
}

func (x *ReleaseConfigContributionsArtifact) GetContributingDirectories() []string {
	if x != nil {
		return x.ContributingDirectories
	}
	return nil
}

type ReleaseConfigContributionsArtifacts struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The artifacts
	ReleaseConfigContributionsArtifactList []*ReleaseConfigContributionsArtifact `protobuf:"bytes,1,rep,name=release_config_contributions_artifact_list,json=releaseConfigContributionsArtifactList" json:"release_config_contributions_artifact_list,omitempty"`
}

func (x *ReleaseConfigContributionsArtifacts) Reset() {
	*x = ReleaseConfigContributionsArtifacts{}
	if protoimpl.UnsafeEnabled {
		mi := &file_release_configs_contributions_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReleaseConfigContributionsArtifacts) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReleaseConfigContributionsArtifacts) ProtoMessage() {}

func (x *ReleaseConfigContributionsArtifacts) ProtoReflect() protoreflect.Message {
	mi := &file_release_configs_contributions_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReleaseConfigContributionsArtifacts.ProtoReflect.Descriptor instead.
func (*ReleaseConfigContributionsArtifacts) Descriptor() ([]byte, []int) {
	return file_release_configs_contributions_proto_rawDescGZIP(), []int{1}
}

func (x *ReleaseConfigContributionsArtifacts) GetReleaseConfigContributionsArtifactList() []*ReleaseConfigContributionsArtifact {
	if x != nil {
		return x.ReleaseConfigContributionsArtifactList
	}
	return nil
}

var File_release_configs_contributions_proto protoreflect.FileDescriptor

var file_release_configs_contributions_proto_rawDesc = []byte{
	0x0a, 0x23, 0x72, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67,
	0x73, 0x5f, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1c, 0x61, 0x6e, 0x64, 0x72, 0x6f, 0x69, 0x64, 0x2e, 0x72,
	0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x5f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x22, 0x73, 0x0a, 0x22, 0x52, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x43, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x69, 0x6f, 0x6e,
	0x73, 0x41, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d,
	0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x39, 0x0a,
	0x18, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x69, 0x6e, 0x67, 0x5f, 0x64, 0x69,
	0x72, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x69, 0x65, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09, 0x52,
	0x17, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x69, 0x6e, 0x67, 0x44, 0x69, 0x72,
	0x65, 0x63, 0x74, 0x6f, 0x72, 0x69, 0x65, 0x73, 0x22, 0xc4, 0x01, 0x0a, 0x23, 0x52, 0x65, 0x6c,
	0x65, 0x61, 0x73, 0x65, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x69,
	0x62, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x41, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74, 0x73,
	0x12, 0x9c, 0x01, 0x0a, 0x2a, 0x72, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x5f, 0x63, 0x6f, 0x6e,
	0x66, 0x69, 0x67, 0x5f, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x69, 0x6f, 0x6e,
	0x73, 0x5f, 0x61, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74, 0x5f, 0x6c, 0x69, 0x73, 0x74, 0x18,
	0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x40, 0x2e, 0x61, 0x6e, 0x64, 0x72, 0x6f, 0x69, 0x64, 0x2e,
	0x72, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x5f, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x52, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x43, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x41,
	0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74, 0x52, 0x26, 0x72, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65,
	0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x69,
	0x6f, 0x6e, 0x73, 0x41, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74, 0x4c, 0x69, 0x73, 0x74, 0x42,
	0x33, 0x5a, 0x31, 0x61, 0x6e, 0x64, 0x72, 0x6f, 0x69, 0x64, 0x2f, 0x73, 0x6f, 0x6f, 0x6e, 0x67,
	0x2f, 0x72, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2f,
	0x72, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x5f, 0x70,
	0x72, 0x6f, 0x74, 0x6f,
}

var (
	file_release_configs_contributions_proto_rawDescOnce sync.Once
	file_release_configs_contributions_proto_rawDescData = file_release_configs_contributions_proto_rawDesc
)

func file_release_configs_contributions_proto_rawDescGZIP() []byte {
	file_release_configs_contributions_proto_rawDescOnce.Do(func() {
		file_release_configs_contributions_proto_rawDescData = protoimpl.X.CompressGZIP(file_release_configs_contributions_proto_rawDescData)
	})
	return file_release_configs_contributions_proto_rawDescData
}

var file_release_configs_contributions_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_release_configs_contributions_proto_goTypes = []interface{}{
	(*ReleaseConfigContributionsArtifact)(nil),  // 0: android.release_config_proto.ReleaseConfigContributionsArtifact
	(*ReleaseConfigContributionsArtifacts)(nil), // 1: android.release_config_proto.ReleaseConfigContributionsArtifacts
}
var file_release_configs_contributions_proto_depIdxs = []int32{
	0, // 0: android.release_config_proto.ReleaseConfigContributionsArtifacts.release_config_contributions_artifact_list:type_name -> android.release_config_proto.ReleaseConfigContributionsArtifact
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_release_configs_contributions_proto_init() }
func file_release_configs_contributions_proto_init() {
	if File_release_configs_contributions_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_release_configs_contributions_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReleaseConfigContributionsArtifact); i {
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
		file_release_configs_contributions_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReleaseConfigContributionsArtifacts); i {
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
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_release_configs_contributions_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_release_configs_contributions_proto_goTypes,
		DependencyIndexes: file_release_configs_contributions_proto_depIdxs,
		MessageInfos:      file_release_configs_contributions_proto_msgTypes,
	}.Build()
	File_release_configs_contributions_proto = out.File
	file_release_configs_contributions_proto_rawDesc = nil
	file_release_configs_contributions_proto_goTypes = nil
	file_release_configs_contributions_proto_depIdxs = nil
}
