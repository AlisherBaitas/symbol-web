package ascii

import (
	"bufio"
	"fmt"
	"os"
)

var artResult string

func Generate(input string, fileName string) string {

	// Return map with chars and symbols
	symStore := banner([]rune(input), fileName)

	// Call to save the result of symbol art
	printer(symStore, input)

	resArt := artResult

	// Clean the global var
	artResult = ""

	return resArt
}

func printer(lookUp map[rune][]string, input string) {

	if len(input) == 0 {
		return
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
				printRow(lookUp, word)
			}

			// If new line at start or at the end of the string
			if i == 0 || i == len(input)-1 {
				artResult += "\n"
			}

			// If between runes more than one '\n'
			j := i
			for j+1 < len(input) && input[j+1] == '\n' {
				artResult += "\n"
				j++
			}

		} else if input[i] != '\n' {
			temp = append(temp, rune(input[i]))
		}
	}

	if len(temp) != 0 {
		word := string(temp)
		temp = temp[:0]
		printRow(lookUp, word)
	}
}

// Function: to print symbols
func printRow(lookUp map[rune][]string, input string) {
	lines := 0
	inputLen := len(input)

	for lines < 8 {
		currSymbol := 0

		for currSymbol < inputLen {
			currInput := lookUp[rune(input[currSymbol])]

			for _, val := range currInput[lines] {
				artResult += string(val)

			}
			currSymbol++
		}

		artResult += "\n"
		lines++
	}
}


func banner(input []rune, fileName string) map[rune][]string {
	input = []rune(deleteLineBreaks(string(input)))

	if len(input) == 0 {
		return map[rune][]string{}
	}

	// Initialize the storage
	lookUp := make(map[rune][]string)

	// Open the file (read-only)
	file, err := os.Open(fileName)

	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
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

	if err := scanner.Err(); err != nil {
		fmt.Printf("Scanning error: %v\n", err)
	}

	return lookUp
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

