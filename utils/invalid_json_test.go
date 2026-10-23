package utils_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/Daniel-Ric/CraftBox/utils"
)

var invalidJson = []byte(`{"test": 001 /* comment */}frgvejmdorgvm`)

func TestInvalidJsonFix(t *testing.T) {
	var test struct {
		Test int `json:"test"`
	}
	err := utils.ParseJson(invalidJson, &test)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%+#v\n", test)
}

func TestLegacyJSONNumbersPreserveStringsAndComments(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		want        map[string]any
	}{
		{"leading zeros", `{"integer":001,"negative":-002,"fraction":000.5,"exponent":001e2,"zero":000}`, map[string]any{"integer": float64(1), "negative": float64(-2), "fraction": 0.5, "exponent": float64(100), "zero": float64(0)}},
		{"strings and comments", `{/* 001 */"text":"001 and \"002\"","array":[003,004],"normal":10}`, map[string]any{"text": `001 and "002"`, "array": []any{float64(3), float64(4)}, "normal": float64(10)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got map[string]any
			if err := utils.ParseJson([]byte(tc.input), &got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %#v, want %#v", got, tc.want)
			}
		})
	}
	for _, input := range []string{`{"n":001oops}`, `{"n":0x10}`, `{"n":01.}`, `{"text":"unterminated}`} {
		var got map[string]any
		if err := utils.ParseJson([]byte(input), &got); err == nil {
			t.Fatalf("accepted malformed JSON: %s", input)
		}
	}
}
