package module

import (
	pgs "github.com/lyft/protoc-gen-star/v2"
	"github.com/oranpix/protoc-gen-go-flags/flags"
	"google.golang.org/protobuf/runtime/protoimpl"
)

// Heavily taken from https://github.com/envoyproxy/protoc-gen-validate/blob/main/module/checker.go

type FieldType interface {
	ProtoType() pgs.ProtoType
	Embed() pgs.Message
}

type Repeatable interface {
	IsRepeated() bool
}

type Element interface {
	Element() pgs.FieldTypeElem
}

type fieldFlagInfo struct {
	name       string
	short      string
	disabled   bool
	message    *flags.MessageFlag
	hasMessage bool
}

// shouldGenerate checks if a proto file contains any flag-related options
// and determines whether a .flags.go file should be generated for it.
// It returns true if any message in the file has flag configurations.
func (m *Module) shouldGenerate(f pgs.File) bool {
	if len(f.Messages()) == 0 {
		return false
	}

	// Check each message in the file for flag configurations
	for _, msg := range f.Messages() {
		// Check message-level options: disabled, unexported, allow_empty
		if m.hasMessageLevelOptions(msg) {
			return true
		}

		// Check if message has any field-level flag configurations
		if m.hasFieldLevelOptions(msg) {
			return true
		}
	}

	return false
}

// hasMessageLevelOptions checks if a message has any message-level flag options.
// Returns true if disabled, unexported, or allow_empty options are present.
func (m *Module) hasMessageLevelOptions(msg pgs.Message) bool {
	extensions := []*protoimpl.ExtensionInfo{
		flags.E_Disabled,
		flags.E_Unexported,
		flags.E_AllowEmpty,
	}

	for _, ext := range extensions {
		var val bool
		ok, err := msg.Extension(ext, &val)
		if err != nil {
			m.CheckErr(err, "unable to read flags extension from message")
		}
		if ok {
			return true
		}
	}

	return false
}

// hasFieldLevelOptions checks if a message has any field-level flag configurations.
// Returns true if any field has flags.E_Value extension.
func (m *Module) hasFieldLevelOptions(msg pgs.Message) bool {
	for _, field := range msg.Fields() {
		var fieldFlags flags.FieldFlags
		ok, err := field.Extension(flags.E_Value, &fieldFlags)
		if err != nil {
			m.CheckErr(err, "unable to read flags extension from field")
		}
		if ok {
			return true
		}
	}
	return false
}

func (m *Module) Check(msg pgs.Message) {
	m.Push("msg: " + msg.Name().String())
	defer m.Pop()

	var disabled bool
	_, err := msg.Extension(flags.E_Disabled, &disabled)
	m.CheckErr(err, "unable to read flags extension from message")

	if disabled {
		m.Debug("flags disabled, skipping checks")
		return
	}

	// Check for duplicate flag names within this message
	m.checkFlagName(msg)
	m.checkNestedFlagRegistration(msg)

	for _, f := range msg.Fields() {
		m.Push(f.Name().String())

		var field flags.FieldFlags
		_, err := f.Extension(flags.E_Value, &field)

		m.CheckErr(err, "unable to read flags from field")
		m.CheckFieldRules(f, &field)
		m.Pop()
	}
}

func (m *Module) checkFlagName(msg pgs.Message) {
	flagNames := make(map[string]string)
	shortNames := make(map[string]string)

	for _, f := range msg.Fields() {
		info, ok := m.getFieldFlagInfo(f)
		if !ok || info.disabled {
			continue
		}

		if info.name != "" {
			if existingField, exists := flagNames[info.name]; exists {
				m.Failf("duplicate flag name '%s' found in message '%s': field '%s' and field '%s' both use this flag name",
					info.name, msg.Name().String(), existingField, f.Name().String())
			}
			flagNames[info.name] = f.Name().String()
		}

		if info.short == "" {
			continue
		}
		if len(info.short) != 1 || info.short[0] > 127 {
			m.Failf("short flag '%s' found in message '%s' field '%s' must be exactly one ASCII character",
				info.short, msg.Name().String(), f.Name().String())
		}
		if existingField, exists := shortNames[info.short]; exists {
			m.Failf("duplicate short flag '%s' found in message '%s': field '%s' and field '%s' both use this short flag",
				info.short, msg.Name().String(), existingField, f.Name().String())
		}
		shortNames[info.short] = f.Name().String()
	}
}

func (m *Module) checkNestedFlagRegistration(msg pgs.Message) {
	longNames := make(map[string]string)
	shortNames := make(map[string]string)
	m.collectRegisteredFlags(msg, "", msg.Name().String(), longNames, shortNames)
}

func (m *Module) collectRegisteredFlags(msg pgs.Message, prefix, path string, longNames, shortNames map[string]string) {
	for _, f := range msg.Fields() {
		if f.InRealOneOf() {
			continue
		}

		info, ok := m.getFieldFlagInfo(f)
		if !ok || info.disabled {
			continue
		}

		fieldPath := path + "." + f.Name().String()
		if info.hasMessage {
			if info.message == nil || !info.message.GetNested() {
				continue
			}
			if !f.Type().IsEmbed() || f.Type().Embed() == nil {
				continue
			}
			if !m.messageGeneratesPublicAddFlags(f.Type().Embed()) {
				continue
			}
			nextPrefix := joinFlagPath(prefix, m.messagePrefix(f, info.message))
			m.collectRegisteredFlags(f.Type().Embed(), nextPrefix, fieldPath, longNames, shortNames)
			continue
		}

		if info.name != "" {
			fullName := joinFlagPath(prefix, info.name)
			if existingField, exists := longNames[fullName]; exists {
				m.Failf("duplicate flag name '%s' found in AddFlags tree rooted at message '%s': field '%s' and field '%s' both register this flag name",
					fullName, msg.Name().String(), existingField, fieldPath)
			}
			longNames[fullName] = fieldPath
		}

		if info.short == "" {
			continue
		}
		if existingField, exists := shortNames[info.short]; exists {
			m.Failf("duplicate short flag '%s' found in AddFlags tree rooted at message '%s': field '%s' and field '%s' both register this short flag",
				info.short, msg.Name().String(), existingField, fieldPath)
		}
		shortNames[info.short] = fieldPath
	}
}

func joinFlagPath(prefix, name string) string {
	if prefix == "" {
		return name
	}
	if name == "" {
		return prefix
	}
	return prefix + "." + name
}

func (m *Module) getFieldFlagInfo(f pgs.Field) (fieldFlagInfo, bool) {
	var field flags.FieldFlags
	ok, err := f.Extension(flags.E_Value, &field)
	if err != nil || !ok {
		return fieldFlagInfo{}, false
	}

	switch r := field.Type.(type) {
	case *flags.FieldFlags_Float:
		return m.infoFromCommonFlag(f, r.Float), true
	case *flags.FieldFlags_Double:
		return m.infoFromCommonFlag(f, r.Double), true
	case *flags.FieldFlags_Int32:
		return m.infoFromCommonFlag(f, r.Int32), true
	case *flags.FieldFlags_Int64:
		return m.infoFromCommonFlag(f, r.Int64), true
	case *flags.FieldFlags_Uint32:
		return m.infoFromCommonFlag(f, r.Uint32), true
	case *flags.FieldFlags_Uint64:
		return m.infoFromCommonFlag(f, r.Uint64), true
	case *flags.FieldFlags_Sint32:
		return m.infoFromCommonFlag(f, r.Sint32), true
	case *flags.FieldFlags_Sint64:
		return m.infoFromCommonFlag(f, r.Sint64), true
	case *flags.FieldFlags_Fixed32:
		return m.infoFromCommonFlag(f, r.Fixed32), true
	case *flags.FieldFlags_Fixed64:
		return m.infoFromCommonFlag(f, r.Fixed64), true
	case *flags.FieldFlags_Sfixed32:
		return m.infoFromCommonFlag(f, r.Sfixed32), true
	case *flags.FieldFlags_Sfixed64:
		return m.infoFromCommonFlag(f, r.Sfixed64), true
	case *flags.FieldFlags_Bool:
		return m.infoFromCommonFlag(f, r.Bool), true
	case *flags.FieldFlags_String_:
		return m.infoFromCommonFlag(f, r.String_), true
	case *flags.FieldFlags_Bytes:
		return m.infoFromCommonFlag(f, r.Bytes), true
	case *flags.FieldFlags_Enum:
		return m.infoFromCommonFlag(f, r.Enum), true
	case *flags.FieldFlags_Duration:
		return m.infoFromCommonFlag(f, r.Duration), true
	case *flags.FieldFlags_Timestamp:
		return m.infoFromCommonFlag(f, r.Timestamp), true
	case *flags.FieldFlags_Repeated:
		return m.infoFromRepeatedFlag(f, r.Repeated), true
	case *flags.FieldFlags_Map:
		return m.infoFromCommonFlag(f, r.Map), true
	case *flags.FieldFlags_Message:
		return fieldFlagInfo{message: r.Message, hasMessage: true}, true
	default:
		return fieldFlagInfo{}, false
	}
}

func (m *Module) infoFromCommonFlag(f pgs.Field, flag commonFlag) fieldFlagInfo {
	if flag == nil || flag.GetDisabled() {
		return fieldFlagInfo{disabled: true}
	}
	return fieldFlagInfo{
		name:  m.fieldFlagName(f, flag),
		short: flag.GetShort(),
	}
}

func (m *Module) infoFromRepeatedFlag(f pgs.Field, flag *flags.RepeatedFlags) fieldFlagInfo {
	if flag == nil {
		return fieldFlagInfo{disabled: true}
	}

	switch r := flag.Type.(type) {
	case *flags.RepeatedFlags_Float:
		return m.infoFromCommonFlag(f, r.Float)
	case *flags.RepeatedFlags_Double:
		return m.infoFromCommonFlag(f, r.Double)
	case *flags.RepeatedFlags_Int32:
		return m.infoFromCommonFlag(f, r.Int32)
	case *flags.RepeatedFlags_Int64:
		return m.infoFromCommonFlag(f, r.Int64)
	case *flags.RepeatedFlags_Uint32:
		return m.infoFromCommonFlag(f, r.Uint32)
	case *flags.RepeatedFlags_Uint64:
		return m.infoFromCommonFlag(f, r.Uint64)
	case *flags.RepeatedFlags_Sint32:
		return m.infoFromCommonFlag(f, r.Sint32)
	case *flags.RepeatedFlags_Sint64:
		return m.infoFromCommonFlag(f, r.Sint64)
	case *flags.RepeatedFlags_Fixed32:
		return m.infoFromCommonFlag(f, r.Fixed32)
	case *flags.RepeatedFlags_Fixed64:
		return m.infoFromCommonFlag(f, r.Fixed64)
	case *flags.RepeatedFlags_Sfixed32:
		return m.infoFromCommonFlag(f, r.Sfixed32)
	case *flags.RepeatedFlags_Sfixed64:
		return m.infoFromCommonFlag(f, r.Sfixed64)
	case *flags.RepeatedFlags_Bool:
		return m.infoFromCommonFlag(f, r.Bool)
	case *flags.RepeatedFlags_String_:
		return m.infoFromCommonFlag(f, r.String_)
	case *flags.RepeatedFlags_Bytes:
		return m.infoFromCommonFlag(f, r.Bytes)
	case *flags.RepeatedFlags_Enum:
		return m.infoFromCommonFlag(f, r.Enum)
	case *flags.RepeatedFlags_Duration:
		return m.infoFromCommonFlag(f, r.Duration)
	case *flags.RepeatedFlags_Timestamp:
		return m.infoFromCommonFlag(f, r.Timestamp)
	default:
		return fieldFlagInfo{}
	}
}

func (m *Module) CheckFieldRules(f pgs.Field, field *flags.FieldFlags) {
	if field == nil {
		return
	}
	typ := f.Type()

	switch r := field.Type.(type) {
	case *flags.FieldFlags_Float:
		m.checkCommon(typ, r.Float, pgs.FloatT, pgs.FloatValueWKT, false)
	case *flags.FieldFlags_Double:
		m.checkCommon(typ, r.Double, pgs.DoubleT, pgs.DoubleValueWKT, false)
	case *flags.FieldFlags_Int32:
		m.checkCommon(typ, r.Int32, pgs.Int32T, pgs.Int32ValueWKT, false)
	case *flags.FieldFlags_Int64:
		m.checkCommon(typ, r.Int64, pgs.Int64T, pgs.Int64ValueWKT, false)
	case *flags.FieldFlags_Uint32:
		m.checkCommon(typ, r.Uint32, pgs.UInt32T, pgs.UInt32ValueWKT, false)
	case *flags.FieldFlags_Uint64:
		m.checkCommon(typ, r.Uint64, pgs.UInt64T, pgs.UInt64ValueWKT, false)
	case *flags.FieldFlags_Sint32:
		m.checkCommon(typ, r.Sint32, pgs.SInt32, pgs.UnknownWKT, false)
	case *flags.FieldFlags_Sint64:
		m.checkCommon(typ, r.Sint64, pgs.SInt64, pgs.UnknownWKT, false)
	case *flags.FieldFlags_Fixed32:
		m.checkCommon(typ, r.Fixed32, pgs.Fixed32T, pgs.UnknownWKT, false)
	case *flags.FieldFlags_Fixed64:
		m.checkCommon(typ, r.Fixed64, pgs.Fixed64T, pgs.UnknownWKT, false)
	case *flags.FieldFlags_Sfixed32:
		m.checkCommon(typ, r.Sfixed32, pgs.SFixed32, pgs.UnknownWKT, false)
	case *flags.FieldFlags_Sfixed64:
		m.checkCommon(typ, r.Sfixed64, pgs.SFixed64, pgs.UnknownWKT, false)
	case *flags.FieldFlags_Bool:
		m.checkCommon(typ, r.Bool, pgs.BoolT, pgs.BoolValueWKT, false)
	case *flags.FieldFlags_String_:
		m.checkCommon(typ, r.String_, pgs.StringT, pgs.StringValueWKT, false)
	case *flags.FieldFlags_Bytes:
		m.checkBytes(typ, r.Bytes)
	case *flags.FieldFlags_Enum:
		m.checkEnum(typ, r.Enum, pgs.EnumT, pgs.UnknownWKT)
	case *flags.FieldFlags_Duration:
		m.checkCommon(typ, r.Duration, pgs.MessageT, pgs.DurationWKT, false)
	case *flags.FieldFlags_Timestamp:
		m.checkTimestamp(typ, r.Timestamp)
	case *flags.FieldFlags_Repeated:
		el, ok := typ.(Element)
		if !ok || el.Element() == nil {
			m.Failf("field '%s' is not a repeated field (actual type: %v), "+
				"but repeated flag configuration was specified", f.Name(), typ.ProtoType())
			return
		}
		m.CheckRepeatedFlag(el.Element(), r.Repeated)
	case *flags.FieldFlags_Message:
		current := m.ctx.ImportPath(f).String()
		if i := m.ctx.ImportPath(typ.Embed()).String(); i != current {
			m.imports[i] = struct{}{}
		}
		m.checkMessage(typ, r.Message)
	case *flags.FieldFlags_Map:
		m.checkMap(typ, r.Map)
	case nil: // noop
	default:
		m.Failf("unknown rule type (%T)", field.Type)
	}
}

func (m *Module) mustType(typ FieldType, pt pgs.ProtoType, wrapper pgs.WellKnownType) {
	if emb := typ.Embed(); emb != nil && emb.IsWellKnown() && emb.WellKnownType() == wrapper {
		if wrapper != pgs.DurationWKT && wrapper != pgs.TimestampWKT {
			m.mustType(emb.Fields()[0].Type(), pt, pgs.UnknownWKT)
			return
		}
	}

	m.Assert(typ.ProtoType() == pt,
		" expected flags for ",
		typ.ProtoType().Proto(),
		" but got ",
		pt.Proto(),
	)
}

type commonFlag interface {
	GetDisabled() bool
	GetName() string
	GetUsage() string
	GetDeprecated() bool
	GetDeprecatedUsage() string
	GetHidden() bool
	GetShort() string
}

func (m *Module) CheckRepeatedFlag(typ FieldType, repeated *flags.RepeatedFlags) {
	if repeated == nil {
		return
	}

	if typ, ok := typ.(Repeatable); ok {
		m.Assert(typ.IsRepeated(), "repeated flag should be used for repeated fields")
	}

	switch r := repeated.Type.(type) {
	case *flags.RepeatedFlags_Float:
		m.checkCommon(typ, r.Float, pgs.FloatT, pgs.FloatValueWKT, true)
	case *flags.RepeatedFlags_Double:
		m.checkCommon(typ, r.Double, pgs.DoubleT, pgs.DoubleValueWKT, true)
	case *flags.RepeatedFlags_Int32:
		m.checkCommon(typ, r.Int32, pgs.Int32T, pgs.Int32ValueWKT, true)
	case *flags.RepeatedFlags_Int64:
		m.checkCommon(typ, r.Int64, pgs.Int64T, pgs.Int64ValueWKT, true)
	case *flags.RepeatedFlags_Uint32:
		m.checkCommon(typ, r.Uint32, pgs.UInt32T, pgs.UInt32ValueWKT, true)
	case *flags.RepeatedFlags_Uint64:
		m.checkCommon(typ, r.Uint64, pgs.UInt64T, pgs.UInt64ValueWKT, true)
	case *flags.RepeatedFlags_Sint32:
		m.checkCommon(typ, r.Sint32, pgs.SInt32, pgs.UnknownWKT, true)
	case *flags.RepeatedFlags_Sint64:
		m.checkCommon(typ, r.Sint64, pgs.SInt64, pgs.UnknownWKT, true)
	case *flags.RepeatedFlags_Fixed32:
		m.checkCommon(typ, r.Fixed32, pgs.Fixed32T, pgs.UnknownWKT, true)
	case *flags.RepeatedFlags_Fixed64:
		m.checkCommon(typ, r.Fixed64, pgs.Fixed64T, pgs.UnknownWKT, true)
	case *flags.RepeatedFlags_Sfixed32:
		m.checkCommon(typ, r.Sfixed32, pgs.SFixed32, pgs.UnknownWKT, true)
	case *flags.RepeatedFlags_Sfixed64:
		m.checkCommon(typ, r.Sfixed64, pgs.SFixed64, pgs.UnknownWKT, true)
	case *flags.RepeatedFlags_Bool:
		m.checkCommon(typ, r.Bool, pgs.BoolT, pgs.BoolValueWKT, true)
	case *flags.RepeatedFlags_String_:
		m.checkCommon(typ, r.String_, pgs.StringT, pgs.StringValueWKT, true)
	case *flags.RepeatedFlags_Bytes:
		m.checkBytesSlice(typ, r.Bytes)
	case *flags.RepeatedFlags_Enum:
		m.checkEnumSlice(typ, r.Enum, pgs.EnumT, pgs.UnknownWKT)
	case *flags.RepeatedFlags_Duration:
		m.checkCommon(typ, r.Duration, pgs.MessageT, pgs.DurationWKT, true)
	case *flags.RepeatedFlags_Timestamp:
		m.checkTimestampSlice(typ, r.Timestamp)
	default:
		m.Failf("unknown repeated flag type (%T)", repeated.Type)
	}
}

func (m *Module) mustFieldType(ft FieldType) pgs.FieldType {
	typ, ok := ft.(pgs.FieldType)
	if !ok {
		m.Failf("unexpected field type (%T)", ft)
	}
	return typ
}
