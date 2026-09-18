package ascii

import (
	"bufio"
	"fmt"
	"os"
)

// Store character with its symbols
var LookUp map[rune][]string

func Banner(input []rune, fileName string) map[rune][]string {
	input = []rune(deleteLineBreaks(string(input)))

	if len(input) == 0 {
		return map[rune][]string{}
	}

	// Initialize the storage
	LookUp = make(map[rune][]string)

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
	input = BubbleSort(input)

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
				LookUp[target] = tempCopy

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

	return LookUp
}

// Function: to sort characters
func BubbleSort(input []rune) []rune {

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