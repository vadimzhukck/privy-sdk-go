package aptos

import (
	"testing"

	privy "github.com/vadimzhukck/privy-sdk-go"
)

func TestNewHelper(t *testing.T) {
	client := privy.NewClient("app-id", "app-secret")
	h := NewHelper(client)

	if h.client != client {
		t.Error("Expected client to be set")
	}

	if h.aptosClient == nil {
		t.Log("Note: Aptos client may be nil if mainnet is unreachable")
	}
}

func TestDecodeHexSignature(t *testing.T) {
	tests := []struct {
		input    string
		expected []byte
	}{
		{"0xabcdef", []byte{0xab, 0xcd, 0xef}},
		{"abcdef", []byte{0xab, 0xcd, 0xef}},
		{"0x00", []byte{0x00}},
	}

	for _, tt := range tests {
		b, err := decodeHexSignature(tt.input)
		if err != nil {
			t.Fatalf("Failed to decode %s: %v", tt.input, err)
		}
		if len(b) != len(tt.expected) {
			t.Errorf("Expected %d bytes for %s, got %d", len(tt.expected), tt.input, len(b))
		}
	}
}

func TestParseAddress(t *testing.T) {
	_, err := parseAddress("0x1")
	if err != nil {
		t.Errorf("Expected valid address parsing, got error: %v", err)
	}
}
