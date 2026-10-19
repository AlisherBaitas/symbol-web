package ai

import (
	"reflect"
	"testing"
)

func Test_mockSuggestions(t *testing.T) {
	type args struct {
		text string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mockSuggestions(tt.args.text); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mockSuggestions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_chatURL(t *testing.T) {
	type args struct {
		baseURL string
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
			if got := chatURL(tt.args.baseURL); got != tt.want {
				t.Errorf("chatURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseSuggestions(t *testing.T) {
	type args struct {
		answer string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseSuggestions(tt.args.answer); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseSuggestions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_printableASCII(t *testing.T) {
	type args struct {
		s string
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
			if got := printableASCII(tt.args.s); got != tt.want {
				t.Errorf("printableASCII() = %v, want %v", got, tt.want)
			}
		})
	}
}
