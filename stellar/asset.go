package stellar

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Asset is the normalised form of a Stellar asset, aligned with the
// gateway's validation rules (ADR 0003 amendment — business/stellar-quote
// and wallet/transaction endpoints). "XLM" and "native" are equivalent;
// both canonicalise to IsNative() == true.
type Asset struct {
	Code   string
	Issuer string
}

// ErrInvalidAsset is returned by [ParseAsset] when the input is malformed.
var ErrInvalidAsset = errors.New("stellar: invalid asset (expected XLM/native or CODE:ISSUER)")

// assetCodePattern mirrors the gateway regex: 1–12 alphanumerics.
var assetCodePattern = regexp.MustCompile(`^[A-Z0-9]{1,12}$`)

// ParseAsset parses a gateway-formatted asset string. "XLM" and "native"
// (case-insensitive) are both accepted; otherwise the input must be
// "CODE:ISSUER" with a 1-12 character code (upper-cased) and a valid G...
// issuer.
func ParseAsset(s string) (Asset, error) {
	trim := strings.TrimSpace(s)
	if trim == "" {
		return Asset{}, ErrInvalidAsset
	}
	lower := strings.ToLower(trim)
	if lower == "xlm" || lower == "native" {
		return Asset{Code: "XLM"}, nil
	}
	parts := strings.SplitN(trim, ":", 2)
	if len(parts) != 2 {
		return Asset{}, ErrInvalidAsset
	}
	code := strings.ToUpper(parts[0])
	issuer := parts[1]
	if !assetCodePattern.MatchString(code) {
		return Asset{}, fmt.Errorf("%w: code %q", ErrInvalidAsset, code)
	}
	if !publicKeyPattern.MatchString(issuer) {
		return Asset{}, fmt.Errorf("%w: issuer %q", ErrInvalidAsset, issuer)
	}
	return Asset{Code: code, Issuer: issuer}, nil
}

// MustAsset is the panic-on-error variant of [ParseAsset]. Convenient in
// tests and top-level main packages.
func MustAsset(s string) Asset {
	a, err := ParseAsset(s)
	if err != nil {
		panic(err)
	}
	return a
}

// IsNative reports whether this asset is native XLM.
func (a Asset) IsNative() bool { return a.Code == "XLM" && a.Issuer == "" }

// String returns the canonical wire representation: "XLM" for native,
// "CODE:ISSUER" otherwise.
func (a Asset) String() string {
	if a.IsNative() {
		return "XLM"
	}
	return a.Code + ":" + a.Issuer
}

// Equal reports whether two assets are identical after normalisation.
func (a Asset) Equal(other Asset) bool {
	if a.IsNative() && other.IsNative() {
		return true
	}
	return a.Code == other.Code && a.Issuer == other.Issuer
}
