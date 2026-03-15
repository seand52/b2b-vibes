package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClient_IsLinked(t *testing.T) {
	tests := []struct {
		name    string
		auth0ID *string
		want    bool
	}{
		{
			name:    "nil auth0_id",
			auth0ID: nil,
			want:    false,
		},
		{
			name:    "empty auth0_id",
			auth0ID: ptr(""),
			want:    false,
		},
		{
			name:    "valid auth0_id",
			auth0ID: ptr("auth0|12345"),
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{Auth0ID: tt.auth0ID}
			assert.Equal(t, tt.want, c.IsLinked())
		})
	}
}

func TestClient_IsCompany(t *testing.T) {
	tests := []struct {
		name    string
		vatType VATType
		want    bool
	}{
		{
			name:    "CIF is company",
			vatType: VATTypeCIF,
			want:    true,
		},
		{
			name:    "NIF is not company",
			vatType: VATTypeNIF,
			want:    false,
		},
		{
			name:    "empty is not company",
			vatType: "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{VATType: tt.vatType}
			assert.Equal(t, tt.want, c.IsCompany())
		})
	}
}

func ptr(s string) *string {
	return &s
}
