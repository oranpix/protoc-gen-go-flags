package types

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestUInt32SliceValue_Set(t *testing.T) {
	var slice []*wrapperspb.UInt32Value
	value := UInt32Slice(&slice)

	if err := value.Set("1, 4294967295"); err != nil {
		t.Fatalf("UInt32SliceValue.Set() error = %v", err)
	}
	if got, want := len(slice), 2; got != want {
		t.Fatalf("len(slice) = %d, want %d", got, want)
	}
	if slice[0].Value != 1 || slice[1].Value != 4294967295 {
		t.Fatalf("slice = [%d,%d], want [1,4294967295]", slice[0].Value, slice[1].Value)
	}
	if got, want := value.String(), "[1,4294967295]"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestUInt32SliceValue_SetOverflow(t *testing.T) {
	var slice []*wrapperspb.UInt32Value
	value := UInt32Slice(&slice)

	if err := value.Set("4294967296"); err == nil {
		t.Fatal("UInt32SliceValue.Set() expected overflow error")
	}
}

func TestUint32SliceValue_SetAppend(t *testing.T) {
	var slice []uint32
	value := Uint32Slice(&slice)

	if err := value.Set("1,2"); err != nil {
		t.Fatalf("Uint32SliceValue.Set() error = %v", err)
	}
	if err := value.Set("3"); err != nil {
		t.Fatalf("Uint32SliceValue.Set() append error = %v", err)
	}
	if got, want := value.String(), "[1,2,3]"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}
