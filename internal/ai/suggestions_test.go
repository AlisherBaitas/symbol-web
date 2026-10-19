package ai

import (
	"reflect"
	"strings"
	"testing"
)

func Test_mockSuggestions(t *testing.T) {
	// 43 символа — граница: такой текст ещё не обрезается
	exact := strings.Repeat("a", 43)
	// Длинный текст обрезается до 40 символов + "..."
	cut := strings.Repeat("a", 40) + "..."

	type args struct {
		text string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "simple word",
			args: args{text: "Hello"},
			want: []string{"Hello!", "Hello World", "~ Hello ~", "HELLO"},
		},
		{
			name: "spaces and line breaks are collapsed",
			args: args{text: "  Happy \n  Birth  "},
			want: []string{"Happy Birth!", "Happy Birth World", "~ Happy Birth ~", "HAPPY BIRTH"},
		},
		{
			name: "43 characters are kept as is",
			args: args{text: exact},
			want: []string{exact + "!", exact + " World", "~ " + exact + " ~", strings.ToUpper(exact)},
		},
		{
			name: "long text is shortened",
			args: args{text: strings.Repeat("a", 60)},
			want: []string{cut + "!", cut + " World", "~ " + cut + " ~", strings.ToUpper(cut)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mockSuggestions(tt.args.text)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mockSuggestions() = %v, want %v", got, tt.want)
			}
			// Каждая подсказка должна быть короче 50 символов
			for _, s := range got {
				if len(s) >= maxSuggestionChars {
					t.Errorf("mockSuggestions() item %q has %d chars, want < %d", s, len(s), maxSuggestionChars)
				}
			}
		})
	}
}

func Test_chatURL(t *testing.T) {
	type args struct {
		baseURL string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "ollama root",
			args: args{baseURL: "http://localhost:11434"},
			want: "http://localhost:11434/v1/chat/completions",
		},
		{
			name: "trailing slash",
			args: args{baseURL: "http://localhost:11434/"},
			want: "http://localhost:11434/v1/chat/completions",
		},
		{
			name: "already ends with v1",
			args: args{baseURL: "https://api.example.com/v1"},
			want: "https://api.example.com/v1/chat/completions",
		},
		{
			name: "v1 with trailing slash",
			args: args{baseURL: "https://api.example.com/v1/"},
			want: "https://api.example.com/v1/chat/completions",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := chatURL(tt.args.baseURL); got != tt.want {
				t.Errorf("chatURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseSuggestions(t *testing.T) {
	type args struct {
		answer string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "list markers are removed",
			args: args{answer: "1. Happy Birthday!\n2) Happy Birthday Team!\n- Happy Day\n* Happy You\n• Happy Me"},
			want: []string{"Happy Birthday!", "Happy Birthday Team!", "Happy Day", "Happy You", "Happy Me"},
		},
		{
			name: "quotes and spaces are trimmed",
			args: args{answer: "  \"Hello\"  \n'World'\n`Code`"},
			want: []string{"Hello", "World", "Code"},
		},
		{
			name: "duplicates ignore case",
			args: args{answer: "Hello\nhello\nHELLO\nHi"},
			want: []string{"Hello", "Hi"},
		},
		{
			name: "empty lines and CRLF",
			args: args{answer: "\n\nHello\r\n\r\nWorld\r\n"},
			want: []string{"Hello", "World"},
		},
		{
			name: "number without dot is kept",
			args: args{answer: "2026 Happy New Year"},
			want: []string{"2026 Happy New Year"},
		},
		{
			name: "50 characters are too long, 49 are fine",
			args: args{answer: strings.Repeat("a", 50) + "\n" + strings.Repeat("b", 49)},
			want: []string{strings.Repeat("b", 49)},
		},
		{
			name: "non ASCII lines are dropped",
			args: args{answer: "Привет\nHello\ntab\there"},
			want: []string{"Hello"},
		},
		{
			name: "at most 5 suggestions",
			args: args{answer: "a1\nb2\nc3\nd4\ne5\nf6\ng7"},
			want: []string{"a1", "b2", "c3", "d4", "e5"},
		},
		{
			name: "empty answer",
			args: args{answer: ""},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseSuggestions(tt.args.answer); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseSuggestions() = %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_printableASCII(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "letters digits punctuation", args: args{s: "Hello, World! 123"}, want: true},
		{name: "empty string", args: args{s: ""}, want: true},
		{name: "space and tilde are the edges", args: args{s: " ~"}, want: true},
		{name: "cyrillic", args: args{s: "Привет"}, want: false},
		{name: "tab", args: args{s: "a\tb"}, want: false},
		{name: "line break", args: args{s: "a\nb"}, want: false},
		{name: "DEL character", args: args{s: "\x7f"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := printableASCII(tt.args.s); got != tt.want {
				t.Errorf("printableASCII() = %v, want %v", got, tt.want)
			}
		})
	}
}
