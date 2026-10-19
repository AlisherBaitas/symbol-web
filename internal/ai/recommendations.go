package ai

import (
	"strings"
)

const (
	specRatioLimit = 0.3
	shortLen = 6
	longLen = 12
)

type Stats struct {
    upper int
	lower int
	special int  
	newLines int
	length int 
	nonSpace int
}

type Alternative struct {
	Banner string  `json:"banner"`
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}

type Recommendation struct {
	Recommended string `json:"recommended"`
	Reasoning string `json:"reasoning"`
	Alternatives []Alternative `json:"alternatives"`
}

func GetRecommend(input string) Recommendation {
	// To analyze the input word
	sts := analyzeInput([]rune(input))

	specRatio := 0.0

	// To avoid division by zero
	if sts.nonSpace > 0 {
		specRatio = float64(sts.special) / float64(sts.nonSpace)
	}

	if sts.newLines > 0 {
		return Recommendation{
			Recommended: "standard",
			Reasoning: "Multi-line text is easiest to read in the standard banner.",
			Alternatives: []Alternative{
				{
					Banner: "shadow",
					Score: 0.6,
					Reason: "Bold, but heavy across several lines",
				},
				{
					Banner: "thinkertoy",
					Score: 0.4,
					Reason: "Thin decorative lines are hard to read across lines",
				},
			},
		}

	} else if specRatio >= specRatioLimit {
		return Recommendation{
			Recommended: "thinkertoy",
			Reasoning: "Symbol-heavy text suits the playful, decorative style.",
			Alternatives: []Alternative{
				{
					Banner: "standard",
					Score: 0.6,
					Reason: "Clear and readable, less playful",
				},
				{
					Banner: "shadow",
					Score: 0.4,
					Reason: "Shadow makes symbols harder to tell apart",
				},
			},
		}

	} else if (sts.lower == 0 && sts.upper > 0) || sts.length <= shortLen {
		return Recommendation{
			Recommended: "shadow",
			Reasoning: "Short or uppercase text is bold and stands out.",
			Alternatives: []Alternative{
				{
					Banner: "standard",
					Score: 0.7,
					Reason: "Good alternative, slightly less impactful",
				},
				{
					Banner: "thinkertoy",
					Score: 0.4,
					Reason: "Too decorative for short, bold text",
				},
			},
		}
	} else if sts.length > longLen {
		return Recommendation{
			Recommended: "standard",
			Reasoning: "Longer text stays readable in the standard banner.",
			Alternatives: []Alternative{
				{
					Banner: "shadow",
					Score: 0.5,
					Reason: "Too heavy for long text",
				},
				{
					Banner: "thinkertoy",
					Score: 0.4,
					Reason: "Decorative style is hard to read at length",
				},
			},
		}
	}

	return Recommendation{
			Recommended: "thinkertoy",
			Reasoning: "Playful for short mixed-case text.",
			Alternatives: []Alternative{
				{
					Banner: "standard",
					Score: 0.7,
					Reason: "Safe and readable choice",
				},
				{
					Banner: "shadow",
					Score: 0.5,
					Reason: "Bold, but less fitting for mixed case",
				},
			},
	}
	
}


func analyzeInput(input []rune) Stats {
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

	return Stats {
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