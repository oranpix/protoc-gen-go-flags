package tests

import (
	"testing"

	"github.com/spf13/pflag"
)

func TestGeneratedAddFlagsRegisterWithoutConflicts(t *testing.T) {
	tests := []struct {
		name string
		add  func(*pflag.FlagSet)
	}{
		{
			name: "test for message",
			add: func(fs *pflag.FlagSet) {
				var msg TestForMessage
				msg.AddFlags(fs)
			},
		},
		{
			name: "wrapper value message",
			add: func(fs *pflag.FlagSet) {
				var msg WrapperValueMessage
				msg.AddFlags(fs)
			},
		},
		{
			name: "duration slice message",
			add: func(fs *pflag.FlagSet) {
				var msg DurationSliceTestMessage
				msg.AddFlags(fs)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet(tt.name, pflag.ContinueOnError)
			tt.add(fs)
		})
	}
}

func TestNestedMessageDefaultPrefixUsesProtoFieldName(t *testing.T) {
	fs := pflag.NewFlagSet("nested message prefix", pflag.ContinueOnError)

	var msg NestedMessageTestMessage
	msg.AddFlags(fs)

	if flag := fs.Lookup("client_config.name"); flag == nil {
		t.Fatalf("expected nested message flag with proto field prefix client_config.name")
	}
	if flag := fs.Lookup("clientconfig.name"); flag != nil {
		t.Fatalf("did not expect nested message flag with Go field prefix clientconfig.name")
	}
}
