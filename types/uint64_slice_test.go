package types

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestUInt64SliceValue_Set(t *testing.T) {
	var slice []*wrapperspb.UInt64Value
	value := UInt64Slice(&slice)

	if err := value.Set("1, 18446744073709551615"); err != nil {
		t.Fatalf("UInt64SliceValue.Set() error = %v", err)
	}
	if got, want := len(slice), 2; got != want {
		t.Fatalf("len(slice) = %d, want %d", got, want)
	}
	if slice[0].Value != 1 || slice[1].Value != 18446744073709551615 {
		t.Fatalf("slice = [%d,%d], want [1,18446744073709551615]", slice[0].Value, slice[1].Value)
	}
	if got, want := value.String(), "[1,18446744073709551615]"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestUInt64SliceValue_SetOverflow(t *testing.T) {
	var slice []*wrapperspb.UInt64Value
	value := UInt64Slice(&slice)

	if err := value.Set("18446744073709551616"); err == nil {
		t.Fatal("UInt64SliceValue.Set() expected overflow error")
	}
}

func TestUint64SliceValue_SetAppend(t *testing.T) {
	var slice []uint64
	value := Uint64Slice(&slice)

	if err := value.Set("1,2"); err != nil {
		t.Fatalf("Uint64SliceValue.Set() error = %v", err)
	}
	if err := value.Set("3"); err != nil {
		t.Fatalf("Uint64SliceValue.Set() append error = %v", err)
	}
	if got, want := value.String(), "[1,2,3]"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}
