package notify

import "testing"

func TestAppleScriptStringEscapesControlCharacters(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: `""`},
		{name: "quotes and slash", in: `say "buy"\sell`, want: `"say \"buy\"\\sell"`},
		{name: "line controls", in: "buy\r\nnow\tplease", want: `"buy  now please"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := appleScriptString(tt.in); got != tt.want {
				t.Fatalf("appleScriptString = %q, want %q", got, tt.want)
			}
		})
	}
}
