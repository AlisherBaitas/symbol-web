package ascii

import (
	"errors"
	"io/fs"
	"os"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
)

func visible(value string) string {
	lines := strings.Split(value, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " ")
	}
	return strings.Trim(strings.Join(lines, "\n"), "\n")
}

func TestAuditExamples(t *testing.T) {
	for _, tc := range []struct{ name, input, golden string }{{"standard", "{123}\n<Hello> (World)!", "standard-audit.txt"}, {"thinkertoy", "123 T/fs#R", "thinkertoy-audit.txt"}} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Render(os.DirFS("../.."), tc.input, tc.name)
			if err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile("testdata/" + tc.golden)
			if err != nil {
				t.Fatal(err)
			}
			if visible(got) != visible(string(want)) {
				t.Fatalf("рисунок не совпал с аудитом\ngot:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

func TestRenderEveryCharacterAndNewlines(t *testing.T) {
	files := os.DirFS("../..")
	var printable strings.Builder
	for ch := 32; ch <= 126; ch++ {
		printable.WriteByte(byte(ch))
	}
	for _, banner := range []string{"standard", "shadow", "thinkertoy"} {
		got, err := Render(files, printable.String(), banner)
		if err != nil || strings.Count(got, "\n") != 8 {
			t.Fatalf("%s: %v", banner, err)
		}
		one, _ := Render(files, "A", banner)
		for input, want := range map[string]string{"A\nA": one + one, "A\n\nA": one + "\n" + one, "A\r\nA": one + one, `A\nA`: one + one, "\nA": "\n" + one, "A\n": one + "\n", "\n\n": "\n\n"} {
			got, err := Render(files, input, banner+".txt")
			if err != nil || got != want {
				t.Errorf("%s %q: %v", banner, input, err)
			}
		}
	}
}

func TestValidationAndFiles(t *testing.T) {
	for _, input := range []string{"", "Привет", "a\tb", "a\rb", "a\x00b", "😀", "a\x7fb", strings.Repeat("a", 1001), string([]byte{0xff})} {
		if _, err := Normalize(input); !errors.Is(err, ErrInvalidText) {
			t.Errorf("принят недопустимый ввод %q", input)
		}
	}
	if _, err := Normalize(strings.Repeat("a", 1000)); err != nil {
		t.Fatal(err)
	}
	for _, banner := range []string{"", "../standard", "STANDARD", "/standard", "standard.txt.txt", "../../etc/passwd"} {
		if _, err := Render(fstest.MapFS{}, "hi", banner); !errors.Is(err, ErrInvalidBanner) {
			t.Errorf("принят баннер %q", banner)
		}
	}
	if _, err := Render(fstest.MapFS{}, "hi", "standard"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
	for _, bad := range []string{"", "broken", strings.Repeat("x", 129*1024), strings.Repeat("\n", 855) + "extra"} {
		if _, err := Render(fstest.MapFS{"standard.txt": {Data: []byte(bad)}}, "hi", "standard"); !errors.Is(err, ErrCorruptBanner) {
			t.Errorf("повреждение: %v", err)
		}
	}
	valid, err := os.ReadFile("../../standard.txt")
	if err != nil {
		t.Fatal(err)
	}
	crlf := strings.ReplaceAll(string(valid), "\n", "\r\n")
	if _, err := Render(fstest.MapFS{"standard.txt": {Data: []byte(crlf)}}, "hi", "standard"); err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentRender(t *testing.T) {
	files := os.DirFS("../..")
	var wg sync.WaitGroup
	for i := 0; i < 80; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			text := strings.Repeat("A", i%20+1)
			got, err := Render(files, text, "shadow")
			want, _ := Render(files, text, "shadow")
			if err != nil || got != want {
				t.Errorf("конкурентный запрос: %v", err)
			}
		}(i)
	}
	wg.Wait()
}

func FuzzRender(f *testing.F) {
	for _, s := range []string{"Hello", "\n", `A\n\nB`, "Привет", "\x00", strings.Repeat("a", 1000)} {
		f.Add(s)
	}
	files := os.DirFS("../..")
	f.Fuzz(func(t *testing.T, input string) {
		out, err := Render(files, input, "standard")
		_, validation := Normalize(input)
		if validation != nil && err == nil {
			t.Fatal("принят недопустимый ввод")
		}
		if validation == nil && err != nil {
			t.Fatal(err)
		}
		if len(out) > 520000 {
			t.Fatal("неограниченный вывод")
		}
	})
}
