package ascii

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// Banner files live in the project root
const (
	standardFile   = "../../standard.txt"
	shadowFile     = "../../shadow.txt"
	thinkertoyFile = "../../thinkertoy.txt"
)

// Expected art from the banner files
var (
	artHi = " _    _   _  \n" +
		"| |  | | (_) \n" +
		"| |__| |  _  \n" +
		"|  __  | | | \n" +
		"| |  | | | | \n" +
		"|_|  |_| |_| \n" +
		"             \n" +
		"             \n"
	artStandardA = "           \n" +
		"    /\\     \n" +
		"   /  \\    \n" +
		"  / /\\ \\   \n" +
		" / ____ \\  \n" +
		"/_/    \\_\\ \n" +
		"           \n" +
		"           \n"
	artStandardAB = "        \n" +
		"        \n" +
		"  __ _  \n" +
		" / _` | \n" +
		"| (_| | \n" +
		" \\__,_| \n" +
		"        \n" +
		"        \n" +
		" _      \n" +
		"| |     \n" +
		"| |__   \n" +
		"| '_ \\  \n" +
		"| |_) | \n" +
		"|_.__/  \n" +
		"        \n" +
		"        \n"
	artShadowA = "         \n" +
		"  _|_|   \n" +
		"_|    _| \n" +
		"_|_|_|_| \n" +
		"_|    _| \n" +
		"_|    _| \n" +
		"         \n" +
		"         \n"
	artThinkertoyA = "      \n" +
		"  O   \n" +
		" / \\  \n" +
		"o---o \n" +
		"|   | \n" +
		"o   o \n" +
		"      \n" +
		"      \n"
)

func TestGenerate(t *testing.T) {
	type args struct {
		input    string
		fileName string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
		errIs   error
	}{
		{name: "standard Hi", args: args{input: "Hi", fileName: standardFile}, want: artHi},
		{name: "standard A", args: args{input: "A", fileName: standardFile}, want: artStandardA},
		{name: "shadow A", args: args{input: "A", fileName: shadowFile}, want: artShadowA},
		{name: "thinkertoy A", args: args{input: "A", fileName: thinkertoyFile}, want: artThinkertoyA},
		{name: "typed \\n splits lines", args: args{input: "a\\nb", fileName: standardFile}, want: artStandardAB},
		{name: "real line break splits lines", args: args{input: "a\nb", fileName: standardFile}, want: artStandardAB},
		{name: "line break at the start", args: args{input: "\\nA", fileName: standardFile}, want: "\n" + artStandardA},
		{name: "line break at the end", args: args{input: "A\\n", fileName: standardFile}, want: artStandardA + "\n"},
		{name: "empty text", args: args{input: "", fileName: standardFile}, wantErr: true},
		{name: "cyrillic", args: args{input: "Привет", fileName: standardFile}, wantErr: true},
		{name: "banner file not found", args: args{input: "Hi", fileName: "../../nope.txt"}, wantErr: true, errIs: ErrBannerNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Generate(tt.args.input, tt.args.fileName)
			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.errIs != nil && !errors.Is(err, tt.errIs) {
				t.Errorf("Generate() error = %v, want %v", err, tt.errIs)
			}
			if got != tt.want {
				t.Errorf("Generate() =\n%s\nwant\n%s", got, tt.want)
			}
		})
	}
}

func TestGenerate_NoLeftoversBetweenCalls(t *testing.T) {
	// Generate uses a global buffer, a second call must not contain the first result
	first, _ := Generate("Hi", standardFile)
	second, _ := Generate("Hi", standardFile)
	if first != second || artResult != "" {
		t.Errorf("Generate() keeps old result between calls")
	}
}

func Test_banner(t *testing.T) {
	type args struct {
		input    []rune
		fileName string
	}
	tests := []struct {
		name     string
		args     args
		wantKeys []rune
		wantErr  bool
	}{
		{name: "two letters", args: args{input: []rune("Hi"), fileName: standardFile}, wantKeys: []rune{'H', 'i'}},
		{name: "duplicates are stored once", args: args{input: []rune("aab"), fileName: standardFile}, wantKeys: []rune{'a', 'b'}},
		{name: "line break becomes space", args: args{input: []rune("a\\nb"), fileName: standardFile}, wantKeys: []rune{' ', 'a', 'b'}},
		{name: "file not found", args: args{input: []rune("Hi"), fileName: "../../nope.txt"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := banner(tt.args.input, tt.args.fileName)
			if (err != nil) != tt.wantErr {
				t.Errorf("banner() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.wantKeys) {
				t.Errorf("banner() has %d characters, want %d", len(got), len(tt.wantKeys))
			}
			for _, k := range tt.wantKeys {
				if len(got[k]) != 8 {
					t.Errorf("banner() character %q has %d lines, want 8", k, len(got[k]))
				}
			}
		})
	}

	// Lines of 'H' must match the banner file
	got, _ := banner([]rune("H"), standardFile)
	wantH := strings.Split(strings.TrimSuffix(artHi, "\n"), "\n")
	for i := range wantH {
		wantH[i] = wantH[i][:9] // 'H' is the first 9 columns of "Hi"
	}
	if !reflect.DeepEqual(got['H'], wantH) {
		t.Errorf("banner() 'H' = %q, want %q", got['H'], wantH)
	}
}

func Test_printer(t *testing.T) {
	lookUp, err := banner([]rune("ab"), standardFile)
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		lookUp map[rune][]string
		input  string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "two lines", args: args{lookUp: lookUp, input: "a\\nb"}, want: artStandardAB},
		{name: "empty line between", args: args{lookUp: lookUp, input: "a\\n\\nb"}, want: artStandardAB[:len(artStandardAB)/2] + "\n" + artStandardAB[len(artStandardAB)/2:]},
		{name: "only line break", args: args{lookUp: lookUp, input: "\\n"}, want: "\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			artResult = ""
			t.Cleanup(func() { artResult = "" })
			printer(tt.args.lookUp, tt.args.input)
			if artResult != tt.want {
				t.Errorf("printer() =\n%q\nwant\n%q", artResult, tt.want)
			}
		})
	}
}

func Test_printRow(t *testing.T) {
	lookUp, err := banner([]rune("Hi"), standardFile)
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		lookUp map[rune][]string
		input  string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "Hi", args: args{lookUp: lookUp, input: "Hi"}, want: artHi},
		{name: "empty word gives 8 empty lines", args: args{lookUp: lookUp, input: ""}, want: strings.Repeat("\n", 8)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			artResult = ""
			t.Cleanup(func() { artResult = "" })
			printRow(tt.args.lookUp, tt.args.input)
			if artResult != tt.want {
				t.Errorf("printRow() =\n%q\nwant\n%q", artResult, tt.want)
			}
		})
	}
}

func Test_bubbleSort(t *testing.T) {
	type args struct {
		input []rune
	}
	tests := []struct {
		name string
		args args
		want []rune
	}{
		{name: "unsorted", args: args{input: []rune("cab")}, want: []rune("abc")},
		{name: "uppercase before lowercase", args: args{input: []rune("bA")}, want: []rune("Ab")},
		{name: "duplicates", args: args{input: []rune("baba")}, want: []rune("aabb")},
		{name: "already sorted", args: args{input: []rune("abc")}, want: []rune("abc")},
		{name: "empty", args: args{input: []rune{}}, want: []rune{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bubbleSort(tt.args.input); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("bubbleSort() = %q, want %q", string(got), string(tt.want))
			}
		})
	}
}

func Test_initPositions(t *testing.T) {
	type args struct {
		input []rune
	}
	tests := []struct {
		name string
		args args
		want []int
	}{
		// Each character takes 9 lines, the file starts with an empty line, space is the first character
		{name: "space", args: args{input: []rune(" ")}, want: []int{2}},
		{name: "exclamation", args: args{input: []rune("!")}, want: []int{11}},
		{name: "A and a", args: args{input: []rune("Aa")}, want: []int{299, 587}},
		{name: "tilde is the last", args: args{input: []rune("~")}, want: []int{848}},
		{name: "empty", args: args{input: []rune{}}, want: []int{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := initPositions(tt.args.input); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("initPositions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_deleteLineBreaks(t *testing.T) {
	type args struct {
		input string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "one line break", args: args{input: "a\nb"}, want: "a b"},
		{name: "several line breaks", args: args{input: "\na\n\n"}, want: " a  "},
		{name: "no line breaks", args: args{input: "abc"}, want: "abc"},
		{name: "typed \\n is not touched", args: args{input: "a\\nb"}, want: "a\\nb"},
		{name: "empty", args: args{input: ""}, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deleteLineBreaks(tt.args.input); got != tt.want {
				t.Errorf("deleteLineBreaks() = %q, want %q", got, tt.want)
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
		{name: "real line break is kept", args: args{input: "a\nb"}, want: "a\nb"},
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

func Test_validateInput(t *testing.T) {
	type args struct {
		input string
	}
	tests := []struct {
		name    string
		args    args
		wantErr string // "" -> no error
	}{
		{name: "simple text", args: args{input: "Hello"}, wantErr: ""},
		{name: "line break is allowed", args: args{input: "a\nb"}, wantErr: ""},
		{name: "all printable ASCII", args: args{input: " !~"}, wantErr: ""},
		{name: "1000 characters", args: args{input: strings.Repeat("a", 1000)}, wantErr: ""},
		{name: "empty", args: args{input: ""}, wantErr: "Text is empty"},
		{name: "only spaces and line breaks", args: args{input: " \n "}, wantErr: "Text is empty"},
		{name: "1001 characters", args: args{input: strings.Repeat("a", 1001)}, wantErr: "Text is too long (max 1000 characters)"},
		{name: "cyrillic", args: args{input: "Привет"}, wantErr: "Invalid character"},
		{name: "tab", args: args{input: "a\tb"}, wantErr: "Invalid character"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInput(tt.args.input)
			got := ""
			if err != nil {
				got = err.Error()
			}
			if got != tt.wantErr {
				t.Errorf("validateInput() error = %q, want %q", got, tt.wantErr)
			}
		})
	}
}
