package ai

import (
	"reflect"
	"testing"
)

func Test_validateVariationInput(t *testing.T) {
	type args struct {
		text string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateVariationInput(tt.args.text); (err != nil) != tt.wantErr {
				t.Errorf("validateVariationInput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_mockVariations(t *testing.T) {
	type args struct {
		text string
	}
	tests := []struct {
		name string
		args args
		want []Variation
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mockVariations(tt.args.text); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mockVariations() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseVariations(t *testing.T) {
	type args struct {
		answer string
	}
	tests := []struct {
		name    string
		args    args
		want    []Variation
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseVariations(tt.args.answer)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseVariations() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseVariations() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_validVariationText(t *testing.T) {
	type args struct {
		text string
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
			if got := validVariationText(tt.args.text); got != tt.want {
				t.Errorf("validVariationText() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_validVariationDescription(t *testing.T) {
	type args struct {
		description string
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
			if got := validVariationDescription(tt.args.description); got != tt.want {
				t.Errorf("validVariationDescription() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_validVariationBanner(t *testing.T) {
	type args struct {
		banner string
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
			if got := validVariationBanner(tt.args.banner); got != tt.want {
				t.Errorf("validVariationBanner() = %v, want %v", got, tt.want)
			}
		})
	}
}
