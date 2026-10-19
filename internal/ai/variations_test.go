package ai

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func Test_validateVariationInput(t *testing.T) {
	type args struct {
		text string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "simple word", args: args{text: "hello"}, wantErr: false},
		{name: "line break", args: args{text: "line1\nline2"}, wantErr: false},
		{name: "windows line break", args: args{text: "line1\r\nline2"}, wantErr: false},
		{name: "1000 characters", args: args{text: strings.Repeat("a", 1000)}, wantErr: false},
		{name: "empty", args: args{text: ""}, wantErr: true},
		{name: "only spaces", args: args{text: "   "}, wantErr: true},
		{name: "only line breaks", args: args{text: "\n\n"}, wantErr: true},
		{name: "1001 characters", args: args{text: strings.Repeat("a", 1001)}, wantErr: true},
		{name: "lone carriage return", args: args{text: "a\rb"}, wantErr: true},
		{name: "tab", args: args{text: "a\tb"}, wantErr: true},
		{name: "cyrillic", args: args{text: "Привет"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVariationInput(tt.args.text)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateVariationInput() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !errors.Is(err, ErrInvalidVariationInput) {
				t.Errorf("validateVariationInput() error = %v, want ErrInvalidVariationInput", err)
			}
		})
	}
}

func Test_mockVariations(t *testing.T) {
	// Длинный текст обрезается до 42 символов + "..."
	cut := strings.Repeat("x", 42) + "..."

	type args struct {
		text string
	}
	tests := []struct {
		name string
		args args
		want []Variation
	}{
		{
			name: "simple word",
			args: args{text: "hello"},
			want: []Variation{
				{Text: "Hello", Description: "Professional style", SuggestedBanner: "standard"},
				{Text: "HELLO!", Description: "Bold and emphatic", SuggestedBanner: "shadow"},
				{Text: "Hi, hello", Description: "Friendly and casual", SuggestedBanner: "standard"},
				{Text: "~ hello ~", Description: "Decorative style", SuggestedBanner: "thinkertoy"},
			},
		},
		{
			name: "spaces and line breaks are collapsed",
			args: args{text: "  big   World \n"},
			want: []Variation{
				{Text: "Big World", Description: "Professional style", SuggestedBanner: "standard"},
				{Text: "BIG WORLD!", Description: "Bold and emphatic", SuggestedBanner: "shadow"},
				{Text: "Hi, big world", Description: "Friendly and casual", SuggestedBanner: "standard"},
				{Text: "~ big World ~", Description: "Decorative style", SuggestedBanner: "thinkertoy"},
			},
		},
		{
			name: "long text is shortened",
			args: args{text: strings.Repeat("x", 60)},
			want: []Variation{
				{Text: "X" + cut[1:], Description: "Professional style", SuggestedBanner: "standard"},
				{Text: strings.ToUpper(cut) + "!", Description: "Bold and emphatic", SuggestedBanner: "shadow"},
				{Text: "Hi, " + cut, Description: "Friendly and casual", SuggestedBanner: "standard"},
				{Text: "~ " + cut + " ~", Description: "Decorative style", SuggestedBanner: "thinkertoy"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mockVariations(tt.args.text)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mockVariations() = %+v, want %+v", got, tt.want)
			}
			// Mock-ответ должен проходить ту же проверку, что и ответ LLM
			for _, v := range got {
				if !validVariationText(v.Text) || !validVariationDescription(v.Description) || !validVariationBanner(v.SuggestedBanner) {
					t.Errorf("mockVariations() item %+v is not valid", v)
				}
			}
		})
	}
}

func Test_parseVariations(t *testing.T) {
	valid := `[
		{"text":"Hello!","description":"Bold","suggested_banner":"shadow"},
		{"text":"hello","description":"Calm","suggested_banner":"standard"},
		{"text":"~ hi ~","description":"Fancy","suggested_banner":"thinkertoy"}
	]`
	validWant := []Variation{
		{Text: "Hello!", Description: "Bold", SuggestedBanner: "shadow"},
		{Text: "hello", Description: "Calm", SuggestedBanner: "standard"},
		{Text: "~ hi ~", Description: "Fancy", SuggestedBanner: "thinkertoy"},
	}
	item := func(text, desc, banner string) string {
		return `{"text":"` + text + `","description":"` + desc + `","suggested_banner":"` + banner + `"}`
	}

	type args struct {
		answer string
	}
	tests := []struct {
		name    string
		args    args
		want    []Variation
		wantErr bool
	}{
		{name: "valid array", args: args{answer: valid}, want: validWant},
		{name: "markdown json block", args: args{answer: "```json\n" + valid + "\n```"}, want: validWant},
		{
			name: "outer spaces in fields are trimmed",
			args: args{answer: "[" + item(" Hello! ", " Bold ", "shadow") + "," + item("a", "b", "standard") + "," + item("c", "d", "thinkertoy") + "]"},
			want: []Variation{
				{Text: "Hello!", Description: "Bold", SuggestedBanner: "shadow"},
				{Text: "a", Description: "b", SuggestedBanner: "standard"},
				{Text: "c", Description: "d", SuggestedBanner: "thinkertoy"},
			},
		},
		{
			name: "same text in other case is allowed",
			args: args{answer: "[" + item("Hello", "a", "shadow") + "," + item("HELLO", "b", "shadow") + "," + item("hello", "c", "shadow") + "]"},
			want: []Variation{
				{Text: "Hello", Description: "a", SuggestedBanner: "shadow"},
				{Text: "HELLO", Description: "b", SuggestedBanner: "shadow"},
				{Text: "hello", Description: "c", SuggestedBanner: "shadow"},
			},
		},
		{name: "not JSON", args: args{answer: "Sure! Here are some ideas"}, wantErr: true},
		{name: "text around JSON", args: args{answer: "Here: " + valid}, wantErr: true},
		{name: "only 2 items", args: args{answer: "[" + item("a", "a", "shadow") + "," + item("b", "b", "shadow") + "]"}, wantErr: true},
		{
			name:    "6 items",
			args:    args{answer: "[" + item("a", "a", "shadow") + "," + item("b", "b", "shadow") + "," + item("c", "c", "shadow") + "," + item("d", "d", "shadow") + "," + item("e", "e", "shadow") + "," + item("f", "f", "shadow") + "]"},
			wantErr: true,
		},
		{name: "unknown banner", args: args{answer: "[" + item("a", "a", "comic") + "," + item("b", "b", "shadow") + "," + item("c", "c", "shadow") + "]"}, wantErr: true},
		{name: "duplicate text", args: args{answer: "[" + item("a", "a", "shadow") + "," + item("a", "b", "shadow") + "," + item("c", "c", "shadow") + "]"}, wantErr: true},
		{name: "text too long", args: args{answer: "[" + item(strings.Repeat("a", 50), "a", "shadow") + "," + item("b", "b", "shadow") + "," + item("c", "c", "shadow") + "]"}, wantErr: true},
		{name: "description too long", args: args{answer: "[" + item("a", strings.Repeat("d", 30), "shadow") + "," + item("b", "b", "shadow") + "," + item("c", "c", "shadow") + "]"}, wantErr: true},
		{name: "empty answer", args: args{answer: ""}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseVariations(tt.args.answer)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseVariations() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && !errors.Is(err, ErrInvalidLLMResponse) {
				t.Errorf("parseVariations() error = %v, want ErrInvalidLLMResponse", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseVariations() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_validVariationText(t *testing.T) {
	type args struct {
		text string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "simple text", args: args{text: "Hello!"}, want: true},
		{name: "49 characters", args: args{text: strings.Repeat("a", 49)}, want: true},
		{name: "empty", args: args{text: ""}, want: false},
		{name: "50 characters", args: args{text: strings.Repeat("a", 50)}, want: false},
		{name: "cyrillic", args: args{text: "Привет"}, want: false},
		{name: "tab", args: args{text: "a\tb"}, want: false},
		{name: "line break", args: args{text: "a\nb"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validVariationText(tt.args.text); got != tt.want {
				t.Errorf("validVariationText() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_validVariationDescription(t *testing.T) {
	type args struct {
		description string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "simple text", args: args{description: "Bold style"}, want: true},
		{name: "29 characters", args: args{description: strings.Repeat("d", 29)}, want: true},
		{name: "cyrillic is allowed", args: args{description: "Смелый стиль"}, want: true},
		{name: "empty", args: args{description: ""}, want: false},
		{name: "only spaces", args: args{description: "   "}, want: false},
		{name: "30 characters", args: args{description: strings.Repeat("d", 30)}, want: false},
		{name: "control character", args: args{description: "a\tb"}, want: false},
		{name: "invalid UTF-8", args: args{description: "\xff"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validVariationDescription(tt.args.description); got != tt.want {
				t.Errorf("validVariationDescription() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_validVariationBanner(t *testing.T) {
	type args struct {
		banner string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "standard", args: args{banner: "standard"}, want: true},
		{name: "shadow", args: args{banner: "shadow"}, want: true},
		{name: "thinkertoy", args: args{banner: "thinkertoy"}, want: true},
		{name: "capital letter", args: args{banner: "Standard"}, want: false},
		{name: "with .txt", args: args{banner: "standard.txt"}, want: false},
		{name: "unknown", args: args{banner: "comic"}, want: false},
		{name: "empty", args: args{banner: ""}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validVariationBanner(tt.args.banner); got != tt.want {
				t.Errorf("validVariationBanner() = %v, want %v", got, tt.want)
			}
		})
	}
}
