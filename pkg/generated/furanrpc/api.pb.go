// Code generated by protoc-gen-go. DO NOT EDIT.
// source: api.proto

package furanrpc

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	descriptorpb "google.golang.org/protobuf/types/descriptorpb"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type BuildState int32

const (
	BuildState_UNKNOWN          BuildState = 0
	BuildState_NOTSTARTED       BuildState = 1
	BuildState_SKIPPED          BuildState = 2
	BuildState_RUNNING          BuildState = 3
	BuildState_FAILURE          BuildState = 4
	BuildState_SUCCESS          BuildState = 5
	BuildState_CANCEL_REQUESTED BuildState = 6
	BuildState_CANCELLED        BuildState = 7
)

var BuildState_name = map[int32]string{
	0: "UNKNOWN",
	1: "NOTSTARTED",
	2: "SKIPPED",
	3: "RUNNING",
	4: "FAILURE",
	5: "SUCCESS",
	6: "CANCEL_REQUESTED",
	7: "CANCELLED",
}

var BuildState_value = map[string]int32{
	"UNKNOWN":          0,
	"NOTSTARTED":       1,
	"SKIPPED":          2,
	"RUNNING":          3,
	"FAILURE":          4,
	"SUCCESS":          5,
	"CANCEL_REQUESTED": 6,
	"CANCELLED":        7,
}

func (x BuildState) String() string {
	return proto.EnumName(BuildState_name, int32(x))
}

func (BuildState) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{0}
}

// From https://github.com/protocolbuffers/protobuf/blob/master/src/google/protobuf/timestamp.proto
type Timestamp struct {
	// Represents seconds of UTC time since Unix epoch
	// 1970-01-01T00:00:00Z. Must be from 0001-01-01T00:00:00Z to
	// 9999-12-31T23:59:59Z inclusive.
	Seconds              int64    `protobuf:"varint,1,opt,name=seconds,proto3" json:"seconds,omitempty"`
	Nanos                int32    `protobuf:"varint,2,opt,name=nanos,proto3" json:"nanos,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Timestamp) Reset()         { *m = Timestamp{} }
func (m *Timestamp) String() string { return proto.CompactTextString(m) }
func (*Timestamp) ProtoMessage()    {}
func (*Timestamp) Descriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{0}
}

func (m *Timestamp) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Timestamp.Unmarshal(m, b)
}
func (m *Timestamp) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Timestamp.Marshal(b, m, deterministic)
}
func (m *Timestamp) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Timestamp.Merge(m, src)
}
func (m *Timestamp) XXX_Size() int {
	return xxx_messageInfo_Timestamp.Size(m)
}
func (m *Timestamp) XXX_DiscardUnknown() {
	xxx_messageInfo_Timestamp.DiscardUnknown(m)
}

var xxx_messageInfo_Timestamp proto.InternalMessageInfo

func (m *Timestamp) GetSeconds() int64 {
	if m != nil {
		return m.Seconds
	}
	return 0
}

func (m *Timestamp) GetNanos() int32 {
	if m != nil {
		return m.Nanos
	}
	return 0
}

type BuildDefinition struct {
	GithubRepo           string            `protobuf:"bytes,1,opt,name=github_repo,json=githubRepo,proto3" json:"github_repo,omitempty"`
	GithubCredential     string            `protobuf:"bytes,2,opt,name=github_credential,json=githubCredential,proto3" json:"github_credential,omitempty"`
	DockerfilePath       string            `protobuf:"bytes,3,opt,name=dockerfile_path,json=dockerfilePath,proto3" json:"dockerfile_path,omitempty"`
	Ref                  string            `protobuf:"bytes,4,opt,name=ref,proto3" json:"ref,omitempty"`
	Tags                 []string          `protobuf:"bytes,5,rep,name=tags,proto3" json:"tags,omitempty"`
	TagWithCommitSha     bool              `protobuf:"varint,6,opt,name=tag_with_commit_sha,json=tagWithCommitSha,proto3" json:"tag_with_commit_sha,omitempty"`
	Args                 map[string]string `protobuf:"bytes,7,rep,name=args,proto3" json:"args,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	DisableBuildCache    bool              `protobuf:"varint,8,opt,name=disable_build_cache,json=disableBuildCache,proto3" json:"disable_build_cache,omitempty"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *BuildDefinition) Reset()         { *m = BuildDefinition{} }
func (m *BuildDefinition) String() string { return proto.CompactTextString(m) }
func (*BuildDefinition) ProtoMessage()    {}
func (*BuildDefinition) Descriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{1}
}

func (m *BuildDefinition) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BuildDefinition.Unmarshal(m, b)
}
func (m *BuildDefinition) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BuildDefinition.Marshal(b, m, deterministic)
}
func (m *BuildDefinition) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BuildDefinition.Merge(m, src)
}
func (m *BuildDefinition) XXX_Size() int {
	return xxx_messageInfo_BuildDefinition.Size(m)
}
func (m *BuildDefinition) XXX_DiscardUnknown() {
	xxx_messageInfo_BuildDefinition.DiscardUnknown(m)
}

var xxx_messageInfo_BuildDefinition proto.InternalMessageInfo

func (m *BuildDefinition) GetGithubRepo() string {
	if m != nil {
		return m.GithubRepo
	}
	return ""
}

func (m *BuildDefinition) GetGithubCredential() string {
	if m != nil {
		return m.GithubCredential
	}
	return ""
}

func (m *BuildDefinition) GetDockerfilePath() string {
	if m != nil {
		return m.DockerfilePath
	}
	return ""
}

func (m *BuildDefinition) GetRef() string {
	if m != nil {
		return m.Ref
	}
	return ""
}

func (m *BuildDefinition) GetTags() []string {
	if m != nil {
		return m.Tags
	}
	return nil
}

func (m *BuildDefinition) GetTagWithCommitSha() bool {
	if m != nil {
		return m.TagWithCommitSha
	}
	return false
}

func (m *BuildDefinition) GetArgs() map[string]string {
	if m != nil {
		return m.Args
	}
	return nil
}

func (m *BuildDefinition) GetDisableBuildCache() bool {
	if m != nil {
		return m.DisableBuildCache
	}
	return false
}

type PushRegistryDefinition struct {
	Repo                 string   `protobuf:"bytes,1,opt,name=repo,proto3" json:"repo,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *PushRegistryDefinition) Reset()         { *m = PushRegistryDefinition{} }
func (m *PushRegistryDefinition) String() string { return proto.CompactTextString(m) }
func (*PushRegistryDefinition) ProtoMessage()    {}
func (*PushRegistryDefinition) Descriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{2}
}

func (m *PushRegistryDefinition) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PushRegistryDefinition.Unmarshal(m, b)
}
func (m *PushRegistryDefinition) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PushRegistryDefinition.Marshal(b, m, deterministic)
}
func (m *PushRegistryDefinition) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PushRegistryDefinition.Merge(m, src)
}
func (m *PushRegistryDefinition) XXX_Size() int {
	return xxx_messageInfo_PushRegistryDefinition.Size(m)
}
func (m *PushRegistryDefinition) XXX_DiscardUnknown() {
	xxx_messageInfo_PushRegistryDefinition.DiscardUnknown(m)
}

var xxx_messageInfo_PushRegistryDefinition proto.InternalMessageInfo

func (m *PushRegistryDefinition) GetRepo() string {
	if m != nil {
		return m.Repo
	}
	return ""
}

type PushDefinition struct {
	Registries           []*PushRegistryDefinition `protobuf:"bytes,1,rep,name=registries,proto3" json:"registries,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                  `json:"-"`
	XXX_unrecognized     []byte                    `json:"-"`
	XXX_sizecache        int32                     `json:"-"`
}

func (m *PushDefinition) Reset()         { *m = PushDefinition{} }
func (m *PushDefinition) String() string { return proto.CompactTextString(m) }
func (*PushDefinition) ProtoMessage()    {}
func (*PushDefinition) Descriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{3}
}

func (m *PushDefinition) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PushDefinition.Unmarshal(m, b)
}
func (m *PushDefinition) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PushDefinition.Marshal(b, m, deterministic)
}
func (m *PushDefinition) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PushDefinition.Merge(m, src)
}
func (m *PushDefinition) XXX_Size() int {
	return xxx_messageInfo_PushDefinition.Size(m)
}
func (m *PushDefinition) XXX_DiscardUnknown() {
	xxx_messageInfo_PushDefinition.DiscardUnknown(m)
}

var xxx_messageInfo_PushDefinition proto.InternalMessageInfo

func (m *PushDefinition) GetRegistries() []*PushRegistryDefinition {
	if m != nil {
		return m.Registries
	}
	return nil
}

type BuildRequest struct {
	Build                *BuildDefinition `protobuf:"bytes,1,opt,name=build,proto3" json:"build,omitempty"`
	Push                 *PushDefinition  `protobuf:"bytes,2,opt,name=push,proto3" json:"push,omitempty"`
	SkipIfExists         bool             `protobuf:"varint,3,opt,name=skip_if_exists,json=skipIfExists,proto3" json:"skip_if_exists,omitempty"`
	XXX_NoUnkeyedLiteral struct{}         `json:"-"`
	XXX_unrecognized     []byte           `json:"-"`
	XXX_sizecache        int32            `json:"-"`
}

func (m *BuildRequest) Reset()         { *m = BuildRequest{} }
func (m *BuildRequest) String() string { return proto.CompactTextString(m) }
func (*BuildRequest) ProtoMessage()    {}
func (*BuildRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{4}
}

func (m *BuildRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BuildRequest.Unmarshal(m, b)
}
func (m *BuildRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BuildRequest.Marshal(b, m, deterministic)
}
func (m *BuildRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BuildRequest.Merge(m, src)
}
func (m *BuildRequest) XXX_Size() int {
	return xxx_messageInfo_BuildRequest.Size(m)
}
func (m *BuildRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_BuildRequest.DiscardUnknown(m)
}

var xxx_messageInfo_BuildRequest proto.InternalMessageInfo

func (m *BuildRequest) GetBuild() *BuildDefinition {
	if m != nil {
		return m.Build
	}
	return nil
}

func (m *BuildRequest) GetPush() *PushDefinition {
	if m != nil {
		return m.Push
	}
	return nil
}

func (m *BuildRequest) GetSkipIfExists() bool {
	if m != nil {
		return m.SkipIfExists
	}
	return false
}

type BuildStatusRequest struct {
	BuildId              string   `protobuf:"bytes,1,opt,name=build_id,json=buildId,proto3" json:"build_id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *BuildStatusRequest) Reset()         { *m = BuildStatusRequest{} }
func (m *BuildStatusRequest) String() string { return proto.CompactTextString(m) }
func (*BuildStatusRequest) ProtoMessage()    {}
func (*BuildStatusRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{5}
}

func (m *BuildStatusRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BuildStatusRequest.Unmarshal(m, b)
}
func (m *BuildStatusRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BuildStatusRequest.Marshal(b, m, deterministic)
}
func (m *BuildStatusRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BuildStatusRequest.Merge(m, src)
}
func (m *BuildStatusRequest) XXX_Size() int {
	return xxx_messageInfo_BuildStatusRequest.Size(m)
}
func (m *BuildStatusRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_BuildStatusRequest.DiscardUnknown(m)
}

var xxx_messageInfo_BuildStatusRequest proto.InternalMessageInfo

func (m *BuildStatusRequest) GetBuildId() string {
	if m != nil {
		return m.BuildId
	}
	return ""
}

type BuildCancelRequest struct {
	BuildId              string   `protobuf:"bytes,1,opt,name=build_id,json=buildId,proto3" json:"build_id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *BuildCancelRequest) Reset()         { *m = BuildCancelRequest{} }
func (m *BuildCancelRequest) String() string { return proto.CompactTextString(m) }
func (*BuildCancelRequest) ProtoMessage()    {}
func (*BuildCancelRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{6}
}

func (m *BuildCancelRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BuildCancelRequest.Unmarshal(m, b)
}
func (m *BuildCancelRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BuildCancelRequest.Marshal(b, m, deterministic)
}
func (m *BuildCancelRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BuildCancelRequest.Merge(m, src)
}
func (m *BuildCancelRequest) XXX_Size() int {
	return xxx_messageInfo_BuildCancelRequest.Size(m)
}
func (m *BuildCancelRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_BuildCancelRequest.DiscardUnknown(m)
}

var xxx_messageInfo_BuildCancelRequest proto.InternalMessageInfo

func (m *BuildCancelRequest) GetBuildId() string {
	if m != nil {
		return m.BuildId
	}
	return ""
}

type BuildRequestResponse struct {
	BuildId              string   `protobuf:"bytes,1,opt,name=build_id,json=buildId,proto3" json:"build_id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *BuildRequestResponse) Reset()         { *m = BuildRequestResponse{} }
func (m *BuildRequestResponse) String() string { return proto.CompactTextString(m) }
func (*BuildRequestResponse) ProtoMessage()    {}
func (*BuildRequestResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{7}
}

func (m *BuildRequestResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BuildRequestResponse.Unmarshal(m, b)
}
func (m *BuildRequestResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BuildRequestResponse.Marshal(b, m, deterministic)
}
func (m *BuildRequestResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BuildRequestResponse.Merge(m, src)
}
func (m *BuildRequestResponse) XXX_Size() int {
	return xxx_messageInfo_BuildRequestResponse.Size(m)
}
func (m *BuildRequestResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_BuildRequestResponse.DiscardUnknown(m)
}

var xxx_messageInfo_BuildRequestResponse proto.InternalMessageInfo

func (m *BuildRequestResponse) GetBuildId() string {
	if m != nil {
		return m.BuildId
	}
	return ""
}

type BuildCancelResponse struct {
	BuildId              string   `protobuf:"bytes,1,opt,name=build_id,json=buildId,proto3" json:"build_id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *BuildCancelResponse) Reset()         { *m = BuildCancelResponse{} }
func (m *BuildCancelResponse) String() string { return proto.CompactTextString(m) }
func (*BuildCancelResponse) ProtoMessage()    {}
func (*BuildCancelResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{8}
}

func (m *BuildCancelResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BuildCancelResponse.Unmarshal(m, b)
}
func (m *BuildCancelResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BuildCancelResponse.Marshal(b, m, deterministic)
}
func (m *BuildCancelResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BuildCancelResponse.Merge(m, src)
}
func (m *BuildCancelResponse) XXX_Size() int {
	return xxx_messageInfo_BuildCancelResponse.Size(m)
}
func (m *BuildCancelResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_BuildCancelResponse.DiscardUnknown(m)
}

var xxx_messageInfo_BuildCancelResponse proto.InternalMessageInfo

func (m *BuildCancelResponse) GetBuildId() string {
	if m != nil {
		return m.BuildId
	}
	return ""
}

type BuildStatusResponse struct {
	BuildId              string        `protobuf:"bytes,1,opt,name=build_id,json=buildId,proto3" json:"build_id,omitempty"`
	BuildRequest         *BuildRequest `protobuf:"bytes,2,opt,name=build_request,json=buildRequest,proto3" json:"build_request,omitempty"`
	State                BuildState    `protobuf:"varint,3,opt,name=state,proto3,enum=furanrpc.BuildState" json:"state,omitempty"`
	Started              *Timestamp    `protobuf:"bytes,4,opt,name=started,proto3" json:"started,omitempty"`
	Completed            *Timestamp    `protobuf:"bytes,5,opt,name=completed,proto3" json:"completed,omitempty"`
	XXX_NoUnkeyedLiteral struct{}      `json:"-"`
	XXX_unrecognized     []byte        `json:"-"`
	XXX_sizecache        int32         `json:"-"`
}

func (m *BuildStatusResponse) Reset()         { *m = BuildStatusResponse{} }
func (m *BuildStatusResponse) String() string { return proto.CompactTextString(m) }
func (*BuildStatusResponse) ProtoMessage()    {}
func (*BuildStatusResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{9}
}

func (m *BuildStatusResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BuildStatusResponse.Unmarshal(m, b)
}
func (m *BuildStatusResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BuildStatusResponse.Marshal(b, m, deterministic)
}
func (m *BuildStatusResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BuildStatusResponse.Merge(m, src)
}
func (m *BuildStatusResponse) XXX_Size() int {
	return xxx_messageInfo_BuildStatusResponse.Size(m)
}
func (m *BuildStatusResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_BuildStatusResponse.DiscardUnknown(m)
}

var xxx_messageInfo_BuildStatusResponse proto.InternalMessageInfo

func (m *BuildStatusResponse) GetBuildId() string {
	if m != nil {
		return m.BuildId
	}
	return ""
}

func (m *BuildStatusResponse) GetBuildRequest() *BuildRequest {
	if m != nil {
		return m.BuildRequest
	}
	return nil
}

func (m *BuildStatusResponse) GetState() BuildState {
	if m != nil {
		return m.State
	}
	return BuildState_UNKNOWN
}

func (m *BuildStatusResponse) GetStarted() *Timestamp {
	if m != nil {
		return m.Started
	}
	return nil
}

func (m *BuildStatusResponse) GetCompleted() *Timestamp {
	if m != nil {
		return m.Completed
	}
	return nil
}

type BuildEvent struct {
	BuildId              string     `protobuf:"bytes,1,opt,name=build_id,json=buildId,proto3" json:"build_id,omitempty"`
	Message              string     `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
	CurrentState         BuildState `protobuf:"varint,3,opt,name=current_state,json=currentState,proto3,enum=furanrpc.BuildState" json:"current_state,omitempty"`
	XXX_NoUnkeyedLiteral struct{}   `json:"-"`
	XXX_unrecognized     []byte     `json:"-"`
	XXX_sizecache        int32      `json:"-"`
}

func (m *BuildEvent) Reset()         { *m = BuildEvent{} }
func (m *BuildEvent) String() string { return proto.CompactTextString(m) }
func (*BuildEvent) ProtoMessage()    {}
func (*BuildEvent) Descriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{10}
}

func (m *BuildEvent) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BuildEvent.Unmarshal(m, b)
}
func (m *BuildEvent) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BuildEvent.Marshal(b, m, deterministic)
}
func (m *BuildEvent) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BuildEvent.Merge(m, src)
}
func (m *BuildEvent) XXX_Size() int {
	return xxx_messageInfo_BuildEvent.Size(m)
}
func (m *BuildEvent) XXX_DiscardUnknown() {
	xxx_messageInfo_BuildEvent.DiscardUnknown(m)
}

var xxx_messageInfo_BuildEvent proto.InternalMessageInfo

func (m *BuildEvent) GetBuildId() string {
	if m != nil {
		return m.BuildId
	}
	return ""
}

func (m *BuildEvent) GetMessage() string {
	if m != nil {
		return m.Message
	}
	return ""
}

func (m *BuildEvent) GetCurrentState() BuildState {
	if m != nil {
		return m.CurrentState
	}
	return BuildState_UNKNOWN
}

var E_ReadOnly = &proto.ExtensionDesc{
	ExtendedType:  (*descriptorpb.MethodOptions)(nil),
	ExtensionType: (*bool)(nil),
	Field:         1000,
	Name:          "furanrpc.read_only",
	Tag:           "varint,1000,opt,name=read_only",
	Filename:      "api.proto",
}

func init() {
	proto.RegisterEnum("furanrpc.BuildState", BuildState_name, BuildState_value)
	proto.RegisterType((*Timestamp)(nil), "furanrpc.Timestamp")
	proto.RegisterType((*BuildDefinition)(nil), "furanrpc.BuildDefinition")
	proto.RegisterMapType((map[string]string)(nil), "furanrpc.BuildDefinition.ArgsEntry")
	proto.RegisterType((*PushRegistryDefinition)(nil), "furanrpc.PushRegistryDefinition")
	proto.RegisterType((*PushDefinition)(nil), "furanrpc.PushDefinition")
	proto.RegisterType((*BuildRequest)(nil), "furanrpc.BuildRequest")
	proto.RegisterType((*BuildStatusRequest)(nil), "furanrpc.BuildStatusRequest")
	proto.RegisterType((*BuildCancelRequest)(nil), "furanrpc.BuildCancelRequest")
	proto.RegisterType((*BuildRequestResponse)(nil), "furanrpc.BuildRequestResponse")
	proto.RegisterType((*BuildCancelResponse)(nil), "furanrpc.BuildCancelResponse")
	proto.RegisterType((*BuildStatusResponse)(nil), "furanrpc.BuildStatusResponse")
	proto.RegisterType((*BuildEvent)(nil), "furanrpc.BuildEvent")
	proto.RegisterExtension(E_ReadOnly)
}

func init() { proto.RegisterFile("api.proto", fileDescriptor_00212fb1f9d3bf1c) }

var fileDescriptor_00212fb1f9d3bf1c = []byte{
	// 900 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x55, 0xdd, 0x6e, 0xdb, 0x36,
	0x14, 0xae, 0x62, 0xbb, 0xb6, 0x8f, 0x13, 0x57, 0x65, 0x82, 0x42, 0x0d, 0xb6, 0xce, 0xf0, 0x06,
	0xcc, 0xe8, 0x5a, 0xa7, 0xf5, 0x2e, 0xba, 0xb5, 0xc3, 0xb0, 0xcc, 0x51, 0x0b, 0xa3, 0xa9, 0x9d,
	0xd1, 0x31, 0x7a, 0x29, 0xd0, 0x12, 0x2d, 0x11, 0x91, 0x45, 0x8d, 0xa4, 0xba, 0x1a, 0xd8, 0xd5,
	0x5e, 0x61, 0x2f, 0xb4, 0xeb, 0x3d, 0xc9, 0xde, 0x61, 0x17, 0x1b, 0x48, 0xc9, 0x3f, 0xc9, 0xe2,
	0x64, 0x77, 0x3c, 0xe7, 0x7c, 0xdf, 0xf9, 0xfb, 0x44, 0x0a, 0xea, 0x24, 0x65, 0xdd, 0x54, 0x70,
	0xc5, 0x51, 0x6d, 0x96, 0x09, 0x92, 0x88, 0xd4, 0x3f, 0x6c, 0x85, 0x9c, 0x87, 0x31, 0x3d, 0x32,
	0xfe, 0x69, 0x36, 0x3b, 0x0a, 0xa8, 0xf4, 0x05, 0x4b, 0x15, 0x17, 0x39, 0xb6, 0xfd, 0x0a, 0xea,
	0xe7, 0x6c, 0x4e, 0xa5, 0x22, 0xf3, 0x14, 0x39, 0x50, 0x95, 0xd4, 0xe7, 0x49, 0x20, 0x1d, 0xab,
	0x65, 0x75, 0x4a, 0x78, 0x69, 0xa2, 0x03, 0xa8, 0x24, 0x24, 0xe1, 0xd2, 0xd9, 0x69, 0x59, 0x9d,
	0x0a, 0xce, 0x8d, 0xf6, 0x3f, 0x3b, 0x70, 0xef, 0xc7, 0x8c, 0xc5, 0xc1, 0x09, 0x9d, 0xb1, 0x84,
	0x29, 0xc6, 0x13, 0xf4, 0x19, 0x34, 0x42, 0xa6, 0xa2, 0x6c, 0xea, 0x09, 0x9a, 0x72, 0x93, 0xa7,
	0x8e, 0x21, 0x77, 0x61, 0x9a, 0x72, 0xf4, 0x15, 0xdc, 0x2f, 0x00, 0xbe, 0xa0, 0x01, 0x4d, 0x14,
	0x23, 0xb1, 0x49, 0x5b, 0xc7, 0x76, 0x1e, 0xe8, 0xaf, 0xfc, 0xe8, 0x4b, 0xb8, 0x17, 0x70, 0xff,
	0x82, 0x8a, 0x19, 0x8b, 0xa9, 0x97, 0x12, 0x15, 0x39, 0x25, 0x03, 0x6d, 0xae, 0xdd, 0x67, 0x44,
	0x45, 0xc8, 0x86, 0x92, 0xa0, 0x33, 0xa7, 0x6c, 0x82, 0xfa, 0x88, 0x10, 0x94, 0x15, 0x09, 0xa5,
	0x53, 0x69, 0x95, 0x3a, 0x75, 0x6c, 0xce, 0xe8, 0x29, 0xec, 0x2b, 0x12, 0x7a, 0xbf, 0x30, 0x15,
	0x79, 0x3e, 0x9f, 0xcf, 0x99, 0xf2, 0x64, 0x44, 0x9c, 0xbb, 0x2d, 0xab, 0x53, 0xc3, 0xb6, 0x22,
	0xe1, 0x7b, 0xa6, 0xa2, 0xbe, 0x09, 0x8c, 0x23, 0x82, 0x5e, 0x40, 0x99, 0x88, 0x50, 0x3a, 0xd5,
	0x56, 0xa9, 0xd3, 0xe8, 0x7d, 0xde, 0x5d, 0xee, 0xb5, 0x7b, 0x65, 0xe8, 0xee, 0xb1, 0x08, 0xa5,
	0x9b, 0x28, 0xb1, 0xc0, 0x86, 0x80, 0xba, 0xb0, 0x1f, 0x30, 0x49, 0xa6, 0x31, 0xf5, 0xa6, 0x1a,
	0xea, 0xf9, 0xc4, 0x8f, 0xa8, 0x53, 0x33, 0x75, 0xee, 0x17, 0x21, 0x93, 0xa4, 0xaf, 0x03, 0x87,
	0x2f, 0xa0, 0xbe, 0x4a, 0xa1, 0x47, 0xb9, 0xa0, 0x8b, 0x62, 0x73, 0xfa, 0xa8, 0xb7, 0xff, 0x81,
	0xc4, 0x19, 0x2d, 0xd6, 0x94, 0x1b, 0x2f, 0x77, 0xbe, 0xb1, 0xda, 0x4f, 0xe0, 0xc1, 0x59, 0x26,
	0x23, 0x4c, 0x43, 0x26, 0x95, 0x58, 0x6c, 0xe8, 0x80, 0xa0, 0xbc, 0x21, 0x80, 0x39, 0xb7, 0x31,
	0x34, 0x35, 0x7a, 0x03, 0xf5, 0x03, 0x80, 0xc8, 0xb9, 0x8c, 0x6a, 0xd1, 0xf5, 0x9c, 0xad, 0xf5,
	0x9c, 0xd7, 0xe7, 0xc6, 0x1b, 0x9c, 0xf6, 0xef, 0x16, 0xec, 0x9a, 0x49, 0x30, 0xfd, 0x39, 0xa3,
	0x52, 0xa1, 0x23, 0xa8, 0x98, 0x99, 0x4d, 0xe5, 0x46, 0xef, 0xe1, 0xd6, 0xad, 0xe1, 0x1c, 0x87,
	0x9e, 0x40, 0x39, 0xcd, 0x64, 0x64, 0x86, 0x6b, 0xf4, 0x9c, 0xcb, 0xd5, 0x37, 0xe0, 0x06, 0x85,
	0xbe, 0x80, 0xa6, 0xbc, 0x60, 0xa9, 0xc7, 0x66, 0x1e, 0xfd, 0xc8, 0xa4, 0x92, 0xe6, 0x83, 0xa8,
	0xe1, 0x5d, 0xed, 0x1d, 0xcc, 0x5c, 0xe3, 0x6b, 0x1f, 0x01, 0x32, 0xd5, 0xc6, 0x8a, 0xa8, 0x4c,
	0x2e, 0x5b, 0x7b, 0x08, 0xb5, 0x5c, 0x0e, 0x16, 0x14, 0x7b, 0xa9, 0x1a, 0x7b, 0x10, 0xac, 0x08,
	0x7d, 0x92, 0xf8, 0x34, 0xfe, 0x1f, 0x84, 0xe7, 0x70, 0xb0, 0x39, 0x36, 0xa6, 0x32, 0xe5, 0x89,
	0xa4, 0x37, 0x51, 0x9e, 0xc1, 0xfe, 0xa5, 0x1a, 0xb7, 0x33, 0xfe, 0xb6, 0x0a, 0xca, 0x72, 0x8e,
	0x5b, 0x29, 0xe8, 0x15, 0xec, 0xe5, 0x21, 0x91, 0x37, 0x56, 0xac, 0xf5, 0xc1, 0x15, 0x19, 0x96,
	0x6d, 0xef, 0x4e, 0x37, 0xb5, 0x7b, 0x0c, 0x15, 0xa9, 0x88, 0xa2, 0x66, 0xa7, 0xcd, 0xde, 0xc1,
	0x15, 0x92, 0xee, 0x82, 0xe2, 0x1c, 0x82, 0x9e, 0x42, 0x55, 0x2a, 0x22, 0x14, 0x0d, 0xcc, 0xad,
	0x6b, 0xf4, 0xf6, 0xd7, 0xe8, 0xd5, 0x93, 0x82, 0x97, 0x18, 0xf4, 0x1c, 0xea, 0x3e, 0x9f, 0xa7,
	0x31, 0xd5, 0x84, 0xca, 0x76, 0xc2, 0x1a, 0xd5, 0xfe, 0x15, 0xc0, 0x94, 0x75, 0x3f, 0xd0, 0xe4,
	0x26, 0x2d, 0xf4, 0xbb, 0x35, 0xa7, 0x52, 0x92, 0x70, 0x79, 0x43, 0x96, 0x26, 0xfa, 0x16, 0xf6,
	0xfc, 0x4c, 0x08, 0x9a, 0x28, 0xef, 0xf6, 0xc1, 0x76, 0x0b, 0xa8, 0xb1, 0x1e, 0xff, 0x66, 0x15,
	0xe5, 0x8d, 0x89, 0x1a, 0x50, 0x9d, 0x0c, 0xdf, 0x0e, 0x47, 0xef, 0x87, 0xf6, 0x1d, 0xd4, 0x04,
	0x18, 0x8e, 0xce, 0xc7, 0xe7, 0xc7, 0xf8, 0xdc, 0x3d, 0xb1, 0x2d, 0x1d, 0x1c, 0xbf, 0x1d, 0x9c,
	0x9d, 0xb9, 0x27, 0xf6, 0x8e, 0x36, 0xf0, 0x64, 0x38, 0x1c, 0x0c, 0xdf, 0xd8, 0x25, 0x6d, 0xbc,
	0x3e, 0x1e, 0x9c, 0x4e, 0xb0, 0x6b, 0x97, 0x0d, 0x6c, 0xd2, 0xef, 0xbb, 0xe3, 0xb1, 0x5d, 0x41,
	0x07, 0x60, 0xf7, 0x8f, 0x87, 0x7d, 0xf7, 0xd4, 0xc3, 0xee, 0x4f, 0x13, 0x77, 0xac, 0x33, 0xdd,
	0x45, 0x7b, 0x50, 0xcf, 0xbd, 0xa7, 0xee, 0x89, 0x5d, 0xed, 0xfd, 0xb9, 0x03, 0x7b, 0xaf, 0x75,
	0xab, 0xee, 0x47, 0xea, 0x67, 0x8a, 0x0b, 0x34, 0x00, 0x18, 0xeb, 0x95, 0x9a, 0xd6, 0xd0, 0x16,
	0x59, 0x0f, 0x1f, 0x6d, 0x91, 0xbb, 0xf8, 0x80, 0xda, 0xa5, 0x3f, 0xbe, 0xbf, 0x83, 0x30, 0x34,
	0xdf, 0x50, 0xb5, 0xf1, 0x7d, 0xa1, 0x4f, 0xae, 0xd9, 0xcb, 0xea, 0xfa, 0x1c, 0x7e, 0xba, 0x25,
	0xba, 0xce, 0x69, 0xa1, 0x01, 0xec, 0xbe, 0xe3, 0x09, 0x53, 0x5c, 0xe4, 0x0d, 0xde, 0x9c, 0xf1,
	0xaa, 0x0e, 0x46, 0x69, 0x93, 0xe8, 0x99, 0x85, 0x46, 0xd0, 0xc8, 0x6f, 0xca, 0xf5, 0x99, 0x2e,
	0xdd, 0xd4, 0xff, 0xf4, 0x76, 0xf9, 0x8e, 0x99, 0x79, 0x5f, 0x7e, 0x07, 0x75, 0x41, 0x49, 0xe0,
	0xf1, 0x24, 0x5e, 0xa0, 0x47, 0xdd, 0xfc, 0xdf, 0xd8, 0x5d, 0xfe, 0x1b, 0xbb, 0xef, 0xa8, 0x8a,
	0x78, 0x30, 0x4a, 0xf5, 0x63, 0x23, 0x9d, 0xbf, 0xaa, 0xe6, 0x59, 0xa9, 0x69, 0xc6, 0x28, 0x89,
	0x17, 0xd3, 0xbb, 0x06, 0xf8, 0xf5, 0xbf, 0x01, 0x00, 0x00, 0xff, 0xff, 0x84, 0xf1, 0x25, 0xc8,
	0x69, 0x07, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// FuranExecutorClient is the client API for FuranExecutor service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type FuranExecutorClient interface {
	StartBuild(ctx context.Context, in *BuildRequest, opts ...grpc.CallOption) (*BuildRequestResponse, error)
	GetBuildStatus(ctx context.Context, in *BuildStatusRequest, opts ...grpc.CallOption) (*BuildStatusResponse, error)
	MonitorBuild(ctx context.Context, in *BuildStatusRequest, opts ...grpc.CallOption) (FuranExecutor_MonitorBuildClient, error)
	CancelBuild(ctx context.Context, in *BuildCancelRequest, opts ...grpc.CallOption) (*BuildCancelResponse, error)
}

type furanExecutorClient struct {
	cc *grpc.ClientConn
}

func NewFuranExecutorClient(cc *grpc.ClientConn) FuranExecutorClient {
	return &furanExecutorClient{cc}
}

func (c *furanExecutorClient) StartBuild(ctx context.Context, in *BuildRequest, opts ...grpc.CallOption) (*BuildRequestResponse, error) {
	out := new(BuildRequestResponse)
	err := c.cc.Invoke(ctx, "/furanrpc.FuranExecutor/StartBuild", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *furanExecutorClient) GetBuildStatus(ctx context.Context, in *BuildStatusRequest, opts ...grpc.CallOption) (*BuildStatusResponse, error) {
	out := new(BuildStatusResponse)
	err := c.cc.Invoke(ctx, "/furanrpc.FuranExecutor/GetBuildStatus", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *furanExecutorClient) MonitorBuild(ctx context.Context, in *BuildStatusRequest, opts ...grpc.CallOption) (FuranExecutor_MonitorBuildClient, error) {
	stream, err := c.cc.NewStream(ctx, &_FuranExecutor_serviceDesc.Streams[0], "/furanrpc.FuranExecutor/MonitorBuild", opts...)
	if err != nil {
		return nil, err
	}
	x := &furanExecutorMonitorBuildClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type FuranExecutor_MonitorBuildClient interface {
	Recv() (*BuildEvent, error)
	grpc.ClientStream
}

type furanExecutorMonitorBuildClient struct {
	grpc.ClientStream
}

func (x *furanExecutorMonitorBuildClient) Recv() (*BuildEvent, error) {
	m := new(BuildEvent)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *furanExecutorClient) CancelBuild(ctx context.Context, in *BuildCancelRequest, opts ...grpc.CallOption) (*BuildCancelResponse, error) {
	out := new(BuildCancelResponse)
	err := c.cc.Invoke(ctx, "/furanrpc.FuranExecutor/CancelBuild", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// FuranExecutorServer is the server API for FuranExecutor service.
type FuranExecutorServer interface {
	StartBuild(context.Context, *BuildRequest) (*BuildRequestResponse, error)
	GetBuildStatus(context.Context, *BuildStatusRequest) (*BuildStatusResponse, error)
	MonitorBuild(*BuildStatusRequest, FuranExecutor_MonitorBuildServer) error
	CancelBuild(context.Context, *BuildCancelRequest) (*BuildCancelResponse, error)
}

// UnimplementedFuranExecutorServer can be embedded to have forward compatible implementations.
type UnimplementedFuranExecutorServer struct {
}

func (*UnimplementedFuranExecutorServer) StartBuild(ctx context.Context, req *BuildRequest) (*BuildRequestResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StartBuild not implemented")
}
func (*UnimplementedFuranExecutorServer) GetBuildStatus(ctx context.Context, req *BuildStatusRequest) (*BuildStatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetBuildStatus not implemented")
}
func (*UnimplementedFuranExecutorServer) MonitorBuild(req *BuildStatusRequest, srv FuranExecutor_MonitorBuildServer) error {
	return status.Errorf(codes.Unimplemented, "method MonitorBuild not implemented")
}
func (*UnimplementedFuranExecutorServer) CancelBuild(ctx context.Context, req *BuildCancelRequest) (*BuildCancelResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CancelBuild not implemented")
}

func RegisterFuranExecutorServer(s *grpc.Server, srv FuranExecutorServer) {
	s.RegisterService(&_FuranExecutor_serviceDesc, srv)
}

func _FuranExecutor_StartBuild_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BuildRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FuranExecutorServer).StartBuild(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/furanrpc.FuranExecutor/StartBuild",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FuranExecutorServer).StartBuild(ctx, req.(*BuildRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FuranExecutor_GetBuildStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BuildStatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FuranExecutorServer).GetBuildStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/furanrpc.FuranExecutor/GetBuildStatus",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FuranExecutorServer).GetBuildStatus(ctx, req.(*BuildStatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FuranExecutor_MonitorBuild_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(BuildStatusRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(FuranExecutorServer).MonitorBuild(m, &furanExecutorMonitorBuildServer{stream})
}

type FuranExecutor_MonitorBuildServer interface {
	Send(*BuildEvent) error
	grpc.ServerStream
}

type furanExecutorMonitorBuildServer struct {
	grpc.ServerStream
}

func (x *furanExecutorMonitorBuildServer) Send(m *BuildEvent) error {
	return x.ServerStream.SendMsg(m)
}

func _FuranExecutor_CancelBuild_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BuildCancelRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FuranExecutorServer).CancelBuild(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/furanrpc.FuranExecutor/CancelBuild",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FuranExecutorServer).CancelBuild(ctx, req.(*BuildCancelRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _FuranExecutor_serviceDesc = grpc.ServiceDesc{
	ServiceName: "furanrpc.FuranExecutor",
	HandlerType: (*FuranExecutorServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "StartBuild",
			Handler:    _FuranExecutor_StartBuild_Handler,
		},
		{
			MethodName: "GetBuildStatus",
			Handler:    _FuranExecutor_GetBuildStatus_Handler,
		},
		{
			MethodName: "CancelBuild",
			Handler:    _FuranExecutor_CancelBuild_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "MonitorBuild",
			Handler:       _FuranExecutor_MonitorBuild_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "api.proto",
}
