package ai

import (
	"reflect"
	"testing"
)

func TestGetRecommend(t *testing.T) {
	type args struct {
		input string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetRecommend(tt.args.input); got != tt.want {
				t.Errorf("GetRecommend() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_analyzeInput(t *testing.T) {
	type args struct {
		input []rune
	}
	tests := []struct {
		name string
		args args
		want stats
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := analyzeInput(tt.args.input); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("analyzeInput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isUpperCase(t *testing.T) {
	type args struct {
		ch rune
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUpperCase(tt.args.ch); got != tt.want {
				t.Errorf("isUpperCase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isLowerCase(t *testing.T) {
	type args struct {
		ch rune
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLowerCase(tt.args.ch); got != tt.want {
				t.Errorf("isLowerCase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_hasNewLine(t *testing.T) {
	type args struct {
		ch rune
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasNewLine(tt.args.ch); got != tt.want {
				t.Errorf("hasNewLine() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isSpecial(t *testing.T) {
	type args struct {
		ch rune
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSpecial(tt.args.ch); got != tt.want {
				t.Errorf("isSpecial() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_convertLineBreaks(t *testing.T) {
	type args struct {
		input string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertLineBreaks(tt.args.input); got != tt.want {
				t.Errorf("convertLineBreaks() = %v, want %v", got, tt.want)
			}
		})
	}
}
