package log

import (
	"bytes"
	"fmt"
	"strconv"
)

type TextEncoder struct{}

func (t *TextEncoder) Encode(entry *Entry) ([]byte, error) {
	var b bytes.Buffer

	if entry.Level != "" {
		b.WriteString(entry.Level)
		b.WriteString("\t")
	}

	b.WriteString(entry.Time)

	if entry.File != "" {
		b.WriteString("\t")
		b.WriteString(entry.File)
	}

	if entry.Msg != "" {
		b.WriteString("\t")
		b.WriteString(entry.Msg)
	}

	if entry.Formated {
		msg := fmt.Sprintf(entry.Msg, entry.Args)
		if msg != "" {
			b.WriteString("\t")
			b.WriteString(msg)
		}
	} else {
		for _, arg := range entry.Args {
			switch v := arg.(type) {
			case string:
				b.WriteString("\t")
				b.WriteString(v)
			case map[string]any:
				b.Write(t.parseMap(v))
			case Field:
				b.Write(t.parseField(v))
			default:
				b.WriteString("\t")
				b.WriteString(fmt.Sprint(v))
			}
		}
	}

	return b.Bytes(), nil
}

func (t *TextEncoder) parseField(field Field) []byte { // {{{
	var b bytes.Buffer

	b.WriteString("\t")
	b.WriteString(field.Key)
	b.WriteByte('[')

	if m, ok := field.Value.(map[string]any); ok {
		b.Write(t.parseMap(m))
	} else {
		b.WriteString(asString(field.Value))
	}

	b.WriteByte(']')

	return b.Bytes()
} // }}}

func (t *TextEncoder) parseMap(m map[string]any) []byte { // {{{
	var b bytes.Buffer

	for k, v := range m {
		b.WriteString("\t")
		b.WriteString(k)
		b.WriteByte('[')
		b.WriteString(asString(v))
		b.WriteByte(']')
	}

	return b.Bytes()
} // }}}

func asString(v any) string { // {{{
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	case int:
		return strconv.Itoa(val)
	case int8:
		return strconv.FormatInt(int64(val), 10)
	case int16:
		return strconv.FormatInt(int64(val), 10)
	case int32:
		return strconv.FormatInt(int64(val), 10)
	case int64:
		return strconv.FormatInt(val, 10)
	case uint:
		return strconv.FormatUint(uint64(val), 10)
	case uint8:
		return strconv.FormatUint(uint64(val), 10)
	case uint16:
		return strconv.FormatUint(uint64(val), 10)
	case uint32:
		return strconv.FormatUint(uint64(val), 10)
	case uint64:
		return strconv.FormatUint(val, 10)
	case float32:
		return strconv.FormatFloat(float64(val), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	case fmt.Stringer:
		return val.String()
	default:
		return fmt.Sprintf("%+v", v)
	}
} // }}}
