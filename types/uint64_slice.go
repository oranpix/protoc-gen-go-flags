package types

import (
	"strconv"
	"strings"

	"github.com/oranpix/protoc-gen-go-flags/utils"
	"github.com/spf13/pflag"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var _ pflag.Value = (*UInt64SliceValue)(nil)
var _ pflag.Value = (*Uint64SliceValue)(nil)

type UInt64SliceValue struct {
	value   *[]*wrapperspb.UInt64Value
	changed bool
}

func (s *UInt64SliceValue) Set(val string) error {
	ss := strings.Split(val, ",")
	out := make([]*wrapperspb.UInt64Value, len(ss))
	for i, d := range ss {
		temp, err := strconv.ParseUint(strings.TrimSpace(d), 10, 64)
		if err != nil {
			return err
		}
		out[i] = wrapperspb.UInt64(temp)
	}
	if !s.changed {
		*s.value = out
	} else {
		*s.value = append(*s.value, out...)
	}
	s.changed = true
	return nil
}

func (s *UInt64SliceValue) Type() string {
	return "uint64SliceValue"
}

func (s *UInt64SliceValue) String() string {
	uint64StrSlice := make([]string, len(*s.value))
	for i, v := range *s.value {
		uint64StrSlice[i] = strconv.FormatUint(v.Value, 10)
	}
	out, _ := utils.WriteAsCSV(uint64StrSlice)
	return "[" + out + "]"
}

func UInt64Slice(v *[]*wrapperspb.UInt64Value) *UInt64SliceValue {
	return &UInt64SliceValue{value: v}
}

type Uint64SliceValue struct {
	value   *[]uint64
	changed bool
}

func (s *Uint64SliceValue) Set(val string) error {
	ss := strings.Split(val, ",")
	out := make([]uint64, len(ss))
	for i, d := range ss {
		temp, err := strconv.ParseUint(strings.TrimSpace(d), 10, 64)
		if err != nil {
			return err
		}
		out[i] = temp
	}
	if !s.changed {
		*s.value = out
	} else {
		*s.value = append(*s.value, out...)
	}
	s.changed = true
	return nil
}

func (s *Uint64SliceValue) Type() string {
	return "uint64SliceValue"
}

func (s *Uint64SliceValue) String() string {
	uint64StrSlice := make([]string, len(*s.value))
	for i, v := range *s.value {
		uint64StrSlice[i] = strconv.FormatUint(v, 10)
	}
	out, _ := utils.WriteAsCSV(uint64StrSlice)
	return "[" + out + "]"
}

func Uint64Slice(v *[]uint64) *Uint64SliceValue {
	return &Uint64SliceValue{value: v}
}
