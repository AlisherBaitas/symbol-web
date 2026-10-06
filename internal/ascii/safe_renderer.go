package ascii

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"
)

const MaxTextLength = 1000

var (
	ErrInvalidText   = errors.New("используйте от 1 до 1000 печатных ASCII-символов и переносы строк")
	ErrInvalidBanner = errors.New("выберите standard, shadow или thinkertoy")
	ErrCorruptBanner = errors.New("файл баннера повреждён")
)

// Normalize проверяет текст до обращения к файлам и сохраняет пробелы пользователя.
func Normalize(text string) (string, error) {
	if len(text) == 0 || len(text) > MaxTextLength {
		return "", ErrInvalidText
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, `\n`, "\n")
	for _, ch := range text {
		if ch != '\n' && (ch < 32 || ch > 126) {
			return "", ErrInvalidText
		}
	}
	return text, nil
}

func BannerName(name string) (string, error) {
	name = strings.TrimSuffix(name, ".txt")
	switch name {
	case "standard", "shadow", "thinkertoy":
		return name, nil
	default:
		return "", ErrInvalidBanner
	}
}

// Render дополняет исходный пакет: локальный буфер, проверка файла и ошибки вместо os.Exit.
func Render(files fs.FS, text, name string) (string, error) {
	text, err := Normalize(text)
	if err != nil {
		return "", err
	}
	name, err = BannerName(name)
	if err != nil {
		return "", err
	}
	f, err := files.Open(name + ".txt")
	if err != nil {
		return "", fmt.Errorf("баннер %s: %w", name, err)
	}
	defer f.Close()
	const maxBannerSize = 128 * 1024
	b, err := io.ReadAll(io.LimitReader(f, maxBannerSize+1))
	if err != nil {
		return "", fmt.Errorf("чтение баннера: %w", err)
	}
	if len(b) > maxBannerSize {
		return "", ErrCorruptBanner
	}
	lines := strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
	if len(lines) == 856 && lines[855] == "" {
		lines = lines[:855]
	}
	if len(lines) != 855 {
		return "", ErrCorruptBanner
	}
	for i, line := range lines {
		if i%9 == 0 && line != "" {
			return "", ErrCorruptBanner
		}
		if len(line) > 64 {
			return "", ErrCorruptBanner
		}
		for _, ch := range line {
			if ch < 32 || ch > 126 {
				return "", ErrCorruptBanner
			}
		}
	}
	if strings.Trim(text, "\n") == "" {
		return text, nil
	}
	var result strings.Builder
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			result.WriteByte('\n')
			continue
		}
		// Формула позиций остаётся общей с исходным генератором команды.
		positions := initPositions([]rune(line))
		for row := 0; row < 8; row++ {
			for _, position := range positions {
				result.WriteString(lines[position-1+row])
			}
			result.WriteByte('\n')
		}
	}
	return result.String(), nil
}
