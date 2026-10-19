package ai

import (
	"context"
	"net/http"
	"reflect"
	"testing"
)

func TestNewAIClient(t *testing.T) {
	tests := []struct {
		name string
		want *AIClient
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewAIClient(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewAIClient() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAIClient_GetSuggestions(t *testing.T) {
	type fields struct {
		baseURL string
		model   string
		client  *http.Client
	}
	type args struct {
		ctx  context.Context
		text string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &AIClient{
				baseURL: tt.fields.baseURL,
				model:   tt.fields.model,
				client:  tt.fields.client,
			}
			got, err := c.GetSuggestions(tt.args.ctx, tt.args.text)
			if (err != nil) != tt.wantErr {
				t.Fatalf("AIClient.GetSuggestions() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AIClient.GetSuggestions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAIClient_chatSuggest(t *testing.T) {
	type fields struct {
		baseURL string
		model   string
		client  *http.Client
	}
	type args struct {
		ctx    context.Context
		prompt string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &AIClient{
				baseURL: tt.fields.baseURL,
				model:   tt.fields.model,
				client:  tt.fields.client,
			}
			got, err := c.chatSuggest(tt.args.ctx, tt.args.prompt)
			if (err != nil) != tt.wantErr {
				t.Fatalf("AIClient.chatSuggest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("AIClient.chatSuggest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAIClient_GetVariations(t *testing.T) {
	type fields struct {
		baseURL string
		model   string
		client  *http.Client
	}
	type args struct {
		ctx  context.Context
		text string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []Variation
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &AIClient{
				baseURL: tt.fields.baseURL,
				model:   tt.fields.model,
				client:  tt.fields.client,
			}
			got, err := c.GetVariations(tt.args.ctx, tt.args.text)
			if (err != nil) != tt.wantErr {
				t.Fatalf("AIClient.GetVariations() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AIClient.GetVariations() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAIClient_requestVariations(t *testing.T) {
	type fields struct {
		baseURL string
		model   string
		client  *http.Client
	}
	type args struct {
		ctx    context.Context
		prompt string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &AIClient{
				baseURL: tt.fields.baseURL,
				model:   tt.fields.model,
				client:  tt.fields.client,
			}
			got, err := c.requestVariations(tt.args.ctx, tt.args.prompt)
			if (err != nil) != tt.wantErr {
				t.Fatalf("AIClient.requestVariations() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("AIClient.requestVariations() = %v, want %v", got, tt.want)
			}
		})
	}
}
