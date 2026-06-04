package types

import (
	"strconv"
	"strings"

	"github.com/oranpix/protoc-gen-go-flags/utils"
	"github.com/spf13/pflag"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var _ pflag.Value = (*UInt32SliceValue)(nil)
var _ pflag.Value = (*Uint32SliceValue)(nil)

type UInt32SliceValue struct {
	value   *[]*wrapperspb.UInt32Value
	changed bool
}

func (s *UInt32SliceValue) Set(val string) error {
	ss := strings.Split(val, ",")
	out := make([]*wrapperspb.UInt32Value, len(ss))
	for i, d := range ss {
		temp, err := strconv.ParseUint(strings.TrimSpace(d), 10, 32)
		if err != nil {
			return err
		}
		out[i] = wrapperspb.UInt32(uint32(temp))
	}
	if !s.changed {
		*s.value = out
	} else {
		*s.value = append(*s.value, out...)
	}
	s.changed = true
	return nil
}

func (s *UInt32SliceValue) Type() string {
	return "uint32SliceValue"
}

func (s *UInt32SliceValue) String() string {
	uint32StrSlice := make([]string, len(*s.value))
	for i, v := range *s.value {
		uint32StrSlice[i] = strconv.FormatUint(uint64(v.Value), 10)
	}
	out, _ := utils.WriteAsCSV(uint32StrSlice)
	return "[" + out + "]"
}

func UInt32Slice(v *[]*wrapperspb.UInt32Value) *UInt32SliceValue {
	return &UInt32SliceValue{value: v}
}

type Uint32SliceValue struct {
	value   *[]uint32
	changed bool
}

func (s *Uint32SliceValue) Set(val string) error {
	ss := strings.Split(val, ",")
	out := make([]uint32, len(ss))
	for i, d := range ss {
		temp, err := strconv.ParseUint(strings.TrimSpace(d), 10, 32)
		if err != nil {
			return err
		}
		out[i] = uint32(temp)
	}
	if !s.changed {
		*s.value = out
	} else {
		*s.value = append(*s.value, out...)
	}
	s.changed = true
	return nil
}

func (s *Uint32SliceValue) Type() string {
	return "uint32SliceValue"
}

func (s *Uint32SliceValue) String() string {
	uint32StrSlice := make([]string, len(*s.value))
	for i, v := range *s.value {
		uint32StrSlice[i] = strconv.FormatUint(uint64(v), 10)
	}
	out, _ := utils.WriteAsCSV(uint32StrSlice)
	return "[" + out + "]"
}

func Uint32Slice(v *[]uint32) *Uint32SliceValue {
	return &Uint32SliceValue{value: v}
}
