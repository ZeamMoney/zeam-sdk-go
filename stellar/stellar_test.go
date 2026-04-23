package stellar

import (
	"errors"
	"testing"
)

const (
	validPub  = "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN"
	validSeed = "SABCDEFGHIJKLMNOPQRSTUVWXYZ234567ABCDEFGHIJKLMNOPQRSTUVW"
)

func TestParsePublicKey(t *testing.T) {
	kp, err := ParsePublicKey(validPub)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if kp.PublicKey() != validPub {
		t.Errorf("PublicKey() = %q want %q", kp.PublicKey(), validPub)
	}
	if _, err := ParsePublicKey("not-a-key"); !errors.Is(err, ErrInvalidPublicKey) {
		t.Errorf("bad address: expected ErrInvalidPublicKey, got %v", err)
	}
}

func TestParseSeed(t *testing.T) {
	kp, err := ParseSeed(validSeed)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if !kp.CanSign() {
		t.Error("CanSign should be true for seed-bearing keypair")
	}
	kp.Erase()
	if kp.CanSign() {
		t.Error("Erase should drop the seed")
	}

	if _, err := ParseSeed("nope"); !errors.Is(err, ErrInvalidSeed) {
		t.Errorf("bad seed: expected ErrInvalidSeed, got %v", err)
	}
}

func TestParseAsset(t *testing.T) {
	cases := []struct {
		name   string
		in     string
		want   string
		native bool
		err    bool
	}{
		{"xlm upper", "XLM", "XLM", true, false},
		{"native lower", "native", "XLM", true, false},
		{"mixed case native", "Native", "XLM", true, false},
		{"usdc", "USDC:" + validPub, "USDC:" + validPub, false, false},
		{"lowercase code upcased", "usdc:" + validPub, "USDC:" + validPub, false, false},
		{"missing issuer", "USDC", "", false, true},
		{"bad issuer", "USDC:not-a-key", "", false, true},
		{"long code", "ASDFGHJKLQWE1:" + validPub, "", false, true}, // 13 chars
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, err := ParseAsset(c.in)
			if c.err {
				if err == nil {
					t.Fatalf("expected error, got asset %+v", a)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected: %v", err)
			}
			if a.String() != c.want {
				t.Errorf("String() = %q want %q", a.String(), c.want)
			}
			if a.IsNative() != c.native {
				t.Errorf("IsNative() = %v want %v", a.IsNative(), c.native)
			}
		})
	}
}

func TestAssetEqual(t *testing.T) {
	xlm1 := MustAsset("XLM")
	xlm2 := MustAsset("native")
	if !xlm1.Equal(xlm2) {
		t.Error("XLM and native should compare equal")
	}
	usdc1 := MustAsset("USDC:" + validPub)
	usdc2 := MustAsset("usdc:" + validPub)
	if !usdc1.Equal(usdc2) {
		t.Error("case-insensitive codes should compare equal")
	}
	if xlm1.Equal(usdc1) {
		t.Error("XLM and USDC must not be equal")
	}
}

func TestPlaceholderSignerRefusesEmpty(t *testing.T) {
	s := NewSigner(PublicNetworkPassphrase)
	kp, _ := ParseSeed(validSeed)
	if _, err := s.Sign("", kp); err == nil {
		t.Fatal("empty XDR should error")
	}
}
