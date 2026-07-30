package market

import "testing"

func TestResolveSecID(t *testing.T) {
	tests := []struct {
		name string
		code string
		want string
	}{
		{name: "shanghai stock", code: "600000", want: "1.600000"},
		{name: "shanghai fund", code: "513630", want: "1.513630"},
		{name: "shenzhen stock", code: "000001", want: "0.000001"},
		{name: "trims whitespace", code: " 159934\n", want: "0.159934"},
		{name: "rejects non digit", code: "6abcde"},
		{name: "rejects wrong length", code: "60000"},
		{name: "rejects unsupported prefix", code: "900000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveSecID(tt.code); got != tt.want {
				t.Fatalf("ResolveSecID(%q) = %q, want %q", tt.code, got, tt.want)
			}
		})
	}
}
