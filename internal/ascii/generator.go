package ascii

import (
	"bufio"
	"errors"
	"os"
	"strings"
	"unicode/utf8"
)

var ErrBannerNotFound = errors.New("Banner file not found")

// [Fix] Ошибка выбора баннера: имя не из списка -> 400 (раньше любое значение
// из формы открывалось как путь к файлу и давало 404 или даже чтение go.mod).
var ErrInvalidBanner = errors.New("Invalid banner: choose standard, shadow or thinkertoy")

// [Fix] Файл баннера найден, но в нём нет нужного символа (обрезанный/чужой файл).
// Раньше printRow падал с паникой "index out of range".
var ErrBannerCorrupted = errors.New("Banner file is corrupted")

// [Fix] Белый список баннеров. Принимаем имя с .txt и без него.
var bannerFiles = map[string]string{
	"standard":       "standard.txt",
	"standard.txt":   "standard.txt",
	"shadow":         "shadow.txt",
	"shadow.txt":     "shadow.txt",
	"thinkertoy":     "thinkertoy.txt",
	"thinkertoy.txt": "thinkertoy.txt",
}

// [Fix] BannerFile переводит выбор пользователя в имя файла из белого списка.
func BannerFile(name string) (string, error) {
	file, ok := bannerFiles[name]
	if !ok {
		return "", ErrInvalidBanner
	}
	return file, nil
}

// [Fix] Удалена глобальная переменная artResult: все запросы писали в неё
// одновременно (DATA RACE, 8 из 80 ответов приходили с чужим артом).
// Теперь результат собирается в локальном strings.Builder каждого вызова.

func Generate(input string, fileName string) (string, error) {

	// [Fix] Браузер отправляет перенос строки из <textarea> как "\r\n".
	// Приводим к "\n", иначе '\r' считался недопустимым символом (400).
	input = NormalizeLineBreaks(input)

	// Check for any errors
	if validErr := ValidateInput(input); validErr != nil {
		return "", validErr
	}

	// Return map with chars and symbols
	symStore, bannerErr := banner([]rune(input), fileName)

	if bannerErr != nil {
		return "", bannerErr
	}

	// [Fix] Проверяем, что для каждого символа прочитаны все 8 строк.
	for _, r := range deleteLineBreaks(convertLineBreaks(input)) {
		if len(symStore[r]) != 8 {
			return "", ErrBannerCorrupted
		}
	}

	// Call to save the result of symbol art
	return printer(symStore, input), nil
}

// [Fix] NormalizeLineBreaks заменяет переносы Windows/браузера "\r\n" на "\n".
func NormalizeLineBreaks(input string) string {
	return strings.ReplaceAll(input, "\r\n", "\n")
}

func banner(input []rune, fileName string) (map[rune][]string, error) {

	// To convert new lines into '\n'
	input = []rune(convertLineBreaks(string(input)))

	// To delete '\n' to count chars
	input = []rune(deleteLineBreaks(string(input)))

	// Initialize the storage
	lookUp := make(map[rune][]string)

	// Open the file (read-only)
	file, err := os.Open(fileName)

	if err != nil {
		return nil, ErrBannerNotFound // 404
	}

	defer file.Close()

	// [Fix] Пустой ввод: раньше input[0] ниже вызывал панику.
	if len(input) == 0 {
		return lookUp, nil
	}

	// Read data line by line
	scanner := bufio.NewScanner(file)

	// Sort characters by ASCII code (ascending order)
	input = bubbleSort(input)

	// Start position of every runes
	startPos := initPositions(input)

	charCount := 0

	// Rune to search
	target := input[charCount]

	// Position of rune to search
	targetPos := startPos[charCount]

	// Condition for finding start and end of the rune
	symbol := false
	lineNumber := 1

	// Temporary storage of strings of symbol
	var temp []string

	for scanner.Scan() {

		// Find start position of the target (character)
		if lineNumber == targetPos || symbol == true {

			symbol = true
			temp = append(temp, scanner.Text())

			if len(temp) == 8 {

				// Copy temp
				tempCopy := make([]string, len(temp))
				copy(tempCopy, temp)

				// Save the symbol
				lookUp[target] = tempCopy

				// Clean the buffer
				temp = temp[:0]

				if charCount < len(input)-1 {
					charCount++

					// Skip duplicates
					for charCount < len(input) && startPos[charCount] == startPos[charCount-1] {
						charCount++
					}

					if charCount < len(input) {
						targetPos = startPos[charCount]
						target = input[charCount]
					}
				}
				symbol = false
			}
		}
		lineNumber++
	}

	if err := scanner.Err(); err != nil {
		return nil, errors.New("Banner file scanning error")
	}

	return lookUp, nil
}

// [Fix] printer возвращает результат вместо записи в глобальную переменную.
func printer(lookUp map[rune][]string, input string) string {

	// To convert new lines into '\n'
	input = convertLineBreaks(input)

	var result strings.Builder
	var temp []rune

	for i := 0; i < len(input); i++ {

		if input[i] == '\n' {

			// Print symbols before new line
			if len(temp) != 0 {
				word := string(temp)
				temp = temp[:0]
				printRow(&result, lookUp, word)
			}

			// If new line at start or at the end of the string
			if i == 0 || i == len(input)-1 {
				result.WriteString("\n")
			}

			// If between runes more than one '\n'
			j := i
			for j+1 < len(input) && input[j+1] == '\n' {
				result.WriteString("\n")
				j++
			}

		} else if input[i] != '\n' {
			temp = append(temp, rune(input[i]))
		}
	}

	if len(temp) != 0 {
		word := string(temp)
		temp = temp[:0]
		printRow(&result, lookUp, word)
	}

	return result.String()
}

// Function: to print symbols
// [Fix] Пишет в переданный strings.Builder, а не в глобальную переменную.
func printRow(result *strings.Builder, lookUp map[rune][]string, input string) {

	lines := 0
	inputLen := len(input)

	for lines < 8 {
		currSymbol := 0

		for currSymbol < inputLen {
			currInput := lookUp[rune(input[currSymbol])]

			// [Fix] Нет строки символа -> пропускаем, а не паникуем.
			if lines < len(currInput) {
				result.WriteString(currInput[lines])
			}
			currSymbol++
		}

		result.WriteString("\n")
		lines++
	}
}

// Function: to sort characters
func bubbleSort(input []rune) []rune {

	var temp rune

	for i := 0; i < len(input); i++ {
		for j := i + 1; j < len(input); j++ {
			if input[i] > input[j] {
				temp = input[i]
				input[i] = input[j]
				input[j] = temp
			}
		}
	}

	return input
}

// Function: to find position of every runes
func initPositions(input []rune) []int {

	startPos := make([]int, len(input))

	for i := 0; i < len(input); i++ {
		// Formula for finding starting position of every runes
		startPos[i] = ((int(input[i]) - 31) * 9) - 7
	}

	return startPos
}

// Function: to remove '\n' (new line)
func deleteLineBreaks(input string) string {

	result := make([]rune, 0, len(input))

	for _, v := range input {
		if v != '\n' {
			result = append(result, v)
		} else {
			result = append(result, ' ')
		}
	}

	return string(result)
}

// Function: to convert '\n' to new line
func convertLineBreaks(input string) string {

	temp := make([]rune, 0, len(input))

	for i := 0; i < len(input); i++ {
		if input[i] == '\\' && i+1 < len(input) && input[i+1] == 'n' {
			temp = append(temp, '\n')
			i++
		} else {
			temp = append(temp, rune(input[i]))
		}
	}

	return string(temp)
}

// Function: to check the correctness of the input
func ValidateInput(input string) error {

	// Check for empty input
	if strings.TrimSpace(input) == "" {
		return errors.New("Text is empty")
	}

	// Check for input exceeding
	if utf8.RuneCountInString(input) > 1000 {
		return errors.New("Text is too long (max 1000 characters)")
	}

	// Check for printable characters
	for _, r := range input {
		if r == '\n' {
			continue
		}

		if r < ' ' || r > '~' {
			return errors.New("Invalid character")
		}
	}

	return nil
}
