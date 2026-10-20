package ai

// [Fix] Тесты правил выбора баннера были полностью закомментированы и не
// компилировались бы: GetRecommend теперь возвращает (Recommendation, error).
// Покрытие recommendations.go было 0%. Тесты восстановлены и дополнены
// негативными кейсами.

import (
	"reflect"
	"strings"
	"testing"
)

func TestGetRecommend(t *testing.T) {
	type args struct {
		input string
	}
	tests := []struct {
		name string
		args args
		want string // рекомендованный баннер
	}{
		{name: "uppercase word", args: args{input: "WELCOME"}, want: "shadow"},
		{name: "uppercase with symbol", args: args{input: "HELLO WORLD!"}, want: "shadow"},
		{name: "short mixed case", args: args{input: "Hello"}, want: "shadow"},
		{name: "six characters", args: args{input: "abcdef"}, want: "shadow"},
		{name: "digits only, short", args: args{input: "123456"}, want: "shadow"},
		{name: "seven characters mixed case", args: args{input: "Welcome"}, want: "thinkertoy"},
		{name: "medium mixed case", args: args{input: "Hello World"}, want: "thinkertoy"},
		{name: "twelve characters mixed case", args: args{input: "Hello World!"}, want: "thinkertoy"},
		{name: "thirteen characters mixed case", args: args{input: "Hello Worlds!"}, want: "standard"},
		{name: "long mixed case", args: args{input: "Hello there friend"}, want: "standard"},
		{name: "long digits", args: args{input: "1234567890123"}, want: "standard"},
		{name: "real line break", args: args{input: "Hello\nWorld"}, want: "standard"},
		{name: "typed \\n line break", args: args{input: "HI\\nYO"}, want: "standard"},
		{name: "symbol heavy", args: args{input: "@#$%!abc"}, want: "thinkertoy"},
		{name: "symbols beat short text", args: args{input: "#1!"}, want: "thinkertoy"},
		// [Fix] Перенос строки из браузера "\r\n" — обычный многострочный текст.
		{name: "browser CRLF line break", args: args{input: "Hello\r\nWorld"}, want: "standard"},
		{name: "long uppercase text", args: args{input: "HAPPY BIRTHDAY TEAM"}, want: "shadow"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetRecommend(tt.args.input)
			if err != nil {
				t.Fatalf("GetRecommend() error = %v", err)
			}
			if got.Recommended != tt.want {
				t.Errorf("GetRecommend().Recommended = %v, want %v", got.Recommended, tt.want)
			}
			if got.Reasoning == "" {
				t.Errorf("GetRecommend().Reasoning is empty")
			}
			// Две альтернативы: остальные баннеры, без повторов и без рекомендованного
			if len(got.Alternatives) != 2 {
				t.Fatalf("GetRecommend() has %d alternatives, want 2", len(got.Alternatives))
			}
			seen := map[string]bool{got.Recommended: true}
			for _, alt := range got.Alternatives {
				if !validVariationBanner(alt.Banner) || seen[alt.Banner] {
					t.Errorf("GetRecommend() alternative banner %q is unknown or repeated", alt.Banner)
				}
				seen[alt.Banner] = true
				if alt.Score <= 0 || alt.Score >= 1 || alt.Reason == "" {
					t.Errorf("GetRecommend() alternative %+v has bad score or empty reason", alt)
				}
			}
		})
	}
}

func TestGetRecommend_Deterministic(t *testing.T) {
	// Правила без LLM: одинаковый ввод -> одинаковый ответ
	a, _ := GetRecommend("Hello World")
	b, _ := GetRecommend("Hello World")
	if !reflect.DeepEqual(a, b) {
		t.Errorf("GetRecommend() is not deterministic: %+v vs %+v", a, b)
	}
}

func Test_analyzeInput(t *testing.T) {
	type args struct {
		input []rune
	}
	tests := []struct {
		name string
		args args
		want Stats
	}{
		{
			name: "mixed text with punctuation",
			args: args{input: []rune("Hello, World!")},
			want: Stats{upper: 2, lower: 8, special: 2, newLines: 0, length: 13, nonSpace: 12},
		},
		{
			name: "outer spaces are trimmed",
			args: args{input: []rune("  AB c  ")},
			want: Stats{upper: 2, lower: 1, special: 0, newLines: 0, length: 4, nonSpace: 3},
		},
		{
			name: "typed \\n becomes a line break",
			args: args{input: []rune("Hi\\nYo")},
			want: Stats{upper: 2, lower: 2, special: 0, newLines: 1, length: 5, nonSpace: 4},
		},
		{
			name: "digits are not letters or symbols",
			args: args{input: []rune("123")},
			want: Stats{length: 3, nonSpace: 3},
		},
		{
			name: "empty input",
			args: args{input: []rune("")},
			want: Stats{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := analyzeInput(tt.args.input); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("analyzeInput() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_isUpperCase(t *testing.T) {
	type args struct {
		ch rune
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "A", args: args{ch: 'A'}, want: true},
		{name: "Z", args: args{ch: 'Z'}, want: true},
		{name: "lowercase a", args: args{ch: 'a'}, want: false},
		{name: "digit", args: args{ch: '1'}, want: false},
		{name: "symbol", args: args{ch: '@'}, want: false},
		{name: "cyrillic uppercase", args: args{ch: 'Ж'}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUpperCase(tt.args.ch); got != tt.want {
				t.Errorf("isUpperCase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isLowerCase(t *testing.T) {
	type args struct {
		ch rune
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "a", args: args{ch: 'a'}, want: true},
		{name: "z", args: args{ch: 'z'}, want: true},
		{name: "uppercase A", args: args{ch: 'A'}, want: false},
		{name: "digit", args: args{ch: '9'}, want: false},
		{name: "backtick before a", args: args{ch: '`'}, want: false},
		{name: "cyrillic lowercase", args: args{ch: 'ж'}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLowerCase(tt.args.ch); got != tt.want {
				t.Errorf("isLowerCase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_hasNewLine(t *testing.T) {
	type args struct {
		ch rune
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "line break", args: args{ch: '\n'}, want: true},
		{name: "letter n", args: args{ch: 'n'}, want: false},
		{name: "carriage return", args: args{ch: '\r'}, want: false},
		{name: "space", args: args{ch: ' '}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasNewLine(tt.args.ch); got != tt.want {
				t.Errorf("hasNewLine() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isSpecial(t *testing.T) {
	type args struct {
		ch rune
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "exclamation", args: args{ch: '!'}, want: true},
		{name: "slash", args: args{ch: '/'}, want: true},
		{name: "colon", args: args{ch: ':'}, want: true},
		{name: "at sign", args: args{ch: '@'}, want: true},
		{name: "square bracket", args: args{ch: '['}, want: true},
		{name: "backtick", args: args{ch: '`'}, want: true},
		{name: "curly bracket", args: args{ch: '{'}, want: true},
		{name: "tilde", args: args{ch: '~'}, want: true},
		{name: "space", args: args{ch: ' '}, want: false},
		{name: "digit", args: args{ch: '0'}, want: false},
		{name: "uppercase letter", args: args{ch: 'A'}, want: false},
		{name: "lowercase letter", args: args{ch: 'a'}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSpecial(tt.args.ch); got != tt.want {
				t.Errorf("isSpecial() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_convertLineBreaks(t *testing.T) {
	type args struct {
		input string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "typed \\n in the middle", args: args{input: "a\\nb"}, want: "a\nb"},
		{name: "two typed \\n", args: args{input: "\\n\\n"}, want: "\n\n"},
		{name: "no line breaks", args: args{input: "hello"}, want: "hello"},
		{name: "backslash at the end", args: args{input: "end\\"}, want: "end\\"},
		{name: "other escape is kept", args: args{input: "a\\tb"}, want: "a\\tb"},
		{name: "empty", args: args{input: ""}, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertLineBreaks(tt.args.input); got != tt.want {
				t.Errorf("convertLineBreaks() = %q, want %q", got, tt.want)
			}
		})
	}
}

// [Fix] Негативные кейсы: ввод, который нельзя нарисовать, даёт ошибку (400 в обработчике).
func TestGetRecommend_InvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "empty", input: ""},
		{name: "only spaces", input: "   "},
		{name: "cyrillic", input: "Привет"},
		{name: "emoji", input: "hi 🙂"},
		{name: "tab", input: "a\tb"},
		{name: "too long", input: strings.Repeat("a", 1001)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetRecommend(tt.input)
			if err == nil {
				t.Errorf("GetRecommend(%q) = %+v, want an error", tt.input, got)
			}
		})
	}
}
