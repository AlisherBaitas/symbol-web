package handlers

import (
	"net/http"
	"testing"
)

func TestHandleSuggestion(t *testing.T) {
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			HandleSuggestion(tt.args.w, tt.args.r)
		})
	}
}

func TestHandleBannerRec(t *testing.T) {
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			HandleBannerRec(tt.args.w, tt.args.r)
		})
	}
}

func TestHandleVariation(t *testing.T) {
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			HandleVariation(tt.args.w, tt.args.r)
		})
	}
}

func Test_readTextRequest(t *testing.T) {
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name     string
		args     args
		wantText string
		wantOk   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotText, gotOk := readTextRequest(tt.args.w, tt.args.r)
			if gotText != tt.wantText {
				t.Errorf("readTextRequest() gotText = %v, want %v", gotText, tt.wantText)
			}
			if gotOk != tt.wantOk {
				t.Errorf("readTextRequest() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func Test_writeVariationError(t *testing.T) {
	type args struct {
		w       http.ResponseWriter
		status  int
		message string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeVariationError(tt.args.w, tt.args.status, tt.args.message)
		})
	}
}

func Test_writeVariationJSON(t *testing.T) {
	type args struct {
		w       http.ResponseWriter
		status  int
		payload any
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeVariationJSON(tt.args.w, tt.args.status, tt.args.payload)
		})
	}
}

func Test_validateArtText(t *testing.T) {
	type args struct {
		input string
	}
	tests := []struct {
		name  string
		args  args
		want  bool
		want1 string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := validateArtText(tt.args.input)
			if got != tt.want {
				t.Errorf("validateArtText() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("validateArtText() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
