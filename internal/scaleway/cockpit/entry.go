package cockpit

import (
	"fmt"
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

	_, err := jsonparser.ArrayEach(
		data,
		func(value []byte, t jsonparser.ValueType, _ int, _ error) {
			// Assert that both items in array are of type string
			switch i {
			case 0: // timestamp
				if t != jsonparser.String {
					parseError = jsonparser.MalformedStringError

					return
				}

				ts, err := jsonparser.ParseInt(value)
				if err != nil {
					parseError = fmt.Errorf("parsing timestamp: %w", err)

					return
				}

				e.Timestamp = time.Unix(0, ts)
			case 1: // log line
				if t != jsonparser.String {
					parseError = jsonparser.MalformedStringError

					return
				}

				v, err := jsonparser.ParseString(value)
				if err != nil {
					parseError = fmt.Errorf("parsing log line: %w", err)

					return
				}

				e.Line = v
			default:
				return // no-op
			}

			i++
		},
	)

	if parseError != nil {
		return parseError
	}

	if err != nil {
		return fmt.Errorf("parsing log entry array: %w", err)
	}

	return nil
}
