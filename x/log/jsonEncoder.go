package log

import (
	"encoding/json"
	"fmt"
)

type JsonEncoder struct{}

func (e *JsonEncoder) Encode(entry *Entry) ([]byte, error) {
	msg := entry.Msg
	m := map[string]interface{}{
		"time": entry.Time,
	}

	if entry.Level != "" {
		m["level"] = entry.Level
	}

	if entry.File != "" {
		m["file"] = entry.File
	}

	if entry.Formated {
		msg = fmt.Sprintf(entry.Msg, entry.Args)
	} else {
		for _, arg := range entry.Args {
			switch v := arg.(type) {
			case string:
				msg += v
			case map[string]any:
				for mk, mv := range v {
					m[mk] = mv
				}
			case Field:
				m[v.Key] = v.Value
			default:
				msg += fmt.Sprint(v)
			}
		}
	}

	if msg != "" {
		m["msg"] = msg
	}

	return json.Marshal(m)
}
