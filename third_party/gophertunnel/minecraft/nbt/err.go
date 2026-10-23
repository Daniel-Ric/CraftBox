package nbt

import (
	"errors"
	"fmt"
	"reflect"
)

type InvalidTypeError struct {
	Off       int64
	Field     string
	TagType   tagType
	FieldType reflect.Type
}

func (err InvalidTypeError) Error() string {
	return fmt.Sprintf("nbt: invalid type for tag '%v' at offset %v: cannot unmarshal %v into %v", err.Field, err.Off, err.TagType, err.FieldType)
}

type UnknownTagError struct {
	Off     int64
	Op      string
	TagType tagType
}

func (err UnknownTagError) Error() string {
	return fmt.Sprintf("nbt: unknown tag '%v' at offset %v during op '%v'", byte(err.TagType), err.Off, err.Op)
}

type UnexpectedTagError struct {
	Off     int64
	TagType tagType
}

func (err UnexpectedTagError) Error() string {
	return fmt.Sprintf("nbt: unexpected tag %v at offset %v: tag is not valid in its context", err.TagType, err.Off)
}

type NonPointerTypeError struct {
	ActualType reflect.Type
}

func (err NonPointerTypeError) Error() string {
	return fmt.Sprintf("nbt: expected ptr type to decode into, but got '%v'", err.ActualType)
}

type BufferOverrunError struct {
	Op string
}

func (err BufferOverrunError) Error() string {
	return fmt.Sprintf("nbt: unexpected buffer end during op: '%v'", err.Op)
}

type InvalidArraySizeError struct {
	Off       int64
	Op        string
	GoLength  int
	NBTLength int
}

func (err InvalidArraySizeError) Error() string {
	return fmt.Sprintf("nbt: mismatched array size at %v during op '%v': expected size %v, found %v in NBT", err.Off, err.Op, err.GoLength, err.NBTLength)
}

type UnexpectedNamedTagError struct {
	Off     int64
	TagName string
	TagType tagType
}

func (err UnexpectedNamedTagError) Error() string {
	return fmt.Sprintf("nbt: unexpected named tag '%v' with type %v at offset %v: not present in struct to be decoded into", err.TagName, err.TagType, err.Off)
}

type FailedWriteError struct {
	Off int64
	Op  string
	Err error
}

func (err FailedWriteError) Error() string {
	return fmt.Sprintf("nbt: failed write during op '%v' at offset %v: %v", err.Op, err.Off, err.Err)
}

type IncompatibleTypeError struct {
	ValueName string
	Type      reflect.Type
}

func (err IncompatibleTypeError) Error() string {
	return fmt.Sprintf("nbt: value type %v (%v) cannot be translated to an NBT tag", err.Type, err.ValueName)
}

var errStringTooLong = errors.New("string length exceeds maximum length")

type InvalidStringError struct {
	Off int64
	Err error
	N   uint
}

func (err InvalidStringError) Error() string {
	return fmt.Sprintf("nbt: string at offset %v is not valid: %v (len=%v)", err.Off, err.Err, err.N)
}

const maximumNestingDepth = 512

type MaximumDepthReachedError struct {
}

func (err MaximumDepthReachedError) Error() string {
	return fmt.Sprintf("nbt: maximum nesting depth of %v was reached", maximumNestingDepth)
}

const maximumNetworkOffset = 4 * 1024 * 1024

type MaximumBytesReadError struct {
}

func (err MaximumBytesReadError) Error() string {
	return fmt.Sprintf("nbt: limit of bytes read %v with NetworkLittleEndian format exhausted", maximumNetworkOffset)
}

type InvalidVarintError struct {
	Off int64
	N   int
}

func (err InvalidVarintError) Error() string {
	return fmt.Sprintf("nbt: varint did not terminate after %v bytes at offset %v", err.N, err.Off)
}
