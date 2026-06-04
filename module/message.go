package module

import (
	"fmt"
	"strings"

	pgs "github.com/lyft/protoc-gen-star/v2"
	"github.com/oranpix/protoc-gen-go-flags/flags"
)

func (m *Module) checkMessage(typ pgs.FieldType, flag *flags.MessageFlag) {
	if !flag.GetNested() {
		return
	}
	m.mustType(typ, pgs.MessageT, pgs.UnknownWKT)
	if typ, ok := typ.(Repeatable); ok {
		m.Assert(!typ.IsRepeated(), "message flag does not support repeated fields")
	}
}

func (m *Module) genMessageDefaults(f pgs.Field, name pgs.Name, flag *flags.MessageFlag) string {
	var (
		declBuilder = &strings.Builder{}
	)
	if !flag.GetNested() {
		return fmt.Sprint("\n// ", name, ": flags disabled by [(flags.value).message = {nested: false}]")
	}
	if flag.GetNested() {
		_, _ = fmt.Fprintf(declBuilder, `
				if x.%s == nil {
					x.%s = new(%s)
				}
        	`,
			name, name, m.getFieldTypeName(f),
		)
	}
	_, _ = fmt.Fprintf(declBuilder, `
			if v, ok := interface{}(x.%s).(flags.Defaulter); ok {
				v.SetDefaults()
			}
        `,
		name,
	)
	return declBuilder.String()
}

func (m *Module) genMessage(f pgs.Field, name pgs.Name, flag *flags.MessageFlag) string {
	var (
		declBuilder = &strings.Builder{}
	)
	if !flag.GetNested() {
		return fmt.Sprint("\n// ", name, ": flags disabled by [(flags.value).message = {nested: false}]")
	}
	prefix := m.messagePrefix(f, flag)
	if flag.GetNested() {
		_, _ = fmt.Fprintf(declBuilder, `
				if x.%s == nil {
					x.%s = new(%s)
				}
        	`,
			name, name, m.getFieldTypeName(f),
		)
	}
	_, _ = fmt.Fprintf(declBuilder, `
			if v, ok := interface{}(x.%s).(flags.Flagger); ok {
				v.AddFlags(fs, append(opts, flags.WithPrefix(%q))...)
			}
        `,
		name, prefix,
	)
	return declBuilder.String()
}
