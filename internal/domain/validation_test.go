package domain

import "testing"

// TestIsValidMarketplace tests the IsValidMarketplace function
// only uppercase marketplaces are allowed
func TestIsValidMarketplace(t *testing.T) {
	tests := []struct {
		name        string
		marketplace string
		expected    bool
	}{
		{"valid US", "US", true},
		{"valid UK", "UK", true},
		{"valid DE", "DE", true},
		{"valid lowercase us", "us", true},
		{"valid mixed case Uk", "Uk", true},
		{"valid with spaces", " US ", true},
		{"invalid XX", "XX", false},
		{"invalid ZZ", "ZZ", false},
		{"empty string", "", false},
		{"only spaces", "   ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidMarketplace(tt.marketplace)
			if result != tt.expected {
				t.Errorf("IsValidMarketplace(%q) = %v, expected %v", tt.marketplace, result, tt.expected)
			}
		})
	}
}
