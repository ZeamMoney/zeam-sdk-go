package stellar

import (
	"errors"
	"regexp"
)

// Stellar ed25519 keypair address format: 56 characters, base32 alphabet
// (A-Z, 2-7), seeds start with "S", public keys start with "G".
var (
	seedPattern      = regexp.MustCompile(`^S[A-Z2-7]{55}$`)
	publicKeyPattern = regexp.MustCompile(`^G[A-Z2-7]{55}$`)
)

// ErrInvalidSeed is returned when a value does not look like a Stellar
// secret seed.
var ErrInvalidSeed = errors.New("stellar: invalid seed (expected S... 56 chars)")

// ErrInvalidPublicKey is returned when a value does not look like a
// Stellar public key.
var ErrInvalidPublicKey = errors.New("stellar: invalid public key (expected G... 56 chars)")

// Keypair carries a Stellar account address. It optionally holds the
// secret seed for signing; callers SHOULD call [Keypair.Erase] when the
// keypair is no longer needed to wipe the seed from memory.
//
// The underlying Stellar SDK types are intentionally not re-exported so
// the wrapper can evolve independently.
type Keypair struct {
	publicKey string
	seed      []byte
}

// ParsePublicKey validates and returns a read-only keypair carrying just
// the public key.
func ParsePublicKey(address string) (*Keypair, error) {
	if !publicKeyPattern.MatchString(address) {
		return nil, ErrInvalidPublicKey
	}
	return &Keypair{publicKey: address}, nil
}

// ParseSeed validates and returns a keypair carrying both the seed and
// the derived public key. The seed is stored in a []byte the caller can
// zero via [Keypair.Erase].
//
// Note: the current scaffold performs structural validation only. Once
// the upstream Stellar SDK is wired in, this function also verifies that
// the seed's checksum is valid and derives the public key from it. Until
// then, callers must supply both the seed and the public key via
// [NewKeypair] if they need to sign.
func ParseSeed(seed string) (*Keypair, error) {
	if !seedPattern.MatchString(seed) {
		return nil, ErrInvalidSeed
	}
	return &Keypair{seed: []byte(seed)}, nil
}

// NewKeypair constructs a keypair from a seed and its matching public key.
// Both are structurally validated. Intended for SEP-10 flows where the
// public key is already known.
func NewKeypair(seed, publicKey string) (*Keypair, error) {
	if !seedPattern.MatchString(seed) {
		return nil, ErrInvalidSeed
	}
	if !publicKeyPattern.MatchString(publicKey) {
		return nil, ErrInvalidPublicKey
	}
	return &Keypair{
		publicKey: publicKey,
		seed:      []byte(seed),
	}, nil
}

// PublicKey returns the G... address.
func (k *Keypair) PublicKey() string {
	if k == nil {
		return ""
	}
	return k.publicKey
}

// CanSign reports whether the keypair holds a seed.
func (k *Keypair) CanSign() bool { return k != nil && len(k.seed) > 0 }

// Erase zeros the seed bytes. After Erase returns, [Keypair.CanSign]
// returns false. Safe to call on a nil receiver.
func (k *Keypair) Erase() {
	if k == nil {
		return
	}
	for i := range k.seed {
		k.seed[i] = 0
	}
	k.seed = nil
}
