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
// source: build_flags_out.proto

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

type Tracepoint struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Path to declaration or value file relative to $TOP
	Source *string `protobuf:"bytes,1,opt,name=source" json:"source,omitempty"`
	Value  *Value  `protobuf:"bytes,201,opt,name=value" json:"value,omitempty"`
}

func (x *Tracepoint) Reset() {
	*x = Tracepoint{}
	if protoimpl.UnsafeEnabled {
		mi := &file_build_flags_out_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Tracepoint) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Tracepoint) ProtoMessage() {}

func (x *Tracepoint) ProtoReflect() protoreflect.Message {
	mi := &file_build_flags_out_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Tracepoint.ProtoReflect.Descriptor instead.
func (*Tracepoint) Descriptor() ([]byte, []int) {
	return file_build_flags_out_proto_rawDescGZIP(), []int{0}
}

func (x *Tracepoint) GetSource() string {
	if x != nil && x.Source != nil {
		return *x.Source
	}
	return ""
}

func (x *Tracepoint) GetValue() *Value {
	if x != nil {
		return x.Value
	}
	return nil
}

type FlagArtifact struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The original declaration
	FlagDeclaration *FlagDeclaration `protobuf:"bytes,1,opt,name=flag_declaration,json=flagDeclaration" json:"flag_declaration,omitempty"`
	// Value for the flag
	Value *Value `protobuf:"bytes,201,opt,name=value" json:"value,omitempty"`
	// Trace of where the flag value was assigned.
	Traces []*Tracepoint `protobuf:"bytes,8,rep,name=traces" json:"traces,omitempty"`
}

func (x *FlagArtifact) Reset() {
	*x = FlagArtifact{}
	if protoimpl.UnsafeEnabled {
		mi := &file_build_flags_out_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FlagArtifact) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FlagArtifact) ProtoMessage() {}

func (x *FlagArtifact) ProtoReflect() protoreflect.Message {
	mi := &file_build_flags_out_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FlagArtifact.ProtoReflect.Descriptor instead.
func (*FlagArtifact) Descriptor() ([]byte, []int) {
	return file_build_flags_out_proto_rawDescGZIP(), []int{1}
}

func (x *FlagArtifact) GetFlagDeclaration() *FlagDeclaration {
	if x != nil {
		return x.FlagDeclaration
	}
	return nil
}

func (x *FlagArtifact) GetValue() *Value {
	if x != nil {
		return x.Value
	}
	return nil
}

func (x *FlagArtifact) GetTraces() []*Tracepoint {
	if x != nil {
		return x.Traces
	}
	return nil
}

type FlagArtifacts struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The artifacts
	FlagArtifacts []*FlagArtifact `protobuf:"bytes,1,rep,name=flag_artifacts,json=flagArtifacts" json:"flag_artifacts,omitempty"`
}

func (x *FlagArtifacts) Reset() {
	*x = FlagArtifacts{}
	if protoimpl.UnsafeEnabled {
		mi := &file_build_flags_out_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FlagArtifacts) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FlagArtifacts) ProtoMessage() {}

func (x *FlagArtifacts) ProtoReflect() protoreflect.Message {
	mi := &file_build_flags_out_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FlagArtifacts.ProtoReflect.Descriptor instead.
func (*FlagArtifacts) Descriptor() ([]byte, []int) {
	return file_build_flags_out_proto_rawDescGZIP(), []int{2}
}

func (x *FlagArtifacts) GetFlagArtifacts() []*FlagArtifact {
	if x != nil {
		return x.FlagArtifacts
	}
	return nil
}

type ReleaseConfigArtifact struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The name of the release config.
	// See # name for format detail
	Name *string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	// Other names by which this release is known (for example, `next`)
	OtherNames []string `protobuf:"bytes,2,rep,name=other_names,json=otherNames" json:"other_names,omitempty"`
	// The complete set of build flags in this release config, after all
	// inheritance and other processing is complete.
	FlagArtifacts []*FlagArtifact `protobuf:"bytes,3,rep,name=flag_artifacts,json=flagArtifacts" json:"flag_artifacts,omitempty"`
	// The (complete) list of aconfig_value_sets Soong modules to use.
	AconfigValueSets []string `protobuf:"bytes,4,rep,name=aconfig_value_sets,json=aconfigValueSets" json:"aconfig_value_sets,omitempty"`
	// The names of the release_config_artifacts from which we inherited.
	// Included for reference only.
	Inherits []string `protobuf:"bytes,5,rep,name=inherits" json:"inherits,omitempty"`
	// The release config directories used for this config.
	// For example, "build/release".
	Directories []string `protobuf:"bytes,6,rep,name=directories" json:"directories,omitempty"`
}

func (x *ReleaseConfigArtifact) Reset() {
	*x = ReleaseConfigArtifact{}
	if protoimpl.UnsafeEnabled {
		mi := &file_build_flags_out_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReleaseConfigArtifact) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReleaseConfigArtifact) ProtoMessage() {}

func (x *ReleaseConfigArtifact) ProtoReflect() protoreflect.Message {
	mi := &file_build_flags_out_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReleaseConfigArtifact.ProtoReflect.Descriptor instead.
func (*ReleaseConfigArtifact) Descriptor() ([]byte, []int) {
	return file_build_flags_out_proto_rawDescGZIP(), []int{3}
}

func (x *ReleaseConfigArtifact) GetName() string {
	if x != nil && x.Name != nil {
		return *x.Name
	}
	return ""
}

func (x *ReleaseConfigArtifact) GetOtherNames() []string {
	if x != nil {
		return x.OtherNames
	}
	return nil
}

func (x *ReleaseConfigArtifact) GetFlagArtifacts() []*FlagArtifact {
	if x != nil {
		return x.FlagArtifacts
	}
	return nil
}

func (x *ReleaseConfigArtifact) GetAconfigValueSets() []string {
	if x != nil {
		return x.AconfigValueSets
	}
	return nil
}

func (x *ReleaseConfigArtifact) GetInherits() []string {
	if x != nil {
		return x.Inherits
	}
	return nil
}

func (x *ReleaseConfigArtifact) GetDirectories() []string {
	if x != nil {
		return x.Directories
	}
	return nil
}

type ReleaseConfigsArtifact struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The active release config for this build.
	ReleaseConfig *ReleaseConfigArtifact `protobuf:"bytes,1,opt,name=release_config,json=releaseConfig" json:"release_config,omitempty"`
	// All other release configs defined for this TARGET_PRODUCT.
	OtherReleaseConfigs []*ReleaseConfigArtifact `protobuf:"bytes,2,rep,name=other_release_configs,json=otherReleaseConfigs" json:"other_release_configs,omitempty"`
	// Map of release_config_artifact.directories to release_config_map message.
	ReleaseConfigMapsMap map[string]*ReleaseConfigMap `protobuf:"bytes,3,rep,name=release_config_maps_map,json=releaseConfigMapsMap" json:"release_config_maps_map,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

func (x *ReleaseConfigsArtifact) Reset() {
	*x = ReleaseConfigsArtifact{}
	if protoimpl.UnsafeEnabled {
		mi := &file_build_flags_out_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReleaseConfigsArtifact) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReleaseConfigsArtifact) ProtoMessage() {}

func (x *ReleaseConfigsArtifact) ProtoReflect() protoreflect.Message {
	mi := &file_build_flags_out_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReleaseConfigsArtifact.ProtoReflect.Descriptor instead.
func (*ReleaseConfigsArtifact) Descriptor() ([]byte, []int) {
	return file_build_flags_out_proto_rawDescGZIP(), []int{4}
}

func (x *ReleaseConfigsArtifact) GetReleaseConfig() *ReleaseConfigArtifact {
	if x != nil {
		return x.ReleaseConfig
	}
	return nil
}

func (x *ReleaseConfigsArtifact) GetOtherReleaseConfigs() []*ReleaseConfigArtifact {
	if x != nil {
		return x.OtherReleaseConfigs
	}
	return nil
}

func (x *ReleaseConfigsArtifact) GetReleaseConfigMapsMap() map[string]*ReleaseConfigMap {
	if x != nil {
		return x.ReleaseConfigMapsMap
	}
	return nil
}

var File_build_flags_out_proto protoreflect.FileDescriptor

var file_build_flags_out_proto_rawDesc = []byte{
	0x0a, 0x15, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x5f, 0x66, 0x6c, 0x61, 0x67, 0x73, 0x5f, 0x6f, 0x75,
	0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1c, 0x61, 0x6e, 0x64, 0x72, 0x6f, 0x69, 0x64,
	0x2e, 0x72, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x5f,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x15, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x5f, 0x66, 0x6c, 0x61,
	0x67, 0x73, 0x5f, 0x73, 0x72, 0x63, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x60, 0x0a, 0x0a,
	0x74, 0x72, 0x61, 0x63, 0x65, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x6f,
	0x75, 0x72, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x6f, 0x75, 0x72,
	0x63, 0x65, 0x12, 0x3a, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0xc9, 0x01, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x23, 0x2e, 0x61, 0x6e, 0x64, 0x72, 0x6f, 0x69, 0x64, 0x2e, 0x72, 0x65, 0x6c,
	0x65, 0x61, 0x73, 0x65, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x5f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x2e, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x22, 0xe8,
	0x01, 0x0a, 0x0d, 0x66, 0x6c, 0x61, 0x67, 0x5f, 0x61, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74,
	0x12, 0x59, 0x0a, 0x10, 0x66, 0x6c, 0x61, 0x67, 0x5f, 0x64, 0x65, 0x63, 0x6c, 0x61, 0x72, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2e, 0x2e, 0x61, 0x6e, 0x64,
	0x72, 0x6f, 0x69, 0x64, 0x2e, 0x72, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x5f, 0x63, 0x6f, 0x6e,
	0x66, 0x69, 0x67, 0x5f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x66, 0x6c, 0x61, 0x67, 0x5f, 0x64,
	0x65, 0x63, 0x6c, 0x61, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0f, 0x66, 0x6c, 0x61, 0x67,
	0x44, 0x65, 0x63, 0x6c, 0x61, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x3a, 0x0a, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x18, 0xc9, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x61, 0x6e,
	0x64, 0x72, 0x6f, 0x69, 0x64, 0x2e, 0x72, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x5f, 0x63, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x5f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x76, 0x61, 0x6c, 0x75, 0x65,
	0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x12, 0x40, 0x0a, 0x06, 0x74, 0x72, 0x61, 0x63, 0x65,
	0x73, 0x18, 0x08, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x28, 0x2e, 0x61, 0x6e, 0x64, 0x72, 0x6f, 0x69,
	0x64, 0x2e, 0x72, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67,
	0x5f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x74, 0x72, 0x61, 0x63, 0x65, 0x70, 0x6f, 0x69, 0x6e,
	0x74, 0x52, 0x06, 0x74, 0x72, 0x61, 0x63, 0x65, 0x73, 0x22, 0x64, 0x0a, 0x0e, 0x66, 0x6c, 0x61,
	0x67, 0x5f, 0x61, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74, 0x73, 0x12, 0x52, 0x0a, 0x0e, 0x66,
	0x6c, 0x61, 0x67, 0x5f, 0x61, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74, 0x73, 0x18, 0x01, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x2b, 0x2e, 0x61, 0x6e, 0x64, 0x72, 0x6f, 0x69, 0x64, 0x2e, 0x72, 0x65,
	0x6c, 0x65, 0x61, 0x73, 0x65, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x5f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2e, 0x66, 0x6c, 0x61, 0x67, 0x5f, 0x61, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74,
	0x52, 0x0d, 0x66, 0x6c, 0x61, 0x67, 0x41, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74, 0x73, 0x22,
	0x8e, 0x02, 0x0a, 0x17, 0x72, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x5f, 0x63, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x5f, 0x61, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x6e,
	0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12,
	0x1f, 0x0a, 0x0b, 0x6f, 0x74, 0x68, 0x65, 0x72, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x18, 0x02,
	0x20, 0x03, 0x28, 0x09, 0x52, 0x0a, 0x6f, 0x74, 0x68, 0x65, 0x72, 0x4e, 0x61, 0x6d, 0x65, 0x73,
	0x12, 0x52, 0x0a, 0x0e, 0x66, 0x6c, 0x61, 0x67, 0x5f, 0x61, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63,
	0x74, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2b, 0x2e, 0x61, 0x6e, 0x64, 0x72, 0x6f,
	0x69, 0x64, 0x2e, 0x72, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x5f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x66, 0x6c, 0x61, 0x67, 0x5f, 0x61, 0x72, 0x74,
	0x69, 0x66, 0x61, 0x63, 0x74, 0x52, 0x0d, 0x66, 0x6c, 0x61, 0x67, 0x41, 0x72, 0x74, 0x69, 0x66,
	0x61, 0x63, 0x74, 0x73, 0x12, 0x2c, 0x0a, 0x12, 0x61, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x5f,
	0x76, 0x61, 0x6c, 0x75, 0x65, 0x5f, 0x73, 0x65, 0x74, 0x73, 0x18, 0x04, 0x20, 0x03, 0x28, 0x09,
	0x52, 0x10, 0x61, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x53, 0x65,
	0x74, 0x73, 0x12, 0x1a, 0x0a, 0x08, 0x69, 0x6e, 0x68, 0x65, 0x72, 0x69, 0x74, 0x73, 0x18, 0x05,
	0x20, 0x03, 0x28, 0x09, 0x52, 0x08, 0x69, 0x6e, 0x68, 0x65, 0x72, 0x69, 0x74, 0x73, 0x12, 0x20,
	0x0a, 0x0b, 0x64, 0x69, 0x72, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x69, 0x65, 0x73, 0x18, 0x06, 0x20,
	0x03, 0x28, 0x09, 0x52, 0x0b, 0x64, 0x69, 0x72, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x69, 0x65, 0x73,
	0x22, 0xe8, 0x03, 0x0a, 0x18, 0x72, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x5f, 0x63, 0x6f, 0x6e,
	0x66, 0x69, 0x67, 0x73, 0x5f, 0x61, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74, 0x12, 0x5c, 0x0a,
	0x0e, 0x72, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x35, 0x2e, 0x61, 0x6e, 0x64, 0x72, 0x6f, 0x69, 0x64, 0x2e,
	0x72, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x5f, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x72, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x5f, 0x63, 0x6f, 0x6e,
	0x66, 0x69, 0x67, 0x5f, 0x61, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74, 0x52, 0x0d, 0x72, 0x65,
	0x6c, 0x65, 0x61, 0x73, 0x65, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x69, 0x0a, 0x15, 0x6f,
	0x74, 0x68, 0x65, 0x72, 0x5f, 0x72, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x5f, 0x63, 0x6f, 0x6e,
	0x66, 0x69, 0x67, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x35, 0x2e, 0x61, 0x6e, 0x64,
	0x72, 0x6f, 0x69, 0x64, 0x2e, 0x72, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x5f, 0x63, 0x6f, 0x6e,
	0x66, 0x69, 0x67, 0x5f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x72, 0x65, 0x6c, 0x65, 0x61, 0x73,
	0x65, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x5f, 0x61, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63,
	0x74, 0x52, 0x13, 0x6f, 0x74, 0x68, 0x65, 0x72, 0x52, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x43,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x73, 0x12, 0x87, 0x01, 0x0a, 0x17, 0x72, 0x65, 0x6c, 0x65, 0x61,
	0x73, 0x65, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x5f, 0x6d, 0x61, 0x70, 0x73, 0x5f, 0x6d,
	0x61, 0x70, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x50, 0x2e, 0x61, 0x6e, 0x64, 0x72, 0x6f,
	0x69, 0x64, 0x2e, 0x72, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x5f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x72, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x5f,
	0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x73, 0x5f, 0x61, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74,
	0x2e, 0x52, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x4d, 0x61,
	0x70, 0x73, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x14, 0x72, 0x65, 0x6c, 0x65,
	0x61, 0x73, 0x65, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x4d, 0x61, 0x70, 0x73, 0x4d, 0x61, 0x70,
	0x1a, 0x79, 0x0a, 0x19, 0x52, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x43, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x4d, 0x61, 0x70, 0x73, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a,
	0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12,
	0x46, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x30,
	0x2e, 0x61, 0x6e, 0x64, 0x72, 0x6f, 0x69, 0x64, 0x2e, 0x72, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65,
	0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x5f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x72, 0x65,
	0x6c, 0x65, 0x61, 0x73, 0x65, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x5f, 0x6d, 0x61, 0x70,
	0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x42, 0x33, 0x5a, 0x31, 0x61,
	0x6e, 0x64, 0x72, 0x6f, 0x69, 0x64, 0x2f, 0x73, 0x6f, 0x6f, 0x6e, 0x67, 0x2f, 0x72, 0x65, 0x6c,
	0x65, 0x61, 0x73, 0x65, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2f, 0x72, 0x65, 0x6c, 0x65,
	0x61, 0x73, 0x65, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x5f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
}

var (
	file_build_flags_out_proto_rawDescOnce sync.Once
	file_build_flags_out_proto_rawDescData = file_build_flags_out_proto_rawDesc
)

func file_build_flags_out_proto_rawDescGZIP() []byte {
	file_build_flags_out_proto_rawDescOnce.Do(func() {
		file_build_flags_out_proto_rawDescData = protoimpl.X.CompressGZIP(file_build_flags_out_proto_rawDescData)
	})
	return file_build_flags_out_proto_rawDescData
}

var file_build_flags_out_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_build_flags_out_proto_goTypes = []interface{}{
	(*Tracepoint)(nil),             // 0: android.release_config_proto.tracepoint
	(*FlagArtifact)(nil),           // 1: android.release_config_proto.flag_artifact
	(*FlagArtifacts)(nil),          // 2: android.release_config_proto.flag_artifacts
	(*ReleaseConfigArtifact)(nil),  // 3: android.release_config_proto.release_config_artifact
	(*ReleaseConfigsArtifact)(nil), // 4: android.release_config_proto.release_configs_artifact
	nil,                            // 5: android.release_config_proto.release_configs_artifact.ReleaseConfigMapsMapEntry
	(*Value)(nil),                  // 6: android.release_config_proto.value
	(*FlagDeclaration)(nil),        // 7: android.release_config_proto.flag_declaration
	(*ReleaseConfigMap)(nil),       // 8: android.release_config_proto.release_config_map
}
var file_build_flags_out_proto_depIdxs = []int32{
	6,  // 0: android.release_config_proto.tracepoint.value:type_name -> android.release_config_proto.value
	7,  // 1: android.release_config_proto.flag_artifact.flag_declaration:type_name -> android.release_config_proto.flag_declaration
	6,  // 2: android.release_config_proto.flag_artifact.value:type_name -> android.release_config_proto.value
	0,  // 3: android.release_config_proto.flag_artifact.traces:type_name -> android.release_config_proto.tracepoint
	1,  // 4: android.release_config_proto.flag_artifacts.flag_artifacts:type_name -> android.release_config_proto.flag_artifact
	1,  // 5: android.release_config_proto.release_config_artifact.flag_artifacts:type_name -> android.release_config_proto.flag_artifact
	3,  // 6: android.release_config_proto.release_configs_artifact.release_config:type_name -> android.release_config_proto.release_config_artifact
	3,  // 7: android.release_config_proto.release_configs_artifact.other_release_configs:type_name -> android.release_config_proto.release_config_artifact
	5,  // 8: android.release_config_proto.release_configs_artifact.release_config_maps_map:type_name -> android.release_config_proto.release_configs_artifact.ReleaseConfigMapsMapEntry
	8,  // 9: android.release_config_proto.release_configs_artifact.ReleaseConfigMapsMapEntry.value:type_name -> android.release_config_proto.release_config_map
	10, // [10:10] is the sub-list for method output_type
	10, // [10:10] is the sub-list for method input_type
	10, // [10:10] is the sub-list for extension type_name
	10, // [10:10] is the sub-list for extension extendee
	0,  // [0:10] is the sub-list for field type_name
}

func init() { file_build_flags_out_proto_init() }
func file_build_flags_out_proto_init() {
	if File_build_flags_out_proto != nil {
		return
	}
	file_build_flags_src_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_build_flags_out_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Tracepoint); i {
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
		file_build_flags_out_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*FlagArtifact); i {
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
		file_build_flags_out_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*FlagArtifacts); i {
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
		file_build_flags_out_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReleaseConfigArtifact); i {
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
		file_build_flags_out_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReleaseConfigsArtifact); i {
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
			RawDescriptor: file_build_flags_out_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_build_flags_out_proto_goTypes,
		DependencyIndexes: file_build_flags_out_proto_depIdxs,
		MessageInfos:      file_build_flags_out_proto_msgTypes,
	}.Build()
	File_build_flags_out_proto = out.File
	file_build_flags_out_proto_rawDesc = nil
	file_build_flags_out_proto_goTypes = nil
	file_build_flags_out_proto_depIdxs = nil
}
