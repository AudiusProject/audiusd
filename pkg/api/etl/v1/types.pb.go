// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.33.0
// 	protoc        (unknown)
// source: etl/v1/types.proto

package v1

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

type Order int32

const (
	Order_ORDER_UNSPECIFIED Order = 0
	Order_ORDER_ASC         Order = 1
	Order_ORDER_DESC        Order = 2
)

// Enum value maps for Order.
var (
	Order_name = map[int32]string{
		0: "ORDER_UNSPECIFIED",
		1: "ORDER_ASC",
		2: "ORDER_DESC",
	}
	Order_value = map[string]int32{
		"ORDER_UNSPECIFIED": 0,
		"ORDER_ASC":         1,
		"ORDER_DESC":        2,
	}
)

func (x Order) Enum() *Order {
	p := new(Order)
	*p = x
	return p
}

func (x Order) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Order) Descriptor() protoreflect.EnumDescriptor {
	return file_etl_v1_types_proto_enumTypes[0].Descriptor()
}

func (Order) Type() protoreflect.EnumType {
	return &file_etl_v1_types_proto_enumTypes[0]
}

func (x Order) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Order.Descriptor instead.
func (Order) EnumDescriptor() ([]byte, []int) {
	return file_etl_v1_types_proto_rawDescGZIP(), []int{0}
}

type PingRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *PingRequest) Reset() {
	*x = PingRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_etl_v1_types_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PingRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PingRequest) ProtoMessage() {}

func (x *PingRequest) ProtoReflect() protoreflect.Message {
	mi := &file_etl_v1_types_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PingRequest.ProtoReflect.Descriptor instead.
func (*PingRequest) Descriptor() ([]byte, []int) {
	return file_etl_v1_types_proto_rawDescGZIP(), []int{0}
}

type PingResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Message string `protobuf:"bytes,1,opt,name=message,proto3" json:"message,omitempty"`
}

func (x *PingResponse) Reset() {
	*x = PingResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_etl_v1_types_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PingResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PingResponse) ProtoMessage() {}

func (x *PingResponse) ProtoReflect() protoreflect.Message {
	mi := &file_etl_v1_types_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PingResponse.ProtoReflect.Descriptor instead.
func (*PingResponse) Descriptor() ([]byte, []int) {
	return file_etl_v1_types_proto_rawDescGZIP(), []int{1}
}

func (x *PingResponse) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

type GetHealthRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *GetHealthRequest) Reset() {
	*x = GetHealthRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_etl_v1_types_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetHealthRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetHealthRequest) ProtoMessage() {}

func (x *GetHealthRequest) ProtoReflect() protoreflect.Message {
	mi := &file_etl_v1_types_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetHealthRequest.ProtoReflect.Descriptor instead.
func (*GetHealthRequest) Descriptor() ([]byte, []int) {
	return file_etl_v1_types_proto_rawDescGZIP(), []int{2}
}

type GetHealthResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *GetHealthResponse) Reset() {
	*x = GetHealthResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_etl_v1_types_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetHealthResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetHealthResponse) ProtoMessage() {}

func (x *GetHealthResponse) ProtoReflect() protoreflect.Message {
	mi := &file_etl_v1_types_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetHealthResponse.ProtoReflect.Descriptor instead.
func (*GetHealthResponse) Descriptor() ([]byte, []int) {
	return file_etl_v1_types_proto_rawDescGZIP(), []int{3}
}

type GetPlaysRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Pagination *Pagination `protobuf:"bytes,1,opt,name=pagination,proto3" json:"pagination,omitempty"`
	Order      Order       `protobuf:"varint,2,opt,name=order,proto3,enum=etl.v1.Order" json:"order,omitempty"`
	UserId     string      `protobuf:"bytes,3,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	TrackId    string      `protobuf:"bytes,4,opt,name=track_id,json=trackId,proto3" json:"track_id,omitempty"`
}

func (x *GetPlaysRequest) Reset() {
	*x = GetPlaysRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_etl_v1_types_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetPlaysRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetPlaysRequest) ProtoMessage() {}

func (x *GetPlaysRequest) ProtoReflect() protoreflect.Message {
	mi := &file_etl_v1_types_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetPlaysRequest.ProtoReflect.Descriptor instead.
func (*GetPlaysRequest) Descriptor() ([]byte, []int) {
	return file_etl_v1_types_proto_rawDescGZIP(), []int{4}
}

func (x *GetPlaysRequest) GetPagination() *Pagination {
	if x != nil {
		return x.Pagination
	}
	return nil
}

func (x *GetPlaysRequest) GetOrder() Order {
	if x != nil {
		return x.Order
	}
	return Order_ORDER_UNSPECIFIED
}

func (x *GetPlaysRequest) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

func (x *GetPlaysRequest) GetTrackId() string {
	if x != nil {
		return x.TrackId
	}
	return ""
}

type GetPlaysResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Plays []*GetPlayResponse `protobuf:"bytes,1,rep,name=plays,proto3" json:"plays,omitempty"`
}

func (x *GetPlaysResponse) Reset() {
	*x = GetPlaysResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_etl_v1_types_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetPlaysResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetPlaysResponse) ProtoMessage() {}

func (x *GetPlaysResponse) ProtoReflect() protoreflect.Message {
	mi := &file_etl_v1_types_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetPlaysResponse.ProtoReflect.Descriptor instead.
func (*GetPlaysResponse) Descriptor() ([]byte, []int) {
	return file_etl_v1_types_proto_rawDescGZIP(), []int{5}
}

func (x *GetPlaysResponse) GetPlays() []*GetPlayResponse {
	if x != nil {
		return x.Plays
	}
	return nil
}

type GetPlayResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	UserId      string `protobuf:"bytes,1,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	TrackId     string `protobuf:"bytes,2,opt,name=track_id,json=trackId,proto3" json:"track_id,omitempty"`
	Timestamp   int64  `protobuf:"varint,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	City        string `protobuf:"bytes,4,opt,name=city,proto3" json:"city,omitempty"`
	Country     string `protobuf:"bytes,5,opt,name=country,proto3" json:"country,omitempty"`
	Region      string `protobuf:"bytes,6,opt,name=region,proto3" json:"region,omitempty"`
	BlockHeight int64  `protobuf:"varint,7,opt,name=block_height,json=blockHeight,proto3" json:"block_height,omitempty"`
	TxHash      string `protobuf:"bytes,8,opt,name=tx_hash,json=txHash,proto3" json:"tx_hash,omitempty"`
}

func (x *GetPlayResponse) Reset() {
	*x = GetPlayResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_etl_v1_types_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetPlayResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetPlayResponse) ProtoMessage() {}

func (x *GetPlayResponse) ProtoReflect() protoreflect.Message {
	mi := &file_etl_v1_types_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetPlayResponse.ProtoReflect.Descriptor instead.
func (*GetPlayResponse) Descriptor() ([]byte, []int) {
	return file_etl_v1_types_proto_rawDescGZIP(), []int{6}
}

func (x *GetPlayResponse) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

func (x *GetPlayResponse) GetTrackId() string {
	if x != nil {
		return x.TrackId
	}
	return ""
}

func (x *GetPlayResponse) GetTimestamp() int64 {
	if x != nil {
		return x.Timestamp
	}
	return 0
}

func (x *GetPlayResponse) GetCity() string {
	if x != nil {
		return x.City
	}
	return ""
}

func (x *GetPlayResponse) GetCountry() string {
	if x != nil {
		return x.Country
	}
	return ""
}

func (x *GetPlayResponse) GetRegion() string {
	if x != nil {
		return x.Region
	}
	return ""
}

func (x *GetPlayResponse) GetBlockHeight() int64 {
	if x != nil {
		return x.BlockHeight
	}
	return 0
}

func (x *GetPlayResponse) GetTxHash() string {
	if x != nil {
		return x.TxHash
	}
	return ""
}

type Pagination struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Limit  int64 `protobuf:"varint,1,opt,name=limit,proto3" json:"limit,omitempty"`
	Offset int64 `protobuf:"varint,2,opt,name=offset,proto3" json:"offset,omitempty"`
}

func (x *Pagination) Reset() {
	*x = Pagination{}
	if protoimpl.UnsafeEnabled {
		mi := &file_etl_v1_types_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Pagination) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Pagination) ProtoMessage() {}

func (x *Pagination) ProtoReflect() protoreflect.Message {
	mi := &file_etl_v1_types_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Pagination.ProtoReflect.Descriptor instead.
func (*Pagination) Descriptor() ([]byte, []int) {
	return file_etl_v1_types_proto_rawDescGZIP(), []int{7}
}

func (x *Pagination) GetLimit() int64 {
	if x != nil {
		return x.Limit
	}
	return 0
}

func (x *Pagination) GetOffset() int64 {
	if x != nil {
		return x.Offset
	}
	return 0
}

var File_etl_v1_types_proto protoreflect.FileDescriptor

var file_etl_v1_types_proto_rawDesc = []byte{
	0x0a, 0x12, 0x65, 0x74, 0x6c, 0x2f, 0x76, 0x31, 0x2f, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x06, 0x65, 0x74, 0x6c, 0x2e, 0x76, 0x31, 0x22, 0x0d, 0x0a, 0x0b,
	0x50, 0x69, 0x6e, 0x67, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x22, 0x28, 0x0a, 0x0c, 0x50,
	0x69, 0x6e, 0x67, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x6d,
	0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6d, 0x65,
	0x73, 0x73, 0x61, 0x67, 0x65, 0x22, 0x12, 0x0a, 0x10, 0x47, 0x65, 0x74, 0x48, 0x65, 0x61, 0x6c,
	0x74, 0x68, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x22, 0x13, 0x0a, 0x11, 0x47, 0x65, 0x74,
	0x48, 0x65, 0x61, 0x6c, 0x74, 0x68, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x9e,
	0x01, 0x0a, 0x0f, 0x47, 0x65, 0x74, 0x50, 0x6c, 0x61, 0x79, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x12, 0x32, 0x0a, 0x0a, 0x70, 0x61, 0x67, 0x69, 0x6e, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x65, 0x74, 0x6c, 0x2e, 0x76, 0x31, 0x2e,
	0x50, 0x61, 0x67, 0x69, 0x6e, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0a, 0x70, 0x61, 0x67, 0x69,
	0x6e, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x23, 0x0a, 0x05, 0x6f, 0x72, 0x64, 0x65, 0x72, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x0d, 0x2e, 0x65, 0x74, 0x6c, 0x2e, 0x76, 0x31, 0x2e, 0x4f,
	0x72, 0x64, 0x65, 0x72, 0x52, 0x05, 0x6f, 0x72, 0x64, 0x65, 0x72, 0x12, 0x17, 0x0a, 0x07, 0x75,
	0x73, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x75, 0x73,
	0x65, 0x72, 0x49, 0x64, 0x12, 0x19, 0x0a, 0x08, 0x74, 0x72, 0x61, 0x63, 0x6b, 0x5f, 0x69, 0x64,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x74, 0x72, 0x61, 0x63, 0x6b, 0x49, 0x64, 0x22,
	0x41, 0x0a, 0x10, 0x47, 0x65, 0x74, 0x50, 0x6c, 0x61, 0x79, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x12, 0x2d, 0x0a, 0x05, 0x70, 0x6c, 0x61, 0x79, 0x73, 0x18, 0x01, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x17, 0x2e, 0x65, 0x74, 0x6c, 0x2e, 0x76, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x50,
	0x6c, 0x61, 0x79, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x52, 0x05, 0x70, 0x6c, 0x61,
	0x79, 0x73, 0x22, 0xe5, 0x01, 0x0a, 0x0f, 0x47, 0x65, 0x74, 0x50, 0x6c, 0x61, 0x79, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x17, 0x0a, 0x07, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x69,
	0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x75, 0x73, 0x65, 0x72, 0x49, 0x64, 0x12,
	0x19, 0x0a, 0x08, 0x74, 0x72, 0x61, 0x63, 0x6b, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x07, 0x74, 0x72, 0x61, 0x63, 0x6b, 0x49, 0x64, 0x12, 0x1c, 0x0a, 0x09, 0x74, 0x69,
	0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x52, 0x09, 0x74,
	0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x12, 0x12, 0x0a, 0x04, 0x63, 0x69, 0x74, 0x79,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x63, 0x69, 0x74, 0x79, 0x12, 0x18, 0x0a, 0x07,
	0x63, 0x6f, 0x75, 0x6e, 0x74, 0x72, 0x79, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x63,
	0x6f, 0x75, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x16, 0x0a, 0x06, 0x72, 0x65, 0x67, 0x69, 0x6f, 0x6e,
	0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x72, 0x65, 0x67, 0x69, 0x6f, 0x6e, 0x12, 0x21,
	0x0a, 0x0c, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x5f, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x18, 0x07,
	0x20, 0x01, 0x28, 0x03, 0x52, 0x0b, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x48, 0x65, 0x69, 0x67, 0x68,
	0x74, 0x12, 0x17, 0x0a, 0x07, 0x74, 0x78, 0x5f, 0x68, 0x61, 0x73, 0x68, 0x18, 0x08, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x06, 0x74, 0x78, 0x48, 0x61, 0x73, 0x68, 0x22, 0x3a, 0x0a, 0x0a, 0x50, 0x61,
	0x67, 0x69, 0x6e, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x14, 0x0a, 0x05, 0x6c, 0x69, 0x6d, 0x69,
	0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x05, 0x6c, 0x69, 0x6d, 0x69, 0x74, 0x12, 0x16,
	0x0a, 0x06, 0x6f, 0x66, 0x66, 0x73, 0x65, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06,
	0x6f, 0x66, 0x66, 0x73, 0x65, 0x74, 0x2a, 0x3d, 0x0a, 0x05, 0x4f, 0x72, 0x64, 0x65, 0x72, 0x12,
	0x15, 0x0a, 0x11, 0x4f, 0x52, 0x44, 0x45, 0x52, 0x5f, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43, 0x49,
	0x46, 0x49, 0x45, 0x44, 0x10, 0x00, 0x12, 0x0d, 0x0a, 0x09, 0x4f, 0x52, 0x44, 0x45, 0x52, 0x5f,
	0x41, 0x53, 0x43, 0x10, 0x01, 0x12, 0x0e, 0x0a, 0x0a, 0x4f, 0x52, 0x44, 0x45, 0x52, 0x5f, 0x44,
	0x45, 0x53, 0x43, 0x10, 0x02, 0x42, 0x31, 0x5a, 0x2f, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x41, 0x75, 0x64, 0x69, 0x75, 0x73, 0x50, 0x72, 0x6f, 0x6a, 0x65, 0x63,
	0x74, 0x2f, 0x61, 0x75, 0x64, 0x69, 0x75, 0x73, 0x64, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x61, 0x70,
	0x69, 0x2f, 0x65, 0x74, 0x6c, 0x2f, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_etl_v1_types_proto_rawDescOnce sync.Once
	file_etl_v1_types_proto_rawDescData = file_etl_v1_types_proto_rawDesc
)

func file_etl_v1_types_proto_rawDescGZIP() []byte {
	file_etl_v1_types_proto_rawDescOnce.Do(func() {
		file_etl_v1_types_proto_rawDescData = protoimpl.X.CompressGZIP(file_etl_v1_types_proto_rawDescData)
	})
	return file_etl_v1_types_proto_rawDescData
}

var file_etl_v1_types_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_etl_v1_types_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_etl_v1_types_proto_goTypes = []interface{}{
	(Order)(0),                // 0: etl.v1.Order
	(*PingRequest)(nil),       // 1: etl.v1.PingRequest
	(*PingResponse)(nil),      // 2: etl.v1.PingResponse
	(*GetHealthRequest)(nil),  // 3: etl.v1.GetHealthRequest
	(*GetHealthResponse)(nil), // 4: etl.v1.GetHealthResponse
	(*GetPlaysRequest)(nil),   // 5: etl.v1.GetPlaysRequest
	(*GetPlaysResponse)(nil),  // 6: etl.v1.GetPlaysResponse
	(*GetPlayResponse)(nil),   // 7: etl.v1.GetPlayResponse
	(*Pagination)(nil),        // 8: etl.v1.Pagination
}
var file_etl_v1_types_proto_depIdxs = []int32{
	8, // 0: etl.v1.GetPlaysRequest.pagination:type_name -> etl.v1.Pagination
	0, // 1: etl.v1.GetPlaysRequest.order:type_name -> etl.v1.Order
	7, // 2: etl.v1.GetPlaysResponse.plays:type_name -> etl.v1.GetPlayResponse
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_etl_v1_types_proto_init() }
func file_etl_v1_types_proto_init() {
	if File_etl_v1_types_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_etl_v1_types_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PingRequest); i {
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
		file_etl_v1_types_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PingResponse); i {
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
		file_etl_v1_types_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetHealthRequest); i {
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
		file_etl_v1_types_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetHealthResponse); i {
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
		file_etl_v1_types_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetPlaysRequest); i {
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
		file_etl_v1_types_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetPlaysResponse); i {
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
		file_etl_v1_types_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetPlayResponse); i {
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
		file_etl_v1_types_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Pagination); i {
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
			RawDescriptor: file_etl_v1_types_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   8,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_etl_v1_types_proto_goTypes,
		DependencyIndexes: file_etl_v1_types_proto_depIdxs,
		EnumInfos:         file_etl_v1_types_proto_enumTypes,
		MessageInfos:      file_etl_v1_types_proto_msgTypes,
	}.Build()
	File_etl_v1_types_proto = out.File
	file_etl_v1_types_proto_rawDesc = nil
	file_etl_v1_types_proto_goTypes = nil
	file_etl_v1_types_proto_depIdxs = nil
}
