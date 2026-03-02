package vectordb

import "testing"

func TestSearchService_NormalizeFilterValue_ShouldHandleSupportedTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input any
		want  string
	}{
		{name: "string", input: "  HeLLo ", want: "hello"},
		{name: "bytes", input: []byte("  WoRLD  "), want: "world"},
		{name: "float32", input: float32(1.5), want: "1.5"},
		{name: "float64", input: float64(2.5), want: "2.5"},
		{name: "int", input: int(3), want: "3"},
		{name: "int32", input: int32(4), want: "4"},
		{name: "int64", input: int64(5), want: "5"},
		{name: "bool true", input: true, want: "true"},
		{name: "bool false", input: false, want: "false"},
		{name: "default", input: struct{ A string }{A: "x"}, want: "{x}"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeFilterValue(tc.input); got != tc.want {
				t.Fatalf("normalizeFilterValue()=%q, want %q", got, tc.want)
			}
		})
	}
}
