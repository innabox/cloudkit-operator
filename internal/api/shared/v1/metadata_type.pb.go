//
// Copyright (c) 2025 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on
// an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the
// specific language governing permissions and limitations under the License.
//

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.5
// 	protoc        (unknown)
// source: shared/v1/metadata_type.proto

//go:build !protoopaque

package sharedv1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// Metadata common to all kinds of objects.
type Metadata struct {
	state protoimpl.MessageState `protogen:"hybrid.v1"`
	// Time of creation of the object.
	CreationTimestamp *timestamppb.Timestamp `protobuf:"bytes,1,opt,name=creation_timestamp,json=creationTimestamp,proto3" json:"creation_timestamp,omitempty"`
	// Time of deletion of the object.
	DeletionTimestamp *timestamppb.Timestamp `protobuf:"bytes,2,opt,name=deletion_timestamp,json=deletionTimestamp,proto3" json:"deletion_timestamp,omitempty"`
	unknownFields     protoimpl.UnknownFields
	sizeCache         protoimpl.SizeCache
}

func (x *Metadata) Reset() {
	*x = Metadata{}
	mi := &file_shared_v1_metadata_type_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Metadata) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Metadata) ProtoMessage() {}

func (x *Metadata) ProtoReflect() protoreflect.Message {
	mi := &file_shared_v1_metadata_type_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *Metadata) GetCreationTimestamp() *timestamppb.Timestamp {
	if x != nil {
		return x.CreationTimestamp
	}
	return nil
}

func (x *Metadata) GetDeletionTimestamp() *timestamppb.Timestamp {
	if x != nil {
		return x.DeletionTimestamp
	}
	return nil
}

func (x *Metadata) SetCreationTimestamp(v *timestamppb.Timestamp) {
	x.CreationTimestamp = v
}

func (x *Metadata) SetDeletionTimestamp(v *timestamppb.Timestamp) {
	x.DeletionTimestamp = v
}

func (x *Metadata) HasCreationTimestamp() bool {
	if x == nil {
		return false
	}
	return x.CreationTimestamp != nil
}

func (x *Metadata) HasDeletionTimestamp() bool {
	if x == nil {
		return false
	}
	return x.DeletionTimestamp != nil
}

func (x *Metadata) ClearCreationTimestamp() {
	x.CreationTimestamp = nil
}

func (x *Metadata) ClearDeletionTimestamp() {
	x.DeletionTimestamp = nil
}

type Metadata_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	// Time of creation of the object.
	CreationTimestamp *timestamppb.Timestamp
	// Time of deletion of the object.
	DeletionTimestamp *timestamppb.Timestamp
}

func (b0 Metadata_builder) Build() *Metadata {
	m0 := &Metadata{}
	b, x := &b0, m0
	_, _ = b, x
	x.CreationTimestamp = b.CreationTimestamp
	x.DeletionTimestamp = b.DeletionTimestamp
	return m0
}

var File_shared_v1_metadata_type_proto protoreflect.FileDescriptor

var file_shared_v1_metadata_type_proto_rawDesc = string([]byte{
	0x0a, 0x1d, 0x73, 0x68, 0x61, 0x72, 0x65, 0x64, 0x2f, 0x76, 0x31, 0x2f, 0x6d, 0x65, 0x74, 0x61,
	0x64, 0x61, 0x74, 0x61, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x09, 0x73, 0x68, 0x61, 0x72, 0x65, 0x64, 0x2e, 0x76, 0x31, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65,
	0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xa0, 0x01, 0x0a, 0x08,
	0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x12, 0x49, 0x0a, 0x12, 0x63, 0x72, 0x65, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70,
	0x52, 0x11, 0x63, 0x72, 0x65, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74,
	0x61, 0x6d, 0x70, 0x12, 0x49, 0x0a, 0x12, 0x64, 0x65, 0x6c, 0x65, 0x74, 0x69, 0x6f, 0x6e, 0x5f,
	0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x11, 0x64, 0x65, 0x6c,
	0x65, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x42, 0xad,
	0x01, 0x0a, 0x0d, 0x63, 0x6f, 0x6d, 0x2e, 0x73, 0x68, 0x61, 0x72, 0x65, 0x64, 0x2e, 0x76, 0x31,
	0x42, 0x11, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x54, 0x79, 0x70, 0x65, 0x50, 0x72,
	0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x44, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x69, 0x6e, 0x6e, 0x61, 0x62, 0x6f, 0x78, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x6b,
	0x69, 0x74, 0x2d, 0x6f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x6f, 0x72, 0x2f, 0x69, 0x6e, 0x74, 0x65,
	0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x73, 0x68, 0x61, 0x72, 0x65, 0x64, 0x2f,
	0x76, 0x31, 0x3b, 0x73, 0x68, 0x61, 0x72, 0x65, 0x64, 0x76, 0x31, 0xa2, 0x02, 0x03, 0x53, 0x58,
	0x58, 0xaa, 0x02, 0x09, 0x53, 0x68, 0x61, 0x72, 0x65, 0x64, 0x2e, 0x56, 0x31, 0xca, 0x02, 0x09,
	0x53, 0x68, 0x61, 0x72, 0x65, 0x64, 0x5c, 0x56, 0x31, 0xe2, 0x02, 0x15, 0x53, 0x68, 0x61, 0x72,
	0x65, 0x64, 0x5c, 0x56, 0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74,
	0x61, 0xea, 0x02, 0x0a, 0x53, 0x68, 0x61, 0x72, 0x65, 0x64, 0x3a, 0x3a, 0x56, 0x31, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
})

var file_shared_v1_metadata_type_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_shared_v1_metadata_type_proto_goTypes = []any{
	(*Metadata)(nil),              // 0: shared.v1.Metadata
	(*timestamppb.Timestamp)(nil), // 1: google.protobuf.Timestamp
}
var file_shared_v1_metadata_type_proto_depIdxs = []int32{
	1, // 0: shared.v1.Metadata.creation_timestamp:type_name -> google.protobuf.Timestamp
	1, // 1: shared.v1.Metadata.deletion_timestamp:type_name -> google.protobuf.Timestamp
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_shared_v1_metadata_type_proto_init() }
func file_shared_v1_metadata_type_proto_init() {
	if File_shared_v1_metadata_type_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_shared_v1_metadata_type_proto_rawDesc), len(file_shared_v1_metadata_type_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_shared_v1_metadata_type_proto_goTypes,
		DependencyIndexes: file_shared_v1_metadata_type_proto_depIdxs,
		MessageInfos:      file_shared_v1_metadata_type_proto_msgTypes,
	}.Build()
	File_shared_v1_metadata_type_proto = out.File
	file_shared_v1_metadata_type_proto_goTypes = nil
	file_shared_v1_metadata_type_proto_depIdxs = nil
}
