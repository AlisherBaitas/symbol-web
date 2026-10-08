package ai

import (
)

var (
	upCount int
	lowCount int
	specCount int
)

func GetRecommend() {

}


func anayzeInput(input []rune) {
	// To convert new lines into '\n'
	inputString := convertLineBreaks(string(input))

	for _, val := range inputString {
		if isUpperCase(val) {
			upCount++
		} else if isLowerCase(val) {
			lowCount++
		} else if isSpecial(val) {
			specCount++
		}
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

func isSpecial(ch rune) bool {
	if firstSpecial(ch) {
		return true
	} else if secondSpecial(ch) {
		return true
	} else if thirdSpecial(ch) {
		return true
	} else if fourthSpecial(ch) {
		return true
	}
	return false
}


// Functions: to find special characters from ASCII Table
func firstSpecial(ch rune) bool {
	if ch >= '!' && ch <= '/' {
		return true
	}
	return false
}

func secondSpecial(ch rune) bool {
	if ch >= ':' && ch <= '@' {
		return true
	}
	return false
}

func thirdSpecial(ch rune) bool {
	if ch >= '[' && ch <= '`' {
		return true
	}
	return false
}

func fourthSpecial(ch rune) bool {
	if ch >= '{' && ch <= '~' {
		return true
	}
	return false
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