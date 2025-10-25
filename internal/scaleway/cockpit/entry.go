package cockpit

import (
	"time"

	"github.com/buger/jsonparser"
)

type Entry struct {
	Timestamp time.Time
	Line      string
}

func (e *Entry) UnmarshalJSON(data []byte) error {
	var (
		i          int
		parseError error
	)
	_, err := jsonparser.ArrayEach(data, func(value []byte, t jsonparser.ValueType, _ int, _ error) {
		// assert that both items in array are of type string
		switch i {
		case 0: // timestamp
			if t != jsonparser.String {
				parseError = jsonparser.MalformedStringError
				return
			}
			ts, err := jsonparser.ParseInt(value)
			if err != nil {
				parseError = err
				return
			}
			e.Timestamp = time.Unix(0, ts)
		case 1: // value
			if t != jsonparser.String {
				parseError = jsonparser.MalformedStringError
				return
			}
			v, err := jsonparser.ParseString(value)
			if err != nil {
				parseError = err
				return
			}
			e.Line = v
		}
		i++
	})
	if parseError != nil {
		return parseError
	}
	return err
}
