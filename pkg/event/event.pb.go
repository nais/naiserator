// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.4
// 	protoc        v5.29.3
// source: pkg/event/event.proto

package deployment

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// PlatformType denotes what kind of platform are used for deploying.
type PlatformType int32

const (
	PlatformType_jboss PlatformType = 0
	PlatformType_was   PlatformType = 1
	PlatformType_bpm   PlatformType = 2
	PlatformType_nais  PlatformType = 3
)

// Enum value maps for PlatformType.
var (
	PlatformType_name = map[int32]string{
		0: "jboss",
		1: "was",
		2: "bpm",
		3: "nais",
	}
	PlatformType_value = map[string]int32{
		"jboss": 0,
		"was":   1,
		"bpm":   2,
		"nais":  3,
	}
)

func (x PlatformType) Enum() *PlatformType {
	p := new(PlatformType)
	*p = x
	return p
}

func (x PlatformType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (PlatformType) Descriptor() protoreflect.EnumDescriptor {
	return file_pkg_event_event_proto_enumTypes[0].Descriptor()
}

func (PlatformType) Type() protoreflect.EnumType {
	return &file_pkg_event_event_proto_enumTypes[0]
}

func (x PlatformType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use PlatformType.Descriptor instead.
func (PlatformType) EnumDescriptor() ([]byte, []int) {
	return file_pkg_event_event_proto_rawDescGZIP(), []int{0}
}

// Only enumerated systems can report deployment status.
type System int32

const (
	System_aura       System = 0
	System_naisd      System = 1
	System_naiserator System = 2
)

// Enum value maps for System.
var (
	System_name = map[int32]string{
		0: "aura",
		1: "naisd",
		2: "naiserator",
	}
	System_value = map[string]int32{
		"aura":       0,
		"naisd":      1,
		"naiserator": 2,
	}
)

func (x System) Enum() *System {
	p := new(System)
	*p = x
	return p
}

func (x System) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (System) Descriptor() protoreflect.EnumDescriptor {
	return file_pkg_event_event_proto_enumTypes[1].Descriptor()
}

func (System) Type() protoreflect.EnumType {
	return &file_pkg_event_event_proto_enumTypes[1]
}

func (x System) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use System.Descriptor instead.
func (System) EnumDescriptor() ([]byte, []int) {
	return file_pkg_event_event_proto_rawDescGZIP(), []int{1}
}

// RolloutStatus denotes whether a deployment has been initialized,
// rolled out successfully, or if the status is altogether unknown.
type RolloutStatus int32

const (
	RolloutStatus_unknown     RolloutStatus = 0
	RolloutStatus_initialized RolloutStatus = 1
	RolloutStatus_complete    RolloutStatus = 2
)

// Enum value maps for RolloutStatus.
var (
	RolloutStatus_name = map[int32]string{
		0: "unknown",
		1: "initialized",
		2: "complete",
	}
	RolloutStatus_value = map[string]int32{
		"unknown":     0,
		"initialized": 1,
		"complete":    2,
	}
)

func (x RolloutStatus) Enum() *RolloutStatus {
	p := new(RolloutStatus)
	*p = x
	return p
}

func (x RolloutStatus) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (RolloutStatus) Descriptor() protoreflect.EnumDescriptor {
	return file_pkg_event_event_proto_enumTypes[2].Descriptor()
}

func (RolloutStatus) Type() protoreflect.EnumType {
	return &file_pkg_event_event_proto_enumTypes[2]
}

func (x RolloutStatus) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use RolloutStatus.Descriptor instead.
func (RolloutStatus) EnumDescriptor() ([]byte, []int) {
	return file_pkg_event_event_proto_rawDescGZIP(), []int{2}
}

// Environment separates between production and development environments.
type Environment int32

const (
	Environment_production  Environment = 0
	Environment_development Environment = 1
)

// Enum value maps for Environment.
var (
	Environment_name = map[int32]string{
		0: "production",
		1: "development",
	}
	Environment_value = map[string]int32{
		"production":  0,
		"development": 1,
	}
)

func (x Environment) Enum() *Environment {
	p := new(Environment)
	*p = x
	return p
}

func (x Environment) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Environment) Descriptor() protoreflect.EnumDescriptor {
	return file_pkg_event_event_proto_enumTypes[3].Descriptor()
}

func (Environment) Type() protoreflect.EnumType {
	return &file_pkg_event_event_proto_enumTypes[3]
}

func (x Environment) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Environment.Descriptor instead.
func (Environment) EnumDescriptor() ([]byte, []int) {
	return file_pkg_event_event_proto_rawDescGZIP(), []int{3}
}

// A platform represents a place where applications and systems are deployed.
// Since platforms come in different versions and flavors, a variant can also be specified.
type Platform struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Type          PlatformType           `protobuf:"varint,1,opt,name=type,proto3,enum=deployment.PlatformType" json:"type,omitempty"`
	Variant       string                 `protobuf:"bytes,2,opt,name=variant,proto3" json:"variant,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Platform) Reset() {
	*x = Platform{}
	mi := &file_pkg_event_event_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Platform) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Platform) ProtoMessage() {}

func (x *Platform) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_event_event_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Platform.ProtoReflect.Descriptor instead.
func (*Platform) Descriptor() ([]byte, []int) {
	return file_pkg_event_event_proto_rawDescGZIP(), []int{0}
}

func (x *Platform) GetType() PlatformType {
	if x != nil {
		return x.Type
	}
	return PlatformType_jboss
}

func (x *Platform) GetVariant() string {
	if x != nil {
		return x.Variant
	}
	return ""
}

// Actor is a human being or a service account.
type Actor struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Ident         string                 `protobuf:"bytes,1,opt,name=ident,proto3" json:"ident,omitempty"`
	Email         string                 `protobuf:"bytes,2,opt,name=email,proto3" json:"email,omitempty"`
	Name          string                 `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Actor) Reset() {
	*x = Actor{}
	mi := &file_pkg_event_event_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Actor) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Actor) ProtoMessage() {}

func (x *Actor) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_event_event_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Actor.ProtoReflect.Descriptor instead.
func (*Actor) Descriptor() ([]byte, []int) {
	return file_pkg_event_event_proto_rawDescGZIP(), []int{1}
}

func (x *Actor) GetIdent() string {
	if x != nil {
		return x.Ident
	}
	return ""
}

func (x *Actor) GetEmail() string {
	if x != nil {
		return x.Email
	}
	return ""
}

func (x *Actor) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

// ContainerImage is a reference to a image that can be deployed as a container,
// typically a Docker container inside a Kubernetes pod.
type ContainerImage struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Name          string                 `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Tag           string                 `protobuf:"bytes,2,opt,name=tag,proto3" json:"tag,omitempty"`
	Hash          string                 `protobuf:"bytes,3,opt,name=hash,proto3" json:"hash,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ContainerImage) Reset() {
	*x = ContainerImage{}
	mi := &file_pkg_event_event_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ContainerImage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ContainerImage) ProtoMessage() {}

func (x *ContainerImage) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_event_event_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ContainerImage.ProtoReflect.Descriptor instead.
func (*ContainerImage) Descriptor() ([]byte, []int) {
	return file_pkg_event_event_proto_rawDescGZIP(), []int{2}
}

func (x *ContainerImage) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *ContainerImage) GetTag() string {
	if x != nil {
		return x.Tag
	}
	return ""
}

func (x *ContainerImage) GetHash() string {
	if x != nil {
		return x.Hash
	}
	return ""
}

// Event represents a deployment that has been made on any of NAV's systems.
type Event struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// CorrelationID can be used to correlate events across different systems.
	CorrelationID string `protobuf:"bytes,1,opt,name=correlationID,proto3" json:"correlationID,omitempty"`
	// Platform represents the technical platform on which the deployment was made.
	Platform *Platform `protobuf:"bytes,2,opt,name=platform,proto3" json:"platform,omitempty"`
	// Source tells which system that reported the deployment.
	Source System `protobuf:"varint,3,opt,name=source,proto3,enum=deployment.System" json:"source,omitempty"`
	// Deployer is a reference to a human being that started the deployment.
	Deployer *Actor `protobuf:"bytes,4,opt,name=deployer,proto3" json:"deployer,omitempty"`
	// Team is an organizational structure within NAV and refers to a group of people.
	Team string `protobuf:"bytes,5,opt,name=team,proto3" json:"team,omitempty"`
	// RolloutStatus shows the deployment status.
	RolloutStatus RolloutStatus `protobuf:"varint,6,opt,name=rolloutStatus,proto3,enum=deployment.RolloutStatus" json:"rolloutStatus,omitempty"`
	// Environment can be production or development.
	Environment Environment `protobuf:"varint,7,opt,name=environment,proto3,enum=deployment.Environment" json:"environment,omitempty"`
	// The SKYA platform divides between production, development, staging, and test.
	// Furthermore, these environments are divided into smaller segments denoted with
	// a number, such as q0, t6, u11.
	SkyaEnvironment string `protobuf:"bytes,8,opt,name=skyaEnvironment,proto3" json:"skyaEnvironment,omitempty"`
	// Namespace represents the Kubernetes namespace this deployment was made into.
	Namespace string `protobuf:"bytes,9,opt,name=namespace,proto3" json:"namespace,omitempty"`
	// Cluster is the name of the Kubernetes cluster that was deployed to.
	Cluster string `protobuf:"bytes,10,opt,name=cluster,proto3" json:"cluster,omitempty"`
	// Application is the name of the deployed application.
	Application string `protobuf:"bytes,11,opt,name=application,proto3" json:"application,omitempty"`
	// Version is the version of the deployed application.
	Version string `protobuf:"bytes,12,opt,name=version,proto3" json:"version,omitempty"`
	// Image refers to the container source, usually a Docker image.
	Image *ContainerImage `protobuf:"bytes,13,opt,name=image,proto3" json:"image,omitempty"`
	// Timestamp is the generation time of the deployment event.
	Timestamp *timestamppb.Timestamp `protobuf:"bytes,14,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	// GitCommitSha refers to the commit that the application build is based on.
	GitCommitSha  string `protobuf:"bytes,15,opt,name=gitCommitSha,proto3" json:"gitCommitSha,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Event) Reset() {
	*x = Event{}
	mi := &file_pkg_event_event_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Event) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Event) ProtoMessage() {}

func (x *Event) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_event_event_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Event.ProtoReflect.Descriptor instead.
func (*Event) Descriptor() ([]byte, []int) {
	return file_pkg_event_event_proto_rawDescGZIP(), []int{3}
}

func (x *Event) GetCorrelationID() string {
	if x != nil {
		return x.CorrelationID
	}
	return ""
}

func (x *Event) GetPlatform() *Platform {
	if x != nil {
		return x.Platform
	}
	return nil
}

func (x *Event) GetSource() System {
	if x != nil {
		return x.Source
	}
	return System_aura
}

func (x *Event) GetDeployer() *Actor {
	if x != nil {
		return x.Deployer
	}
	return nil
}

func (x *Event) GetTeam() string {
	if x != nil {
		return x.Team
	}
	return ""
}

func (x *Event) GetRolloutStatus() RolloutStatus {
	if x != nil {
		return x.RolloutStatus
	}
	return RolloutStatus_unknown
}

func (x *Event) GetEnvironment() Environment {
	if x != nil {
		return x.Environment
	}
	return Environment_production
}

func (x *Event) GetSkyaEnvironment() string {
	if x != nil {
		return x.SkyaEnvironment
	}
	return ""
}

func (x *Event) GetNamespace() string {
	if x != nil {
		return x.Namespace
	}
	return ""
}

func (x *Event) GetCluster() string {
	if x != nil {
		return x.Cluster
	}
	return ""
}

func (x *Event) GetApplication() string {
	if x != nil {
		return x.Application
	}
	return ""
}

func (x *Event) GetVersion() string {
	if x != nil {
		return x.Version
	}
	return ""
}

func (x *Event) GetImage() *ContainerImage {
	if x != nil {
		return x.Image
	}
	return nil
}

func (x *Event) GetTimestamp() *timestamppb.Timestamp {
	if x != nil {
		return x.Timestamp
	}
	return nil
}

func (x *Event) GetGitCommitSha() string {
	if x != nil {
		return x.GitCommitSha
	}
	return ""
}

var File_pkg_event_event_proto protoreflect.FileDescriptor

var file_pkg_event_event_proto_rawDesc = string([]byte{
	0x0a, 0x15, 0x70, 0x6b, 0x67, 0x2f, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x2f, 0x65, 0x76, 0x65, 0x6e,
	0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0a, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x6d,
	0x65, 0x6e, 0x74, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x22, 0x52, 0x0a, 0x08, 0x50, 0x6c, 0x61, 0x74, 0x66, 0x6f, 0x72, 0x6d,
	0x12, 0x2c, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x18,
	0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x6d, 0x65, 0x6e, 0x74, 0x2e, 0x50, 0x6c, 0x61, 0x74,
	0x66, 0x6f, 0x72, 0x6d, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x18,
	0x0a, 0x07, 0x76, 0x61, 0x72, 0x69, 0x61, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x07, 0x76, 0x61, 0x72, 0x69, 0x61, 0x6e, 0x74, 0x22, 0x47, 0x0a, 0x05, 0x41, 0x63, 0x74, 0x6f,
	0x72, 0x12, 0x14, 0x0a, 0x05, 0x69, 0x64, 0x65, 0x6e, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x05, 0x69, 0x64, 0x65, 0x6e, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x65, 0x6d, 0x61, 0x69, 0x6c,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x65, 0x6d, 0x61, 0x69, 0x6c, 0x12, 0x12, 0x0a,
	0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d,
	0x65, 0x22, 0x4a, 0x0a, 0x0e, 0x43, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x49, 0x6d,
	0x61, 0x67, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x10, 0x0a, 0x03, 0x74, 0x61, 0x67, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x74, 0x61, 0x67, 0x12, 0x12, 0x0a, 0x04, 0x68, 0x61, 0x73,
	0x68, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x68, 0x61, 0x73, 0x68, 0x22, 0xf8, 0x04,
	0x0a, 0x05, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x12, 0x24, 0x0a, 0x0d, 0x63, 0x6f, 0x72, 0x72, 0x65,
	0x6c, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x44, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d,
	0x63, 0x6f, 0x72, 0x72, 0x65, 0x6c, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x44, 0x12, 0x30, 0x0a,
	0x08, 0x70, 0x6c, 0x61, 0x74, 0x66, 0x6f, 0x72, 0x6d, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x14, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x6d, 0x65, 0x6e, 0x74, 0x2e, 0x50, 0x6c, 0x61,
	0x74, 0x66, 0x6f, 0x72, 0x6d, 0x52, 0x08, 0x70, 0x6c, 0x61, 0x74, 0x66, 0x6f, 0x72, 0x6d, 0x12,
	0x2a, 0x0a, 0x06, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0e, 0x32,
	0x12, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x6d, 0x65, 0x6e, 0x74, 0x2e, 0x53, 0x79, 0x73,
	0x74, 0x65, 0x6d, 0x52, 0x06, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x12, 0x2d, 0x0a, 0x08, 0x64,
	0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x11, 0x2e,
	0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x6d, 0x65, 0x6e, 0x74, 0x2e, 0x41, 0x63, 0x74, 0x6f, 0x72,
	0x52, 0x08, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x65,
	0x61, 0x6d, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x74, 0x65, 0x61, 0x6d, 0x12, 0x3f,
	0x0a, 0x0d, 0x72, 0x6f, 0x6c, 0x6c, 0x6f, 0x75, 0x74, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18,
	0x06, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x19, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x6d, 0x65,
	0x6e, 0x74, 0x2e, 0x52, 0x6f, 0x6c, 0x6c, 0x6f, 0x75, 0x74, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73,
	0x52, 0x0d, 0x72, 0x6f, 0x6c, 0x6c, 0x6f, 0x75, 0x74, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12,
	0x39, 0x0a, 0x0b, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x18, 0x07,
	0x20, 0x01, 0x28, 0x0e, 0x32, 0x17, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x6d, 0x65, 0x6e,
	0x74, 0x2e, 0x45, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x52, 0x0b, 0x65,
	0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x12, 0x28, 0x0a, 0x0f, 0x73, 0x6b,
	0x79, 0x61, 0x45, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x18, 0x08, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x0f, 0x73, 0x6b, 0x79, 0x61, 0x45, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e,
	0x6d, 0x65, 0x6e, 0x74, 0x12, 0x1c, 0x0a, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63,
	0x65, 0x18, 0x09, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61,
	0x63, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x18, 0x0a, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x07, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x12, 0x20, 0x0a, 0x0b,
	0x61, 0x70, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x0b, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x0b, 0x61, 0x70, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x18,
	0x0a, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x0c, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x30, 0x0a, 0x05, 0x69, 0x6d, 0x61, 0x67,
	0x65, 0x18, 0x0d, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79,
	0x6d, 0x65, 0x6e, 0x74, 0x2e, 0x43, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x49, 0x6d,
	0x61, 0x67, 0x65, 0x52, 0x05, 0x69, 0x6d, 0x61, 0x67, 0x65, 0x12, 0x38, 0x0a, 0x09, 0x74, 0x69,
	0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x0e, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73,
	0x74, 0x61, 0x6d, 0x70, 0x12, 0x22, 0x0a, 0x0c, 0x67, 0x69, 0x74, 0x43, 0x6f, 0x6d, 0x6d, 0x69,
	0x74, 0x53, 0x68, 0x61, 0x18, 0x0f, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x67, 0x69, 0x74, 0x43,
	0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x53, 0x68, 0x61, 0x2a, 0x35, 0x0a, 0x0c, 0x50, 0x6c, 0x61, 0x74,
	0x66, 0x6f, 0x72, 0x6d, 0x54, 0x79, 0x70, 0x65, 0x12, 0x09, 0x0a, 0x05, 0x6a, 0x62, 0x6f, 0x73,
	0x73, 0x10, 0x00, 0x12, 0x07, 0x0a, 0x03, 0x77, 0x61, 0x73, 0x10, 0x01, 0x12, 0x07, 0x0a, 0x03,
	0x62, 0x70, 0x6d, 0x10, 0x02, 0x12, 0x08, 0x0a, 0x04, 0x6e, 0x61, 0x69, 0x73, 0x10, 0x03, 0x2a,
	0x2d, 0x0a, 0x06, 0x53, 0x79, 0x73, 0x74, 0x65, 0x6d, 0x12, 0x08, 0x0a, 0x04, 0x61, 0x75, 0x72,
	0x61, 0x10, 0x00, 0x12, 0x09, 0x0a, 0x05, 0x6e, 0x61, 0x69, 0x73, 0x64, 0x10, 0x01, 0x12, 0x0e,
	0x0a, 0x0a, 0x6e, 0x61, 0x69, 0x73, 0x65, 0x72, 0x61, 0x74, 0x6f, 0x72, 0x10, 0x02, 0x2a, 0x3b,
	0x0a, 0x0d, 0x52, 0x6f, 0x6c, 0x6c, 0x6f, 0x75, 0x74, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12,
	0x0b, 0x0a, 0x07, 0x75, 0x6e, 0x6b, 0x6e, 0x6f, 0x77, 0x6e, 0x10, 0x00, 0x12, 0x0f, 0x0a, 0x0b,
	0x69, 0x6e, 0x69, 0x74, 0x69, 0x61, 0x6c, 0x69, 0x7a, 0x65, 0x64, 0x10, 0x01, 0x12, 0x0c, 0x0a,
	0x08, 0x63, 0x6f, 0x6d, 0x70, 0x6c, 0x65, 0x74, 0x65, 0x10, 0x02, 0x2a, 0x2e, 0x0a, 0x0b, 0x45,
	0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x12, 0x0e, 0x0a, 0x0a, 0x70, 0x72,
	0x6f, 0x64, 0x75, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x10, 0x00, 0x12, 0x0f, 0x0a, 0x0b, 0x64, 0x65,
	0x76, 0x65, 0x6c, 0x6f, 0x70, 0x6d, 0x65, 0x6e, 0x74, 0x10, 0x01, 0x42, 0x2b, 0x0a, 0x18, 0x6e,
	0x6f, 0x2e, 0x6e, 0x61, 0x76, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2e, 0x64, 0x65, 0x70,
	0x6c, 0x6f, 0x79, 0x6d, 0x65, 0x6e, 0x74, 0x42, 0x0f, 0x44, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x6d,
	0x65, 0x6e, 0x74, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
})

var (
	file_pkg_event_event_proto_rawDescOnce sync.Once
	file_pkg_event_event_proto_rawDescData []byte
)

func file_pkg_event_event_proto_rawDescGZIP() []byte {
	file_pkg_event_event_proto_rawDescOnce.Do(func() {
		file_pkg_event_event_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_pkg_event_event_proto_rawDesc), len(file_pkg_event_event_proto_rawDesc)))
	})
	return file_pkg_event_event_proto_rawDescData
}

var file_pkg_event_event_proto_enumTypes = make([]protoimpl.EnumInfo, 4)
var file_pkg_event_event_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_pkg_event_event_proto_goTypes = []any{
	(PlatformType)(0),             // 0: deployment.PlatformType
	(System)(0),                   // 1: deployment.System
	(RolloutStatus)(0),            // 2: deployment.RolloutStatus
	(Environment)(0),              // 3: deployment.Environment
	(*Platform)(nil),              // 4: deployment.Platform
	(*Actor)(nil),                 // 5: deployment.Actor
	(*ContainerImage)(nil),        // 6: deployment.ContainerImage
	(*Event)(nil),                 // 7: deployment.Event
	(*timestamppb.Timestamp)(nil), // 8: google.protobuf.Timestamp
}
var file_pkg_event_event_proto_depIdxs = []int32{
	0, // 0: deployment.Platform.type:type_name -> deployment.PlatformType
	4, // 1: deployment.Event.platform:type_name -> deployment.Platform
	1, // 2: deployment.Event.source:type_name -> deployment.System
	5, // 3: deployment.Event.deployer:type_name -> deployment.Actor
	2, // 4: deployment.Event.rolloutStatus:type_name -> deployment.RolloutStatus
	3, // 5: deployment.Event.environment:type_name -> deployment.Environment
	6, // 6: deployment.Event.image:type_name -> deployment.ContainerImage
	8, // 7: deployment.Event.timestamp:type_name -> google.protobuf.Timestamp
	8, // [8:8] is the sub-list for method output_type
	8, // [8:8] is the sub-list for method input_type
	8, // [8:8] is the sub-list for extension type_name
	8, // [8:8] is the sub-list for extension extendee
	0, // [0:8] is the sub-list for field type_name
}

func init() { file_pkg_event_event_proto_init() }
func file_pkg_event_event_proto_init() {
	if File_pkg_event_event_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_pkg_event_event_proto_rawDesc), len(file_pkg_event_event_proto_rawDesc)),
			NumEnums:      4,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_pkg_event_event_proto_goTypes,
		DependencyIndexes: file_pkg_event_event_proto_depIdxs,
		EnumInfos:         file_pkg_event_event_proto_enumTypes,
		MessageInfos:      file_pkg_event_event_proto_msgTypes,
	}.Build()
	File_pkg_event_event_proto = out.File
	file_pkg_event_event_proto_goTypes = nil
	file_pkg_event_event_proto_depIdxs = nil
}
