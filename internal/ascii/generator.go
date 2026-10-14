package ascii

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

var (
	// [Validation] Ошибки вместо os.Exit и паники: обработчик превращает их в 400/404/500.
	// ErrUnsupportedChar - в тексте есть символ, которого нет в шрифте (кириллица, Tab и т.п.)
	ErrUnsupportedChar = errors.New("text contains a character that is not in the banner")
	// ErrBannerNotFound - файл баннера не найден на диске
	ErrBannerNotFound = errors.New("banner file not found")
	// ErrBannerCorrupted - файл баннера не читается или в нём не хватает строк
	ErrBannerCorrupted = errors.New("banner file is corrupted")
)

// [Validation] Generate возвращает ошибку вместо падения сервера.
// Результат собирается в локальном strings.Builder, а не в глобальной переменной:
// раньше одновременные запросы и упавшие запросы портили ответы друг другу.
func Generate(input string, fileName string) (string, error) {

	// Return map with chars and symbols
	symStore, err := banner([]rune(input), fileName)
	if err != nil {
		return "", err
	}

	var art strings.Builder

	// Call to save the result of symbol art
	if err := printer(&art, symStore, input); err != nil {
		return "", err
	}

	return art.String(), nil
}

// [Validation] printer пишет в переданный art вместо глобальной artResult
// и возвращает ошибку от printRow, а не продолжает с неполными данными.
func printer(art *strings.Builder, lookUp map[rune][]string, input string) error {

	if len(input) == 0 {
		return nil
	}

	// To convert new lines into '\n'
	input = convertLineBreaks(input)

	var temp []rune

	for i := 0; i < len(input); i++ {

		if input[i] == '\n' {

			// Print symbols before new line
			if len(temp) != 0 {
				word := string(temp)
				temp = temp[:0]
				if err := printRow(art, lookUp, word); err != nil {
					return err
				}
			}

			// If new line at start or at the end of the string
			// [Validation] Было artResult += "\n" (глобальная переменная).
			if i == 0 || i == len(input)-1 {
				art.WriteString("\n")
			}

			// If between runes more than one '\n'
			j := i
			for j+1 < len(input) && input[j+1] == '\n' {
				art.WriteString("\n")
				j++
			}

		} else if input[i] != '\n' {
			temp = append(temp, rune(input[i]))
		}
	}

	if len(temp) != 0 {
		word := string(temp)
		temp = temp[:0]
		if err := printRow(art, lookUp, word); err != nil {
			return err
		}
	}
	return nil
}

// Function: to print symbols
func printRow(art *strings.Builder, lookUp map[rune][]string, input string) error {
	// [Validation] Раньше символ без картинки в шрифте давал панику
	// "index out of range" на currInput[lines]. Проверяем все символы заранее.
	for i := 0; i < len(input); i++ {
		if len(lookUp[rune(input[i])]) != 8 {
			return fmt.Errorf("%w: %q", ErrUnsupportedChar, input[i])
		}
	}

	lines := 0
	inputLen := len(input)

	for lines < 8 {
		currSymbol := 0

		for currSymbol < inputLen {
			currInput := lookUp[rune(input[currSymbol])]

			// [Validation] Было посимвольное artResult += string(val):
			// глобальная переменная и квадратичная склейка строк.
			art.WriteString(currInput[lines])
			currSymbol++
		}

		art.WriteString("\n")
		lines++
	}
	return nil
}


// [Validation] banner теперь возвращает ошибку вместо os.Exit и печати в консоль.
func banner(input []rune, fileName string) (map[rune][]string, error) {
	input = []rune(deleteLineBreaks(string(input)))

	if len(input) == 0 {
		return map[rune][]string{}, nil
	}

	// Initialize the storage
	lookUp := make(map[rune][]string)

	// Open the file (read-only)
	file, err := os.Open(fileName)

	// [Validation] Раньше здесь был os.Exit(1): один запрос с неверным баннером
	// завершал весь сервер. Теперь возвращаем ошибку.
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrBannerNotFound, fileName)
		}
		return nil, fmt.Errorf("%w: %v", ErrBannerCorrupted, err)
	}

	defer file.Close()

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

	// [Validation] Ошибка чтения (например, папка вместо файла) больше не игнорируется.
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBannerCorrupted, err)
	}

	return lookUp, nil
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

