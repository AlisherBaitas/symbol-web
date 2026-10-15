package ai

import (
	"strings"
)

type stats struct {
    upper int
	lower int
	special int  
	newLines int
	length int 
	nonSpace int
}

func GetRecommend(input string) string {
	// To analyze the input word
	sts := analyzeInput([]rune(input))

	//specRatio := 0.0

	// To avoid division by zero
	if sts.nonSpace > 0 {
		//specRatio = float64(sts.special) / float64(sts.nonSpace)
	}

	






	

	return ""
}


func analyzeInput(input []rune) stats {
	// To convert new lines into '\n'
	inputString := strings.TrimSpace(convertLineBreaks(string(input)))

	// Get length after converting new lines
	inputLen := len(inputString)

	upCount, lowCount, specCount, newLineCount, nonSpaceCount := 0, 0, 0, 0, 0

	for _, val := range inputString {

		if val != ' ' && val != '\n' {
			nonSpaceCount++
		}

		if isUpperCase(val) {
			upCount++
		} else if isLowerCase(val) {
			lowCount++
		} else if isSpecial(val) {
			specCount++
		} else if hasNewLine(val) {
			newLineCount++
		}
	}

	return stats {
		upper: upCount,
		lower: lowCount,
		special: specCount,
		newLines: newLineCount,
		length: inputLen,
		nonSpace: nonSpaceCount,
	}
}

// Character Classification Functions
func isUpperCase(ch rune) bool {
	if ch >= 'A' && ch <= 'Z' {
		return true
	}
	return false
}

func isLowerCase(ch rune) bool {
	if ch >= 'a' && ch <= 'z' {
		return true
	}
	return false
}

func hasNewLine(ch rune) bool {
	if ch == '\n' {
		return true
	}
	return false
}

func isSpecial(ch rune) bool {
	return (ch >= '!' && ch <= '/') ||
		(ch >= ':' && ch <= '@') ||
		(ch >= '[' && ch <= '`') ||
		(ch >= '{' && ch <= '~')
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