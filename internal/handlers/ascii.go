package handlers

import (
	"symbol-web/internal/ascii"
)

var artResult string

func Generate(input string, fileName string) string {

	// Return map with chars and symbols
	symStore := ascii.Banner([]rune(input), fileName)

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
