package trace2receiver

// Tests in this file are concerned with whether the key/value
// map (created by the JSON decoder) contains the necessary
// required keys, optionally contains optional keys, and that
// the key-value is of the expected type (and only casted in
// a few special cases).
//
// `evt_parse.go` will build upon this to decode Trace2 JSON
// event messages into actual structure fields.

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

var jm *jmap = &jmap{
	"optional-string":       "a",
	"optional-int":          42,
	"optional-int-as-float": 13.0,

	"required-string":             "b",
	"required-int":                99,
	"required-int-as-float":       7.0,
	"required-bool":               true,
	"required-float":              3.14,
	"required-time":               "2023-01-14T15:04:05.999999Z",
	"required-array-as-string":    []string{"a3", "b3", "c3"},
	"required-array-as-interface": append(make([]interface{}, 0), "x2", "y2"),

	"required-data-json": map[string]interface{}{
		"x": 1,
		"y": "foo",
	},

	"alternate-time": "2023-01-14T15:04:05.999999+00:00",
}

// Optional getter functions

func Test_getOptionalString_Present(t *testing.T) {
	ps, err := jm.getOptionalString("optional-string")
	if err != nil || ps == nil || *ps != "a" {
		t.Fatalf("getOptionalString")
	}
}
func Test_getOptionalString_NotPresent(t *testing.T) {
	ps, err := jm.getOptionalString("not-present-string")
	if err != nil || ps != nil {
		t.Fatalf("getOptionalString")
	}
}
func Test_getOptionalString_WrongType(t *testing.T) {
	_, err := jm.getOptionalString("optional-int")
	if err == nil {
		t.Fatal("getOptionaString")
	}
}

func Test_getOptionalInt64_Present(t *testing.T) {
	pi, err := jm.getOptionalInt64("optional-int")
	if err != nil || pi == nil || *pi != 42 {
		t.Fatalf("getOptionalInt64")
	}
}
func Test_getOptionalInt64_Present_AsFloat(t *testing.T) {
	pi, err := jm.getOptionalInt64("optional-int-as-float")
	if err != nil || pi == nil || *pi != 13 {
		t.Fatalf("getOptionalInt64")
	}
}
func Test_getOptionalInt64_NotPresent(t *testing.T) {
	pi, err := jm.getOptionalInt64("not-present-int")
	if err != nil || pi != nil {
		t.Fatalf("getOptionalInt64")
	}
}
func Test_getOptionalInt64_WrongType(t *testing.T) {
	_, err := jm.getOptionalInt64("optional-string")
	if err == nil {
		t.Fatalf("getOptionalInt64")
	}
}

// Required getter functions

func Test_getRequiredString_Present(t *testing.T) {
	s, err := jm.getRequiredString("required-string")
	if err != nil || s != "b" {
		t.Fatalf("getRequiredString")
	}
}
func Test_getRequiredString_NotPresent(t *testing.T) {
	_, err := jm.getRequiredString("not-present-string")
	if err == nil {
		t.Fatalf("getRequiredString")
	}
}
func Test_getRequiredString_WrongType(t *testing.T) {
	_, err := jm.getRequiredString("required-int")
	if err == nil {
		t.Fatalf("getRequiredString")
	}
}

func Test_getRequiredInt64_Present(t *testing.T) {
	i, err := jm.getRequiredInt64("required-int")
	if err != nil || i != 99 {
		t.Fatalf("getRequiredInt64")
	}
}
func Test_getRequiredInt64_Present_AsFloat(t *testing.T) {
	// JSON parser interns ints as floats and we need to undo that.
	i, err := jm.getRequiredInt64("required-int-as-float")
	if err != nil || i != 7 {
		t.Fatalf("getRequiredInt64")
	}
}
func Test_getRequiredInt64_NotPresent(t *testing.T) {
	_, err := jm.getRequiredInt64("not-present-int")
	if err == nil {
		t.Fatalf("getRequiredInt64")
	}
}
func Test_getRequiredInt64_WrongType(t *testing.T) {
	_, err := jm.getRequiredInt64("required-string")
	if err == nil {
		t.Fatalf("getRequiredInt64")
	}
}

func Test_getRequiredBool_Present(t *testing.T) {
	b, err := jm.getRequiredBool("required-bool")
	if err != nil || b != true {
		t.Fatalf("getRequiredBool")
	}
}
func Test_getRequiredBool_NotPresent(t *testing.T) {
	_, err := jm.getRequiredBool("not-present-bool")
	if err == nil {
		t.Fatalf("getRequiredBool")
	}
}
func Test_getRequiredBool_WrongType(t *testing.T) {
	_, err := jm.getRequiredBool("required-string")
	if err == nil {
		t.Fatalf("getRequiredBool")
	}
}

func Test_getRequiredStringOrInt64_Present(t *testing.T) {
	s, err1 := jm.getRequiredStringOrInt64("required-string")
	if err1 != nil || s.(string) != "b" {
		t.Fatalf("getRequiredStringOrInt64")
	}
	i, err2 := jm.getRequiredStringOrInt64("required-int")
	if err2 != nil || i.(int64) != 99 {
		t.Fatalf("getRequiredStringOrInt64")
	}
}

func float_is_near(v float64, v_ref float64) bool {
	return math.Abs(v-v_ref) < 0.001
}

func Test_getRequiredFloat64_Present(t *testing.T) {
	f, err := jm.getRequiredFloat64("required-float")
	if err != nil || !float_is_near(f, 3.14) {
		t.Fatalf("getRequiredFloat64")
	}
}
func Test_getRequiredFloat64_Present_AsInt(t *testing.T) {
	// Allow sloppy JSON writers to omit trailing .0 on whole numbers
	f, err := jm.getRequiredFloat64("required-int")
	if err != nil || !float_is_near(f, 99.0) {
		t.Fatalf("getRequiredFloat64")
	}
}
func Test_getRequiredFloat64_NotPresent(t *testing.T) {
	_, err := jm.getRequiredInt64("not-present-float")
	if err == nil {
		t.Fatalf("getRequiredFloat64")
	}
}
func Test_getRequiredFloat64_WrongType(t *testing.T) {
	_, err := jm.getRequiredFloat64("required-string")
	if err == nil {
		t.Fatalf("getRequiredFloat64")
	}
}

func Test_getRequiredTime_Present(t *testing.T) {
	tm, err := jm.getRequiredTime("required-time")
	if err != nil || tm.Year() != 2023 || tm.Month() != 1 || tm.Day() != 14 {
		t.Fatalf("getRequiredTime")
	}
}
func Test_tryAlternateTimeFormat(t *testing.T) {
	tm, err := jm.getRequiredTime("alternate-time")
	if err != nil || tm.Year() != 2023 || tm.Month() != 1 || tm.Day() != 14 {
		t.Fatalf("getRequiredTime on alternate format")
	}
}
func Test_getRequiredTime_NotPresent(t *testing.T) {
	_, err := jm.getRequiredString("not-present-time")
	if err == nil {
		t.Fatalf("getRequiredTime")
	}
}
func Test_getRequiredTime_WrongType_AsInt(t *testing.T) {
	_, err := jm.getRequiredTime("required-int")
	if err == nil {
		t.Fatalf("getRequiredTime")
	}
}
func Test_getRequiredTime_WrongType_AsString(t *testing.T) {
	// Try a string value and make time.Parse() fail.
	_, err := jm.getRequiredTime("required-string")
	if err == nil {
		t.Fatalf("getRequiredTime")
	}
}

func Test_getRequiredArray_Present_AsString(t *testing.T) {
	a, err := jm.getRequiredArray("required-array-as-string")
	if err != nil || len(a) != 3 || a[0] != "a3" {
		t.Fatalf("getRequiredArray")
	}
}
func Test_getRequiredArray_Present_AsInterface(t *testing.T) {
	a, err := jm.getRequiredArray("required-array-as-interface")
	if err != nil || len(a) != 2 || a[0] != "x2" {
		t.Fatalf("getRequiredArray")
	}
}
func Test_getRequiredArray_NotPresent(t *testing.T) {
	_, err := jm.getRequiredArray("not-present-array")
	if err == nil {
		t.Fatalf("getRequiredArray")
	}
}
func Test_getRequiredArray_WrongType(t *testing.T) {
	_, err := jm.getRequiredArray("required-string")
	if err == nil {
		t.Fatalf("getRequiredArray")
	}
}

func Test_getRequiredSerialized_Present(t *testing.T) {
	jv, err := jm.getRequiredJsonValue("required-data-json")
	if err != nil {
		t.Fatalf("getRequiredSerialized")
	}
	b, err := json.Marshal(jv)
	if err != nil {
		t.Fatalf("getRequiredSerialized")
	}
	s := string(b)
	if !strings.Contains(s, "\"x\":1") {
		t.Fatalf("getRequiredSerialized")
	}
	if !strings.Contains(s, "\"y\":\"foo\"") {
		t.Fatalf("getRequiredSerialized")
	}
}
