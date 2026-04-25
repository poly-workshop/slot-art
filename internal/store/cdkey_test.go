package store

import "testing"

func TestCDKeyNormalizeHashAndMask(t *testing.T) {
	plain := "sa-abcd efgh-2345-6789"
	normalized := NormalizeCDKey(plain)
	if normalized != "SAABCDEFGH23456789" {
		t.Fatalf("NormalizeCDKey() = %q", normalized)
	}

	if HashCDKey(plain) != HashCDKey("SA-ABCD-EFGH-2345-6789") {
		t.Fatal("HashCDKey should ignore case, spaces and hyphens")
	}

	if got := MaskCDKey(plain); got != "SAAB****6789" {
		t.Fatalf("MaskCDKey() = %q", got)
	}
	if got := MaskCDKey("1234"); got != "****" {
		t.Fatalf("MaskCDKey(short) = %q", got)
	}
}
