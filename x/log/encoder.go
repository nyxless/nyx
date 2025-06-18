package log

type Encoder interface {
	Encode(entry *Entry) ([]byte, error)
}
