package handlers

import (
	"net/http"
	"testing"
)

func TestHandleRoot(t *testing.T) {
	type args struct {
		w   http.ResponseWriter
		in1 *http.Request
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			HandleRoot(tt.args.w, tt.args.in1)
		})
	}
}

func TestHandleSymbolArt(t *testing.T) {
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
			HandleSymbolArt(tt.args.w, tt.args.r)
		})
	}
}
