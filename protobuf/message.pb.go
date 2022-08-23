// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.21.5
// source: message.proto

package protobuf

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

type Request_Type int32

const (
	Request_CREATE  Request_Type = 0
	Request_LINK    Request_Type = 1
	Request_MKDIR   Request_Type = 2
	Request_RENAME  Request_Type = 3
	Request_RMDIR   Request_Type = 4
	Request_SETATTR Request_Type = 5
	Request_SYMLINK Request_Type = 6
	Request_UNLINK  Request_Type = 7
	Request_WRITE   Request_Type = 8
)

// Enum value maps for Request_Type.
var (
	Request_Type_name = map[int32]string{
		0: "CREATE",
		1: "LINK",
		2: "MKDIR",
		3: "RENAME",
		4: "RMDIR",
		5: "SETATTR",
		6: "SYMLINK",
		7: "UNLINK",
		8: "WRITE",
	}
	Request_Type_value = map[string]int32{
		"CREATE":  0,
		"LINK":    1,
		"MKDIR":   2,
		"RENAME":  3,
		"RMDIR":   4,
		"SETATTR": 5,
		"SYMLINK": 6,
		"UNLINK":  7,
		"WRITE":   8,
	}
)

func (x Request_Type) Enum() *Request_Type {
	p := new(Request_Type)
	*p = x
	return p
}

func (x Request_Type) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Request_Type) Descriptor() protoreflect.EnumDescriptor {
	return file_message_proto_enumTypes[0].Descriptor()
}

func (Request_Type) Type() protoreflect.EnumType {
	return &file_message_proto_enumTypes[0]
}

func (x Request_Type) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Request_Type.Descriptor instead.
func (Request_Type) EnumDescriptor() ([]byte, []int) {
	return file_message_proto_rawDescGZIP(), []int{1, 0}
}

type Response_Type int32

const (
	Response_ACK  Response_Type = 0
	Response_NACK Response_Type = 1
)

// Enum value maps for Response_Type.
var (
	Response_Type_name = map[int32]string{
		0: "ACK",
		1: "NACK",
	}
	Response_Type_value = map[string]int32{
		"ACK":  0,
		"NACK": 1,
	}
)

func (x Response_Type) Enum() *Response_Type {
	p := new(Response_Type)
	*p = x
	return p
}

func (x Response_Type) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Response_Type) Descriptor() protoreflect.EnumDescriptor {
	return file_message_proto_enumTypes[1].Descriptor()
}

func (Response_Type) Type() protoreflect.EnumType {
	return &file_message_proto_enumTypes[1]
}

func (x Response_Type) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Response_Type.Descriptor instead.
func (Response_Type) EnumDescriptor() ([]byte, []int) {
	return file_message_proto_rawDescGZIP(), []int{2, 0}
}

type Response_Error int32

const (
	Response_ERR_UNKNOWN              Response_Error = 0
	Response_ERR_ALREADY_EXISTS       Response_Error = 1
	Response_ERR_DOES_NOT_EXIST       Response_Error = 2
	Response_ERR_TRANSACTION_CONFLICT Response_Error = 3
	Response_ERR_NOT_A_DIRECTORY      Response_Error = 4
)

// Enum value maps for Response_Error.
var (
	Response_Error_name = map[int32]string{
		0: "ERR_UNKNOWN",
		1: "ERR_ALREADY_EXISTS",
		2: "ERR_DOES_NOT_EXIST",
		3: "ERR_TRANSACTION_CONFLICT",
		4: "ERR_NOT_A_DIRECTORY",
	}
	Response_Error_value = map[string]int32{
		"ERR_UNKNOWN":              0,
		"ERR_ALREADY_EXISTS":       1,
		"ERR_DOES_NOT_EXIST":       2,
		"ERR_TRANSACTION_CONFLICT": 3,
		"ERR_NOT_A_DIRECTORY":      4,
	}
)

func (x Response_Error) Enum() *Response_Error {
	p := new(Response_Error)
	*p = x
	return p
}

func (x Response_Error) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Response_Error) Descriptor() protoreflect.EnumDescriptor {
	return file_message_proto_enumTypes[2].Descriptor()
}

func (Response_Error) Type() protoreflect.EnumType {
	return &file_message_proto_enumTypes[2]
}

func (x Response_Error) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Response_Error.Descriptor instead.
func (Response_Error) EnumDescriptor() ([]byte, []int) {
	return file_message_proto_rawDescGZIP(), []int{2, 1}
}

type Message struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// tid is transaction id.
	Tid      string `protobuf:"bytes,1,opt,name=tid,proto3" json:"tid,omitempty"`
	PeerName string `protobuf:"bytes,2,opt,name=peer_name,json=peerName,proto3" json:"peer_name,omitempty"`
	// Types that are assignable to Type:
	//
	//	*Message_Request
	//	*Message_Response
	Type isMessage_Type `protobuf_oneof:"type"`
}

func (x *Message) Reset() {
	*x = Message{}
	if protoimpl.UnsafeEnabled {
		mi := &file_message_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Message) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Message) ProtoMessage() {}

func (x *Message) ProtoReflect() protoreflect.Message {
	mi := &file_message_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Message.ProtoReflect.Descriptor instead.
func (*Message) Descriptor() ([]byte, []int) {
	return file_message_proto_rawDescGZIP(), []int{0}
}

func (x *Message) GetTid() string {
	if x != nil {
		return x.Tid
	}
	return ""
}

func (x *Message) GetPeerName() string {
	if x != nil {
		return x.PeerName
	}
	return ""
}

func (m *Message) GetType() isMessage_Type {
	if m != nil {
		return m.Type
	}
	return nil
}

func (x *Message) GetRequest() *Request {
	if x, ok := x.GetType().(*Message_Request); ok {
		return x.Request
	}
	return nil
}

func (x *Message) GetResponse() *Response {
	if x, ok := x.GetType().(*Message_Response); ok {
		return x.Response
	}
	return nil
}

type isMessage_Type interface {
	isMessage_Type()
}

type Message_Request struct {
	Request *Request `protobuf:"bytes,3,opt,name=request,proto3,oneof"`
}

type Message_Response struct {
	Response *Response `protobuf:"bytes,4,opt,name=response,proto3,oneof"`
}

func (*Message_Request) isMessage_Type() {}

func (*Message_Response) isMessage_Type() {}

type Request struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Type     Request_Type      `protobuf:"varint,1,opt,name=type,proto3,enum=message.Request_Type" json:"type,omitempty"`
	Metadata *Request_Metadata `protobuf:"bytes,2,opt,name=metadata,proto3" json:"metadata,omitempty"`
	Clock    uint64            `protobuf:"varint,3,opt,name=clock,proto3" json:"clock,omitempty"`
	Content  []byte            `protobuf:"bytes,4,opt,name=content,proto3,oneof" json:"content,omitempty"`
}

func (x *Request) Reset() {
	*x = Request{}
	if protoimpl.UnsafeEnabled {
		mi := &file_message_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Request) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Request) ProtoMessage() {}

func (x *Request) ProtoReflect() protoreflect.Message {
	mi := &file_message_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Request.ProtoReflect.Descriptor instead.
func (*Request) Descriptor() ([]byte, []int) {
	return file_message_proto_rawDescGZIP(), []int{1}
}

func (x *Request) GetType() Request_Type {
	if x != nil {
		return x.Type
	}
	return Request_CREATE
}

func (x *Request) GetMetadata() *Request_Metadata {
	if x != nil {
		return x.Metadata
	}
	return nil
}

func (x *Request) GetClock() uint64 {
	if x != nil {
		return x.Clock
	}
	return 0
}

func (x *Request) GetContent() []byte {
	if x != nil {
		return x.Content
	}
	return nil
}

type Response struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Type     Response_Type   `protobuf:"varint,1,opt,name=type,proto3,enum=message.Response_Type" json:"type,omitempty"`
	Error    *Response_Error `protobuf:"varint,2,opt,name=error,proto3,enum=message.Response_Error,oneof" json:"error,omitempty"`
	ErrorMsg *string         `protobuf:"bytes,3,opt,name=error_msg,json=errorMsg,proto3,oneof" json:"error_msg,omitempty"`
}

func (x *Response) Reset() {
	*x = Response{}
	if protoimpl.UnsafeEnabled {
		mi := &file_message_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Response) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Response) ProtoMessage() {}

func (x *Response) ProtoReflect() protoreflect.Message {
	mi := &file_message_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Response.ProtoReflect.Descriptor instead.
func (*Response) Descriptor() ([]byte, []int) {
	return file_message_proto_rawDescGZIP(), []int{2}
}

func (x *Response) GetType() Response_Type {
	if x != nil {
		return x.Type
	}
	return Response_ACK
}

func (x *Response) GetError() Response_Error {
	if x != nil && x.Error != nil {
		return *x.Error
	}
	return Response_ERR_UNKNOWN
}

func (x *Response) GetErrorMsg() string {
	if x != nil && x.ErrorMsg != nil {
		return *x.ErrorMsg
	}
	return ""
}

type Request_Metadata struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	RelativePath    string `protobuf:"bytes,1,opt,name=relative_path,json=relativePath,proto3" json:"relative_path,omitempty"`
	NewRelativePath string `protobuf:"bytes,2,opt,name=new_relative_path,json=newRelativePath,proto3" json:"new_relative_path,omitempty"`
	Mode            uint32 `protobuf:"varint,3,opt,name=mode,proto3" json:"mode,omitempty"`
}

func (x *Request_Metadata) Reset() {
	*x = Request_Metadata{}
	if protoimpl.UnsafeEnabled {
		mi := &file_message_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Request_Metadata) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Request_Metadata) ProtoMessage() {}

func (x *Request_Metadata) ProtoReflect() protoreflect.Message {
	mi := &file_message_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Request_Metadata.ProtoReflect.Descriptor instead.
func (*Request_Metadata) Descriptor() ([]byte, []int) {
	return file_message_proto_rawDescGZIP(), []int{1, 0}
}

func (x *Request_Metadata) GetRelativePath() string {
	if x != nil {
		return x.RelativePath
	}
	return ""
}

func (x *Request_Metadata) GetNewRelativePath() string {
	if x != nil {
		return x.NewRelativePath
	}
	return ""
}

func (x *Request_Metadata) GetMode() uint32 {
	if x != nil {
		return x.Mode
	}
	return 0
}

var File_message_proto protoreflect.FileDescriptor

var file_message_proto_rawDesc = []byte{
	0x0a, 0x0d, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x22, 0x9f, 0x01, 0x0a, 0x07, 0x4d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x12, 0x10, 0x0a, 0x03, 0x74, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x03, 0x74, 0x69, 0x64, 0x12, 0x1b, 0x0a, 0x09, 0x70, 0x65, 0x65, 0x72, 0x5f, 0x6e,
	0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x70, 0x65, 0x65, 0x72, 0x4e,
	0x61, 0x6d, 0x65, 0x12, 0x2c, 0x0a, 0x07, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2e, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x48, 0x00, 0x52, 0x07, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x12, 0x2f, 0x0a, 0x08, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x18, 0x04, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x11, 0x2e, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2e, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x48, 0x00, 0x52, 0x08, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x42, 0x06, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x22, 0x8e, 0x03, 0x0a, 0x07, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x29, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0e, 0x32, 0x15, 0x2e, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2e, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74, 0x79, 0x70,
	0x65, 0x12, 0x35, 0x0a, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2e, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x52, 0x08,
	0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x12, 0x14, 0x0a, 0x05, 0x63, 0x6c, 0x6f, 0x63,
	0x6b, 0x18, 0x03, 0x20, 0x01, 0x28, 0x04, 0x52, 0x05, 0x63, 0x6c, 0x6f, 0x63, 0x6b, 0x12, 0x1d,
	0x0a, 0x07, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0c, 0x48,
	0x00, 0x52, 0x07, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x88, 0x01, 0x01, 0x1a, 0x6f, 0x0a,
	0x08, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x12, 0x23, 0x0a, 0x0d, 0x72, 0x65, 0x6c,
	0x61, 0x74, 0x69, 0x76, 0x65, 0x5f, 0x70, 0x61, 0x74, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x0c, 0x72, 0x65, 0x6c, 0x61, 0x74, 0x69, 0x76, 0x65, 0x50, 0x61, 0x74, 0x68, 0x12, 0x2a,
	0x0a, 0x11, 0x6e, 0x65, 0x77, 0x5f, 0x72, 0x65, 0x6c, 0x61, 0x74, 0x69, 0x76, 0x65, 0x5f, 0x70,
	0x61, 0x74, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0f, 0x6e, 0x65, 0x77, 0x52, 0x65,
	0x6c, 0x61, 0x74, 0x69, 0x76, 0x65, 0x50, 0x61, 0x74, 0x68, 0x12, 0x12, 0x0a, 0x04, 0x6d, 0x6f,
	0x64, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x04, 0x6d, 0x6f, 0x64, 0x65, 0x22, 0x6f,
	0x0a, 0x04, 0x54, 0x79, 0x70, 0x65, 0x12, 0x0a, 0x0a, 0x06, 0x43, 0x52, 0x45, 0x41, 0x54, 0x45,
	0x10, 0x00, 0x12, 0x08, 0x0a, 0x04, 0x4c, 0x49, 0x4e, 0x4b, 0x10, 0x01, 0x12, 0x09, 0x0a, 0x05,
	0x4d, 0x4b, 0x44, 0x49, 0x52, 0x10, 0x02, 0x12, 0x0a, 0x0a, 0x06, 0x52, 0x45, 0x4e, 0x41, 0x4d,
	0x45, 0x10, 0x03, 0x12, 0x09, 0x0a, 0x05, 0x52, 0x4d, 0x44, 0x49, 0x52, 0x10, 0x04, 0x12, 0x0b,
	0x0a, 0x07, 0x53, 0x45, 0x54, 0x41, 0x54, 0x54, 0x52, 0x10, 0x05, 0x12, 0x0b, 0x0a, 0x07, 0x53,
	0x59, 0x4d, 0x4c, 0x49, 0x4e, 0x4b, 0x10, 0x06, 0x12, 0x0a, 0x0a, 0x06, 0x55, 0x4e, 0x4c, 0x49,
	0x4e, 0x4b, 0x10, 0x07, 0x12, 0x09, 0x0a, 0x05, 0x57, 0x52, 0x49, 0x54, 0x45, 0x10, 0x08, 0x42,
	0x0a, 0x0a, 0x08, 0x5f, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x22, 0xc0, 0x02, 0x0a, 0x08,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x2a, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x16, 0x2e, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65,
	0x2e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x2e, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04,
	0x74, 0x79, 0x70, 0x65, 0x12, 0x32, 0x0a, 0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x0e, 0x32, 0x17, 0x2e, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2e, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x2e, 0x45, 0x72, 0x72, 0x6f, 0x72, 0x48, 0x00, 0x52, 0x05,
	0x65, 0x72, 0x72, 0x6f, 0x72, 0x88, 0x01, 0x01, 0x12, 0x20, 0x0a, 0x09, 0x65, 0x72, 0x72, 0x6f,
	0x72, 0x5f, 0x6d, 0x73, 0x67, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x48, 0x01, 0x52, 0x08, 0x65,
	0x72, 0x72, 0x6f, 0x72, 0x4d, 0x73, 0x67, 0x88, 0x01, 0x01, 0x22, 0x19, 0x0a, 0x04, 0x54, 0x79,
	0x70, 0x65, 0x12, 0x07, 0x0a, 0x03, 0x41, 0x43, 0x4b, 0x10, 0x00, 0x12, 0x08, 0x0a, 0x04, 0x4e,
	0x41, 0x43, 0x4b, 0x10, 0x01, 0x22, 0x7f, 0x0a, 0x05, 0x45, 0x72, 0x72, 0x6f, 0x72, 0x12, 0x0f,
	0x0a, 0x0b, 0x45, 0x52, 0x52, 0x5f, 0x55, 0x4e, 0x4b, 0x4e, 0x4f, 0x57, 0x4e, 0x10, 0x00, 0x12,
	0x16, 0x0a, 0x12, 0x45, 0x52, 0x52, 0x5f, 0x41, 0x4c, 0x52, 0x45, 0x41, 0x44, 0x59, 0x5f, 0x45,
	0x58, 0x49, 0x53, 0x54, 0x53, 0x10, 0x01, 0x12, 0x16, 0x0a, 0x12, 0x45, 0x52, 0x52, 0x5f, 0x44,
	0x4f, 0x45, 0x53, 0x5f, 0x4e, 0x4f, 0x54, 0x5f, 0x45, 0x58, 0x49, 0x53, 0x54, 0x10, 0x02, 0x12,
	0x1c, 0x0a, 0x18, 0x45, 0x52, 0x52, 0x5f, 0x54, 0x52, 0x41, 0x4e, 0x53, 0x41, 0x43, 0x54, 0x49,
	0x4f, 0x4e, 0x5f, 0x43, 0x4f, 0x4e, 0x46, 0x4c, 0x49, 0x43, 0x54, 0x10, 0x03, 0x12, 0x17, 0x0a,
	0x13, 0x45, 0x52, 0x52, 0x5f, 0x4e, 0x4f, 0x54, 0x5f, 0x41, 0x5f, 0x44, 0x49, 0x52, 0x45, 0x43,
	0x54, 0x4f, 0x52, 0x59, 0x10, 0x04, 0x42, 0x08, 0x0a, 0x06, 0x5f, 0x65, 0x72, 0x72, 0x6f, 0x72,
	0x42, 0x0c, 0x0a, 0x0a, 0x5f, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x5f, 0x6d, 0x73, 0x67, 0x42, 0x3d,
	0x5a, 0x3b, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x5a, 0x61, 0x6d,
	0x65, 0x72, 0x79, 0x6b, 0x61, 0x6e, 0x69, 0x7a, 0x6f, 0x77, 0x61, 0x6e, 0x61, 0x2f, 0x72, 0x65,
	0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x65, 0x64, 0x2d, 0x66, 0x69, 0x6c, 0x65, 0x2d, 0x73, 0x79,
	0x73, 0x74, 0x65, 0x6d, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_message_proto_rawDescOnce sync.Once
	file_message_proto_rawDescData = file_message_proto_rawDesc
)

func file_message_proto_rawDescGZIP() []byte {
	file_message_proto_rawDescOnce.Do(func() {
		file_message_proto_rawDescData = protoimpl.X.CompressGZIP(file_message_proto_rawDescData)
	})
	return file_message_proto_rawDescData
}

var file_message_proto_enumTypes = make([]protoimpl.EnumInfo, 3)
var file_message_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_message_proto_goTypes = []interface{}{
	(Request_Type)(0),        // 0: message.Request.Type
	(Response_Type)(0),       // 1: message.Response.Type
	(Response_Error)(0),      // 2: message.Response.Error
	(*Message)(nil),          // 3: message.Message
	(*Request)(nil),          // 4: message.Request
	(*Response)(nil),         // 5: message.Response
	(*Request_Metadata)(nil), // 6: message.Request.Metadata
}
var file_message_proto_depIdxs = []int32{
	4, // 0: message.Message.request:type_name -> message.Request
	5, // 1: message.Message.response:type_name -> message.Response
	0, // 2: message.Request.type:type_name -> message.Request.Type
	6, // 3: message.Request.metadata:type_name -> message.Request.Metadata
	1, // 4: message.Response.type:type_name -> message.Response.Type
	2, // 5: message.Response.error:type_name -> message.Response.Error
	6, // [6:6] is the sub-list for method output_type
	6, // [6:6] is the sub-list for method input_type
	6, // [6:6] is the sub-list for extension type_name
	6, // [6:6] is the sub-list for extension extendee
	0, // [0:6] is the sub-list for field type_name
}

func init() { file_message_proto_init() }
func file_message_proto_init() {
	if File_message_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_message_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Message); i {
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
		file_message_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Request); i {
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
		file_message_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Response); i {
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
		file_message_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Request_Metadata); i {
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
	file_message_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*Message_Request)(nil),
		(*Message_Response)(nil),
	}
	file_message_proto_msgTypes[1].OneofWrappers = []interface{}{}
	file_message_proto_msgTypes[2].OneofWrappers = []interface{}{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_message_proto_rawDesc,
			NumEnums:      3,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_message_proto_goTypes,
		DependencyIndexes: file_message_proto_depIdxs,
		EnumInfos:         file_message_proto_enumTypes,
		MessageInfos:      file_message_proto_msgTypes,
	}.Build()
	File_message_proto = out.File
	file_message_proto_rawDesc = nil
	file_message_proto_goTypes = nil
	file_message_proto_depIdxs = nil
}
