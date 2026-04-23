package stellar

import "errors"

// ChallengeSigner signs a SEP-10 challenge XDR with a Keypair's seed.
// Implementations delegate to the upstream Stellar SDK; the SDK wraps the
// upstream surface so consumers depend only on this interface.
type ChallengeSigner interface {
	// Sign signs the base64-encoded SEP-10 challenge XDR with the provided
	// keypair's seed and returns the signed XDR. The keypair must satisfy
	// [Keypair.CanSign].
	Sign(xdr string, kp *Keypair) (string, error)
}

// NewSigner returns the default SEP-10 signer for the given network
// passphrase. Pass [PublicNetworkPassphrase] for production use.
func NewSigner(network string) ChallengeSigner {
	return &placeholderSigner{network: network}
}

// placeholderSigner is the scaffold implementation. Once the upstream
// stellar SDK is wired in, this type will be replaced by a real ed25519
// signer that decodes the XDR, signs the hash with the seed, and
// re-encodes the signed envelope.
type placeholderSigner struct{ network string }

// Sign implements [ChallengeSigner]. The scaffold variant returns an
// explicit error so callers cannot accidentally ship unsigned challenges;
// Phase 1 of the implementation plan wires the upstream keypair into this
// method.
func (s *placeholderSigner) Sign(xdr string, kp *Keypair) (string, error) {
	if xdr == "" {
		return "", errors.New("stellar: empty challenge XDR")
	}
	if !kp.CanSign() {
		return "", errors.New("stellar: keypair has no seed")
	}
	return "", errors.New("stellar: signer not yet wired to upstream SDK (Phase 1)")
}

// Network returns the passphrase this signer targets.
func (s *placeholderSigner) Network() string { return s.network }
