package main

import (
	"reflect"
	"testing"
)

func TestCorsAllowedOrigins(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want []string
	}{
		{"unset falls back to default", "", []string{"http://localhost:5173"}},
		{"single origin", "http://localhost:3000", []string{"http://localhost:3000"}},
		{"multiple origins without spaces", "http://a.example,http://b.example", []string{"http://a.example", "http://b.example"}},
		{"multiple origins with spaces after comma", "http://a.example, http://b.example", []string{"http://a.example", "http://b.example"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CORS_ALLOWED_ORIGINS", tt.env)
			got := corsAllowedOrigins()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("corsAllowedOrigins() = %v, want %v", got, tt.want)
			}
		})
	}
}
