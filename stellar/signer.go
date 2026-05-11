package stellar

import (
	"errors"
	"fmt"

	"github.com/stellar/go-stellar-sdk/keypair"
	"github.com/stellar/go-stellar-sdk/txnbuild"
)

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
	return &stellarSigner{network: network}
}

type stellarSigner struct{ network string }

// Sign implements [ChallengeSigner]. It parses the challenge XDR, signs
// it with the keypair's seed, and returns the signed base64 XDR.
func (s *stellarSigner) Sign(challengeXDR string, kp *Keypair) (string, error) {
	if challengeXDR == "" {
		return "", errors.New("stellar: empty challenge XDR")
	}
	if !kp.CanSign() {
		return "", errors.New("stellar: keypair has no seed")
	}

	// Parse seed into upstream keypair.
	full, err := keypair.ParseFull(string(kp.seed))
	if err != nil {
		return "", fmt.Errorf("stellar: parse seed: %w", err)
	}

	// Parse the challenge XDR into a transaction.
	genericTx, err := txnbuild.TransactionFromXDR(challengeXDR)
	if err != nil {
		return "", fmt.Errorf("stellar: parse challenge XDR: %w", err)
	}

	tx, ok := genericTx.Transaction()
	if !ok {
		return "", errors.New("stellar: challenge is not a regular transaction")
	}

	// Sign with our keypair on the configured network.
	signed, err := tx.Sign(s.network, full)
	if err != nil {
		return "", fmt.Errorf("stellar: sign challenge: %w", err)
	}

	// Encode signed envelope back to base64 XDR.
	result, err := signed.Base64()
	if err != nil {
		return "", fmt.Errorf("stellar: encode signed XDR: %w", err)
	}
	return result, nil
}

// Network returns the passphrase this signer targets.
func (s *stellarSigner) Network() string { return s.network }
