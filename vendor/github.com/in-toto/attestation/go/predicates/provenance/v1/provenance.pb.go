// Keep in sync with schema at https://github.com/slsa-framework/slsa/blob/main/docs/provenance/schema/v1/provenance.proto

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.33.0
// 	protoc        v4.24.4
// source: in_toto_attestation/predicates/provenance/v1/provenance.proto

package v1

import (
	v1 "github.com/in-toto/attestation/go/v1"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	structpb "google.golang.org/protobuf/types/known/structpb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// Proto representation of predicate type https://slsa.dev/provenance/v1
// Validation of all fields is left to the users of this proto.
type Provenance struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	BuildDefinition *BuildDefinition `protobuf:"bytes,1,opt,name=build_definition,json=buildDefinition,proto3" json:"build_definition,omitempty"`
	RunDetails      *RunDetails      `protobuf:"bytes,2,opt,name=run_details,json=runDetails,proto3" json:"run_details,omitempty"`
}

func (x *Provenance) Reset() {
	*x = Provenance{}
	if protoimpl.UnsafeEnabled {
		mi := &file_in_toto_attestation_predicates_provenance_v1_provenance_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Provenance) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Provenance) ProtoMessage() {}

func (x *Provenance) ProtoReflect() protoreflect.Message {
	mi := &file_in_toto_attestation_predicates_provenance_v1_provenance_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Provenance.ProtoReflect.Descriptor instead.
func (*Provenance) Descriptor() ([]byte, []int) {
	return file_in_toto_attestation_predicates_provenance_v1_provenance_proto_rawDescGZIP(), []int{0}
}

func (x *Provenance) GetBuildDefinition() *BuildDefinition {
	if x != nil {
		return x.BuildDefinition
	}
	return nil
}

func (x *Provenance) GetRunDetails() *RunDetails {
	if x != nil {
		return x.RunDetails
	}
	return nil
}

type BuildDefinition struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	BuildType            string                   `protobuf:"bytes,1,opt,name=build_type,json=buildType,proto3" json:"build_type,omitempty"`
	ExternalParameters   *structpb.Struct         `protobuf:"bytes,2,opt,name=external_parameters,json=externalParameters,proto3" json:"external_parameters,omitempty"`
	InternalParameters   *structpb.Struct         `protobuf:"bytes,3,opt,name=internal_parameters,json=internalParameters,proto3" json:"internal_parameters,omitempty"`
	ResolvedDependencies []*v1.ResourceDescriptor `protobuf:"bytes,4,rep,name=resolved_dependencies,json=resolvedDependencies,proto3" json:"resolved_dependencies,omitempty"`
}

func (x *BuildDefinition) Reset() {
	*x = BuildDefinition{}
	if protoimpl.UnsafeEnabled {
		mi := &file_in_toto_attestation_predicates_provenance_v1_provenance_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BuildDefinition) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BuildDefinition) ProtoMessage() {}

func (x *BuildDefinition) ProtoReflect() protoreflect.Message {
	mi := &file_in_toto_attestation_predicates_provenance_v1_provenance_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BuildDefinition.ProtoReflect.Descriptor instead.
func (*BuildDefinition) Descriptor() ([]byte, []int) {
	return file_in_toto_attestation_predicates_provenance_v1_provenance_proto_rawDescGZIP(), []int{1}
}

func (x *BuildDefinition) GetBuildType() string {
	if x != nil {
		return x.BuildType
	}
	return ""
}

func (x *BuildDefinition) GetExternalParameters() *structpb.Struct {
	if x != nil {
		return x.ExternalParameters
	}
	return nil
}

func (x *BuildDefinition) GetInternalParameters() *structpb.Struct {
	if x != nil {
		return x.InternalParameters
	}
	return nil
}

func (x *BuildDefinition) GetResolvedDependencies() []*v1.ResourceDescriptor {
	if x != nil {
		return x.ResolvedDependencies
	}
	return nil
}

type RunDetails struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Builder    *Builder                 `protobuf:"bytes,1,opt,name=builder,proto3" json:"builder,omitempty"`
	Metadata   *BuildMetadata           `protobuf:"bytes,2,opt,name=metadata,proto3" json:"metadata,omitempty"`
	Byproducts []*v1.ResourceDescriptor `protobuf:"bytes,3,rep,name=byproducts,proto3" json:"byproducts,omitempty"`
}

func (x *RunDetails) Reset() {
	*x = RunDetails{}
	if protoimpl.UnsafeEnabled {
		mi := &file_in_toto_attestation_predicates_provenance_v1_provenance_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RunDetails) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RunDetails) ProtoMessage() {}

func (x *RunDetails) ProtoReflect() protoreflect.Message {
	mi := &file_in_toto_attestation_predicates_provenance_v1_provenance_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RunDetails.ProtoReflect.Descriptor instead.
func (*RunDetails) Descriptor() ([]byte, []int) {
	return file_in_toto_attestation_predicates_provenance_v1_provenance_proto_rawDescGZIP(), []int{2}
}

func (x *RunDetails) GetBuilder() *Builder {
	if x != nil {
		return x.Builder
	}
	return nil
}

func (x *RunDetails) GetMetadata() *BuildMetadata {
	if x != nil {
		return x.Metadata
	}
	return nil
}

func (x *RunDetails) GetByproducts() []*v1.ResourceDescriptor {
	if x != nil {
		return x.Byproducts
	}
	return nil
}

type Builder struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id                  string                   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Version             map[string]string        `protobuf:"bytes,2,rep,name=version,proto3" json:"version,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	BuilderDependencies []*v1.ResourceDescriptor `protobuf:"bytes,3,rep,name=builder_dependencies,json=builderDependencies,proto3" json:"builder_dependencies,omitempty"`
}

func (x *Builder) Reset() {
	*x = Builder{}
	if protoimpl.UnsafeEnabled {
		mi := &file_in_toto_attestation_predicates_provenance_v1_provenance_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Builder) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Builder) ProtoMessage() {}

func (x *Builder) ProtoReflect() protoreflect.Message {
	mi := &file_in_toto_attestation_predicates_provenance_v1_provenance_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Builder.ProtoReflect.Descriptor instead.
func (*Builder) Descriptor() ([]byte, []int) {
	return file_in_toto_attestation_predicates_provenance_v1_provenance_proto_rawDescGZIP(), []int{3}
}

func (x *Builder) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Builder) GetVersion() map[string]string {
	if x != nil {
		return x.Version
	}
	return nil
}

func (x *Builder) GetBuilderDependencies() []*v1.ResourceDescriptor {
	if x != nil {
		return x.BuilderDependencies
	}
	return nil
}

type BuildMetadata struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	InvocationId string                 `protobuf:"bytes,1,opt,name=invocation_id,json=invocationId,proto3" json:"invocation_id,omitempty"`
	StartedOn    *timestamppb.Timestamp `protobuf:"bytes,2,opt,name=started_on,json=startedOn,proto3" json:"started_on,omitempty"`
	FinishedOn   *timestamppb.Timestamp `protobuf:"bytes,3,opt,name=finished_on,json=finishedOn,proto3" json:"finished_on,omitempty"`
}

func (x *BuildMetadata) Reset() {
	*x = BuildMetadata{}
	if protoimpl.UnsafeEnabled {
		mi := &file_in_toto_attestation_predicates_provenance_v1_provenance_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BuildMetadata) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BuildMetadata) ProtoMessage() {}

func (x *BuildMetadata) ProtoReflect() protoreflect.Message {
	mi := &file_in_toto_attestation_predicates_provenance_v1_provenance_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BuildMetadata.ProtoReflect.Descriptor instead.
func (*BuildMetadata) Descriptor() ([]byte, []int) {
	return file_in_toto_attestation_predicates_provenance_v1_provenance_proto_rawDescGZIP(), []int{4}
}

func (x *BuildMetadata) GetInvocationId() string {
	if x != nil {
		return x.InvocationId
	}
	return ""
}

func (x *BuildMetadata) GetStartedOn() *timestamppb.Timestamp {
	if x != nil {
		return x.StartedOn
	}
	return nil
}

func (x *BuildMetadata) GetFinishedOn() *timestamppb.Timestamp {
	if x != nil {
		return x.FinishedOn
	}
	return nil
}

var File_in_toto_attestation_predicates_provenance_v1_provenance_proto protoreflect.FileDescriptor

var file_in_toto_attestation_predicates_provenance_v1_provenance_proto_rawDesc = []byte{
	0x0a, 0x3d, 0x69, 0x6e, 0x5f, 0x74, 0x6f, 0x74, 0x6f, 0x5f, 0x61, 0x74, 0x74, 0x65, 0x73, 0x74,
	0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x70, 0x72, 0x65, 0x64, 0x69, 0x63, 0x61, 0x74, 0x65, 0x73,
	0x2f, 0x70, 0x72, 0x6f, 0x76, 0x65, 0x6e, 0x61, 0x6e, 0x63, 0x65, 0x2f, 0x76, 0x31, 0x2f, 0x70,
	0x72, 0x6f, 0x76, 0x65, 0x6e, 0x61, 0x6e, 0x63, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x2c, 0x69, 0x6e, 0x5f, 0x74, 0x6f, 0x74, 0x6f, 0x5f, 0x61, 0x74, 0x74, 0x65, 0x73, 0x74, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x65, 0x64, 0x69, 0x63, 0x61, 0x74, 0x65, 0x73, 0x2e,
	0x70, 0x72, 0x6f, 0x76, 0x65, 0x6e, 0x61, 0x6e, 0x63, 0x65, 0x2e, 0x76, 0x31, 0x1a, 0x30, 0x69,
	0x6e, 0x5f, 0x74, 0x6f, 0x74, 0x6f, 0x5f, 0x61, 0x74, 0x74, 0x65, 0x73, 0x74, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x2f, 0x76, 0x31, 0x2f, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x5f, 0x64,
	0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x6f, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a,
	0x1c, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2f, 0x73, 0x74, 0x72, 0x75, 0x63, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1f, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74,
	0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xd1,
	0x01, 0x0a, 0x0a, 0x50, 0x72, 0x6f, 0x76, 0x65, 0x6e, 0x61, 0x6e, 0x63, 0x65, 0x12, 0x68, 0x0a,
	0x10, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x5f, 0x64, 0x65, 0x66, 0x69, 0x6e, 0x69, 0x74, 0x69, 0x6f,
	0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x3d, 0x2e, 0x69, 0x6e, 0x5f, 0x74, 0x6f, 0x74,
	0x6f, 0x5f, 0x61, 0x74, 0x74, 0x65, 0x73, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72,
	0x65, 0x64, 0x69, 0x63, 0x61, 0x74, 0x65, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x65, 0x6e, 0x61,
	0x6e, 0x63, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x42, 0x75, 0x69, 0x6c, 0x64, 0x44, 0x65, 0x66, 0x69,
	0x6e, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0f, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x44, 0x65, 0x66,
	0x69, 0x6e, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x59, 0x0a, 0x0b, 0x72, 0x75, 0x6e, 0x5f, 0x64,
	0x65, 0x74, 0x61, 0x69, 0x6c, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x38, 0x2e, 0x69,
	0x6e, 0x5f, 0x74, 0x6f, 0x74, 0x6f, 0x5f, 0x61, 0x74, 0x74, 0x65, 0x73, 0x74, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x65, 0x64, 0x69, 0x63, 0x61, 0x74, 0x65, 0x73, 0x2e, 0x70, 0x72,
	0x6f, 0x76, 0x65, 0x6e, 0x61, 0x6e, 0x63, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x52, 0x75, 0x6e, 0x44,
	0x65, 0x74, 0x61, 0x69, 0x6c, 0x73, 0x52, 0x0a, 0x72, 0x75, 0x6e, 0x44, 0x65, 0x74, 0x61, 0x69,
	0x6c, 0x73, 0x22, 0xa5, 0x02, 0x0a, 0x0f, 0x42, 0x75, 0x69, 0x6c, 0x64, 0x44, 0x65, 0x66, 0x69,
	0x6e, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x1d, 0x0a, 0x0a, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x5f,
	0x74, 0x79, 0x70, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x62, 0x75, 0x69, 0x6c,
	0x64, 0x54, 0x79, 0x70, 0x65, 0x12, 0x48, 0x0a, 0x13, 0x65, 0x78, 0x74, 0x65, 0x72, 0x6e, 0x61,
	0x6c, 0x5f, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x73, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x17, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74, 0x52, 0x12, 0x65, 0x78, 0x74,
	0x65, 0x72, 0x6e, 0x61, 0x6c, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x73, 0x12,
	0x48, 0x0a, 0x13, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x5f, 0x70, 0x61, 0x72, 0x61,
	0x6d, 0x65, 0x74, 0x65, 0x72, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53,
	0x74, 0x72, 0x75, 0x63, 0x74, 0x52, 0x12, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x50,
	0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x73, 0x12, 0x5f, 0x0a, 0x15, 0x72, 0x65, 0x73,
	0x6f, 0x6c, 0x76, 0x65, 0x64, 0x5f, 0x64, 0x65, 0x70, 0x65, 0x6e, 0x64, 0x65, 0x6e, 0x63, 0x69,
	0x65, 0x73, 0x18, 0x04, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2a, 0x2e, 0x69, 0x6e, 0x5f, 0x74, 0x6f,
	0x74, 0x6f, 0x5f, 0x61, 0x74, 0x74, 0x65, 0x73, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x76,
	0x31, 0x2e, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x44, 0x65, 0x73, 0x63, 0x72, 0x69,
	0x70, 0x74, 0x6f, 0x72, 0x52, 0x14, 0x72, 0x65, 0x73, 0x6f, 0x6c, 0x76, 0x65, 0x64, 0x44, 0x65,
	0x70, 0x65, 0x6e, 0x64, 0x65, 0x6e, 0x63, 0x69, 0x65, 0x73, 0x22, 0x82, 0x02, 0x0a, 0x0a, 0x52,
	0x75, 0x6e, 0x44, 0x65, 0x74, 0x61, 0x69, 0x6c, 0x73, 0x12, 0x4f, 0x0a, 0x07, 0x62, 0x75, 0x69,
	0x6c, 0x64, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x35, 0x2e, 0x69, 0x6e, 0x5f,
	0x74, 0x6f, 0x74, 0x6f, 0x5f, 0x61, 0x74, 0x74, 0x65, 0x73, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x2e, 0x70, 0x72, 0x65, 0x64, 0x69, 0x63, 0x61, 0x74, 0x65, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x76,
	0x65, 0x6e, 0x61, 0x6e, 0x63, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x42, 0x75, 0x69, 0x6c, 0x64, 0x65,
	0x72, 0x52, 0x07, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x65, 0x72, 0x12, 0x57, 0x0a, 0x08, 0x6d, 0x65,
	0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x3b, 0x2e, 0x69,
	0x6e, 0x5f, 0x74, 0x6f, 0x74, 0x6f, 0x5f, 0x61, 0x74, 0x74, 0x65, 0x73, 0x74, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x65, 0x64, 0x69, 0x63, 0x61, 0x74, 0x65, 0x73, 0x2e, 0x70, 0x72,
	0x6f, 0x76, 0x65, 0x6e, 0x61, 0x6e, 0x63, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x42, 0x75, 0x69, 0x6c,
	0x64, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x52, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64,
	0x61, 0x74, 0x61, 0x12, 0x4a, 0x0a, 0x0a, 0x62, 0x79, 0x70, 0x72, 0x6f, 0x64, 0x75, 0x63, 0x74,
	0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2a, 0x2e, 0x69, 0x6e, 0x5f, 0x74, 0x6f, 0x74,
	0x6f, 0x5f, 0x61, 0x74, 0x74, 0x65, 0x73, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x76, 0x31,
	0x2e, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x44, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70,
	0x74, 0x6f, 0x72, 0x52, 0x0a, 0x62, 0x79, 0x70, 0x72, 0x6f, 0x64, 0x75, 0x63, 0x74, 0x73, 0x22,
	0x92, 0x02, 0x0a, 0x07, 0x42, 0x75, 0x69, 0x6c, 0x64, 0x65, 0x72, 0x12, 0x0e, 0x0a, 0x02, 0x69,
	0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x5c, 0x0a, 0x07, 0x76,
	0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x42, 0x2e, 0x69,
	0x6e, 0x5f, 0x74, 0x6f, 0x74, 0x6f, 0x5f, 0x61, 0x74, 0x74, 0x65, 0x73, 0x74, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x65, 0x64, 0x69, 0x63, 0x61, 0x74, 0x65, 0x73, 0x2e, 0x70, 0x72,
	0x6f, 0x76, 0x65, 0x6e, 0x61, 0x6e, 0x63, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x42, 0x75, 0x69, 0x6c,
	0x64, 0x65, 0x72, 0x2e, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x45, 0x6e, 0x74, 0x72, 0x79,
	0x52, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x5d, 0x0a, 0x14, 0x62, 0x75, 0x69,
	0x6c, 0x64, 0x65, 0x72, 0x5f, 0x64, 0x65, 0x70, 0x65, 0x6e, 0x64, 0x65, 0x6e, 0x63, 0x69, 0x65,
	0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2a, 0x2e, 0x69, 0x6e, 0x5f, 0x74, 0x6f, 0x74,
	0x6f, 0x5f, 0x61, 0x74, 0x74, 0x65, 0x73, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x76, 0x31,
	0x2e, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x44, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70,
	0x74, 0x6f, 0x72, 0x52, 0x13, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x65, 0x72, 0x44, 0x65, 0x70, 0x65,
	0x6e, 0x64, 0x65, 0x6e, 0x63, 0x69, 0x65, 0x73, 0x1a, 0x3a, 0x0a, 0x0c, 0x56, 0x65, 0x72, 0x73,
	0x69, 0x6f, 0x6e, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61,
	0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65,
	0x3a, 0x02, 0x38, 0x01, 0x22, 0xac, 0x01, 0x0a, 0x0d, 0x42, 0x75, 0x69, 0x6c, 0x64, 0x4d, 0x65,
	0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x12, 0x23, 0x0a, 0x0d, 0x69, 0x6e, 0x76, 0x6f, 0x63, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x69,
	0x6e, 0x76, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x12, 0x39, 0x0a, 0x0a, 0x73,
	0x74, 0x61, 0x72, 0x74, 0x65, 0x64, 0x5f, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x09, 0x73, 0x74, 0x61,
	0x72, 0x74, 0x65, 0x64, 0x4f, 0x6e, 0x12, 0x3b, 0x0a, 0x0b, 0x66, 0x69, 0x6e, 0x69, 0x73, 0x68,
	0x65, 0x64, 0x5f, 0x6f, 0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69,
	0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x0a, 0x66, 0x69, 0x6e, 0x69, 0x73, 0x68, 0x65,
	0x64, 0x4f, 0x6e, 0x42, 0x73, 0x0a, 0x35, 0x69, 0x6f, 0x2e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62,
	0x2e, 0x69, 0x6e, 0x74, 0x6f, 0x74, 0x6f, 0x2e, 0x61, 0x74, 0x74, 0x65, 0x73, 0x74, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x65, 0x64, 0x69, 0x63, 0x61, 0x74, 0x65, 0x73, 0x2e, 0x70,
	0x72, 0x6f, 0x76, 0x65, 0x6e, 0x61, 0x6e, 0x63, 0x65, 0x2e, 0x76, 0x31, 0x5a, 0x3a, 0x67, 0x69,
	0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x69, 0x6e, 0x2d, 0x74, 0x6f, 0x74, 0x6f,
	0x2f, 0x61, 0x74, 0x74, 0x65, 0x73, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x67, 0x6f, 0x2f,
	0x70, 0x72, 0x65, 0x64, 0x69, 0x63, 0x61, 0x74, 0x65, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x76, 0x65,
	0x6e, 0x61, 0x6e, 0x63, 0x65, 0x2f, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_in_toto_attestation_predicates_provenance_v1_provenance_proto_rawDescOnce sync.Once
	file_in_toto_attestation_predicates_provenance_v1_provenance_proto_rawDescData = file_in_toto_attestation_predicates_provenance_v1_provenance_proto_rawDesc
)

func file_in_toto_attestation_predicates_provenance_v1_provenance_proto_rawDescGZIP() []byte {
	file_in_toto_attestation_predicates_provenance_v1_provenance_proto_rawDescOnce.Do(func() {
		file_in_toto_attestation_predicates_provenance_v1_provenance_proto_rawDescData = protoimpl.X.CompressGZIP(file_in_toto_attestation_predicates_provenance_v1_provenance_proto_rawDescData)
	})
	return file_in_toto_attestation_predicates_provenance_v1_provenance_proto_rawDescData
}

var file_in_toto_attestation_predicates_provenance_v1_provenance_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_in_toto_attestation_predicates_provenance_v1_provenance_proto_goTypes = []interface{}{
	(*Provenance)(nil),            // 0: in_toto_attestation.predicates.provenance.v1.Provenance
	(*BuildDefinition)(nil),       // 1: in_toto_attestation.predicates.provenance.v1.BuildDefinition
	(*RunDetails)(nil),            // 2: in_toto_attestation.predicates.provenance.v1.RunDetails
	(*Builder)(nil),               // 3: in_toto_attestation.predicates.provenance.v1.Builder
	(*BuildMetadata)(nil),         // 4: in_toto_attestation.predicates.provenance.v1.BuildMetadata
	nil,                           // 5: in_toto_attestation.predicates.provenance.v1.Builder.VersionEntry
	(*structpb.Struct)(nil),       // 6: google.protobuf.Struct
	(*v1.ResourceDescriptor)(nil), // 7: in_toto_attestation.v1.ResourceDescriptor
	(*timestamppb.Timestamp)(nil), // 8: google.protobuf.Timestamp
}
var file_in_toto_attestation_predicates_provenance_v1_provenance_proto_depIdxs = []int32{
	1,  // 0: in_toto_attestation.predicates.provenance.v1.Provenance.build_definition:type_name -> in_toto_attestation.predicates.provenance.v1.BuildDefinition
	2,  // 1: in_toto_attestation.predicates.provenance.v1.Provenance.run_details:type_name -> in_toto_attestation.predicates.provenance.v1.RunDetails
	6,  // 2: in_toto_attestation.predicates.provenance.v1.BuildDefinition.external_parameters:type_name -> google.protobuf.Struct
	6,  // 3: in_toto_attestation.predicates.provenance.v1.BuildDefinition.internal_parameters:type_name -> google.protobuf.Struct
	7,  // 4: in_toto_attestation.predicates.provenance.v1.BuildDefinition.resolved_dependencies:type_name -> in_toto_attestation.v1.ResourceDescriptor
	3,  // 5: in_toto_attestation.predicates.provenance.v1.RunDetails.builder:type_name -> in_toto_attestation.predicates.provenance.v1.Builder
	4,  // 6: in_toto_attestation.predicates.provenance.v1.RunDetails.metadata:type_name -> in_toto_attestation.predicates.provenance.v1.BuildMetadata
	7,  // 7: in_toto_attestation.predicates.provenance.v1.RunDetails.byproducts:type_name -> in_toto_attestation.v1.ResourceDescriptor
	5,  // 8: in_toto_attestation.predicates.provenance.v1.Builder.version:type_name -> in_toto_attestation.predicates.provenance.v1.Builder.VersionEntry
	7,  // 9: in_toto_attestation.predicates.provenance.v1.Builder.builder_dependencies:type_name -> in_toto_attestation.v1.ResourceDescriptor
	8,  // 10: in_toto_attestation.predicates.provenance.v1.BuildMetadata.started_on:type_name -> google.protobuf.Timestamp
	8,  // 11: in_toto_attestation.predicates.provenance.v1.BuildMetadata.finished_on:type_name -> google.protobuf.Timestamp
	12, // [12:12] is the sub-list for method output_type
	12, // [12:12] is the sub-list for method input_type
	12, // [12:12] is the sub-list for extension type_name
	12, // [12:12] is the sub-list for extension extendee
	0,  // [0:12] is the sub-list for field type_name
}

func init() { file_in_toto_attestation_predicates_provenance_v1_provenance_proto_init() }
func file_in_toto_attestation_predicates_provenance_v1_provenance_proto_init() {
	if File_in_toto_attestation_predicates_provenance_v1_provenance_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_in_toto_attestation_predicates_provenance_v1_provenance_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Provenance); i {
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
		file_in_toto_attestation_predicates_provenance_v1_provenance_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BuildDefinition); i {
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
		file_in_toto_attestation_predicates_provenance_v1_provenance_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RunDetails); i {
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
		file_in_toto_attestation_predicates_provenance_v1_provenance_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Builder); i {
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
		file_in_toto_attestation_predicates_provenance_v1_provenance_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BuildMetadata); i {
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
			RawDescriptor: file_in_toto_attestation_predicates_provenance_v1_provenance_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_in_toto_attestation_predicates_provenance_v1_provenance_proto_goTypes,
		DependencyIndexes: file_in_toto_attestation_predicates_provenance_v1_provenance_proto_depIdxs,
		MessageInfos:      file_in_toto_attestation_predicates_provenance_v1_provenance_proto_msgTypes,
	}.Build()
	File_in_toto_attestation_predicates_provenance_v1_provenance_proto = out.File
	file_in_toto_attestation_predicates_provenance_v1_provenance_proto_rawDesc = nil
	file_in_toto_attestation_predicates_provenance_v1_provenance_proto_goTypes = nil
	file_in_toto_attestation_predicates_provenance_v1_provenance_proto_depIdxs = nil
}