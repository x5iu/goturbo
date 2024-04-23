package upgrade

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	type testcase struct {
		Version string
		Want    SemanticVersion
	}
	var testcases = []testcase{
		{
			Version: "v0.0.5",
			Want: SemanticVersion{
				Major: 0,
				Minor: 0,
				Patch: 5,
			},
		},
		{
			Version: "v1.20.103",
			Want: SemanticVersion{
				Major: 1,
				Minor: 20,
				Patch: 103,
			},
		},
		{
			Version: "v0.12.4 beta",
			Want: SemanticVersion{
				Major: 0,
				Minor: 12,
				Patch: 4,
			},
		},
	}
	for _, tc := range testcases {
		sv, err := parse(tc.Version)
		if err != nil {
			t.Errorf("parse: %s", err)
			return
		}
		if !reflect.DeepEqual(sv, tc.Want) {
			t.Errorf("parse: want %q, got %q", tc.Want, sv)
			return
		}
	}
}
