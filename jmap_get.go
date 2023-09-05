package trace2receiver

import (
	"encoding/json"
	"fmt"
	"time"
)

// Dictionary used to decode a JSON object record containing a
// single Trace2 Event record into a generic map.  This typedef
// is mainly to hide the awkward GOLANG syntax.
type jmap map[string]interface{}

// Optional keys/value pairs return a pointer to the value so
// that a non-present key is returned as a NULL pointer rather
// than an overloaded zero value.  If the map value is of a
// different type than requested, we return an error rather than
// trying to convert it (since we are trying to parse and validate
// a known document format.)
//
// All variants follow these rules:
//   Returns (p, nil) when successful.
//   Returns (nil, nil) if not present.
//   Returns (nil, err) if the value in the map is of a different type.

// Get an optional string value from the map.
func (jm *jmap) getOptionalString(key string) (*string, error) {
	var v interface{}
	var ok bool

	if v, ok = (*jm)[key]; !ok {
		return nil, nil
	}

	ps := new(string)

	switch v := v.(type) {
	case string:
		*ps = v
		return ps, nil
	default:
		return nil, fmt.Errorf("optional key '%s' does not have string value", key)
	}
}

func (jm *jmap) getOptionalInt64(key string) (*int64, error) {
	var v interface{}
	var ok bool

	if v, ok = (*jm)[key]; !ok {
		return nil, nil
	}

	pi := new(int64)

	// Allow both int and int64 in case we are unit testing. Allow float64
	// because the generic JSON decoder always creates floats (because JavaScript
	// does not have integer data types), so we have to convert it back.
	switch v := v.(type) {
	case int64:
		*pi = v
		return pi, nil
	case int:
		*pi = int64(v)
		return pi, nil
	case float64:
		*pi = int64(v)
		return pi, nil
	default:
		return nil, fmt.Errorf("key '%s' does not have an integer value", key)
	}
}

// Required keys/value pairs return the value or an hard error if
// the key is not present or the map value is of a different type
// than requested.
//
// All variants follow these rules:
//   Returns (p, nil) when successful.
//   Returns (nil, err) if not present.
//   Returns (nil, err) if value type wrong.

func (jm *jmap) getRequired(key string) (interface{}, error) {
	var v interface{}
	var ok bool

	if v, ok = (*jm)[key]; !ok {
		return nil, fmt.Errorf("key '%s' not present in Trace2 event", key)
	}
	return v, nil
}

func (jm *jmap) getRequiredBool(key string) (bool, error) {
	var v interface{}
	var err error

	if v, err = jm.getRequired(key); err != nil {
		return false, err
	}

	switch v := v.(type) {
	case bool:
		return v, nil
	default:
		return false, fmt.Errorf("key '%s' does not have bool value", key)
	}
}

func (jm *jmap) getRequiredString(key string) (string, error) {
	var v interface{}
	var err error

	if v, err = jm.getRequired(key); err != nil {
		return "", err
	}

	switch v := v.(type) {
	case string:
		return v, nil
	default:
		return "", fmt.Errorf("key '%s' does not have string value", key)
	}
}

func (jm *jmap) getRequiredInt64(key string) (int64, error) {
	var v interface{}
	var err error

	if v, err = jm.getRequired(key); err != nil {
		return 0, err
	}

	// Allow both int and int64 in case we are unit testing. Allow float64
	// because the generic JSON decoder always creates floats (because JavaScript
	// does not have integer data types), so we have to convert it back.
	switch v := v.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case float64:
		return int64(v), nil
	default:
		return 0, fmt.Errorf("key '%s' does not have an integer value", key)
	}
}

func (jm *jmap) getRequiredStringOrInt64(key string) (interface{}, error) {
	var v interface{}
	var err error

	if v, err = jm.getRequired(key); err != nil {
		return 0, err
	}

	// Allow both int and int64 in case we are unit testing. Allow float64
	// because the generic JSON decoder always creates floats (because JavaScript
	// does not have integer data types), so we have to convert it back.
	switch v := v.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case string:
		return v, nil
	default:
		return 0, fmt.Errorf("key '%s' does not have an integer or string value", key)
	}

}

func (jm *jmap) getRequiredFloat64(key string) (float64, error) {
	var v interface{}
	var err error

	if v, err = jm.getRequired(key); err != nil {
		return 0, err
	}

	// Allow both int and int64 in case the JSON writer is sloppy and doesn't
	// add a trailing .0 for whole numbers.  This is primarily for unit testing
	// since the generic JSON decoder always creates floats because of JavaScript
	// limitations.
	switch v := v.(type) {
	case float64:
		return v, nil
	case int64:
		return float64(v), nil
	case int:
		return float64(v), nil
	default:
		return 0.0, fmt.Errorf("key '%s' does not have an float value", key)
	}
}

func (jm *jmap) getRequiredTime(key string) (time.Time, error) {
	var v interface{}
	var err error

	if v, err = jm.getRequired(key); err != nil {
		return time.Time{}, err
	}

	switch v := v.(type) {
	case string:
		t, err := time.Parse("2006-01-02T15:04:05.999999Z", v)
		if err == nil {
			return t, err
		}
		// A version of GCM sends "+00:00" for the TZ rather than a "Z".
		t, err = time.Parse("2006-01-02T15:04:05.999999-07:00", v)
		return t, err
	default:
		return time.Time{}, fmt.Errorf("key '%s' does not have string value", key)
	}
}

// Extract required JSON array value.
//
// We usually use this for "argv", but leave it as an []interface{}
// type rather than assuming it is an []string (because we'll probably
// need that later).
func (jm *jmap) getRequiredArray(key string) ([]interface{}, error) {
	var v interface{}
	var err error

	if v, err = jm.getRequired(key); err != nil {
		return nil, err
	}

	switch v := v.(type) {
	case []interface{}:
		return v, nil
	case []string:
		// Implicitly case []string back to []interface{} for unit tests.
		vv := make([]interface{}, len(v))
		for k := range v {
			vv[k] = v[k]
		}
		return vv, nil
	default:
		return nil, fmt.Errorf("key '%s' is not an array", key)
	}
}

func (jm *jmap) getRequiredJsonValue(key string) (interface{}, error) {
	var v interface{}
	var err error

	if v, err = jm.getRequired(key); err != nil {
		return nil, err
	}

	switch v := v.(type) {
	case interface{}:
		_, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("key '%s' is not a JSON object", key)
		}
		return v, nil
	default:
		return nil, fmt.Errorf("key '%s' is not a JSON object", key)
	}
}
