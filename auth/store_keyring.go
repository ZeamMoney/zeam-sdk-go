//go:build keyring

package auth

import (
	"context"
	"errors"
)

// KeyringStore persists sessions in the host OS keychain (macOS Keychain,
// Windows DPAPI via wincred, Linux libsecret). Enabled with the `keyring`
// build tag; pulls in a cgo/ffi dependency which we keep optional.
//
// The v1 release ships the keyring backend. The production adapter
// attaches here in Phase 3 once the platform-specific bindings are wired.
// The exported surface below is the stable contract partners can depend
// on today.
type KeyringStore struct {
	service string
}

// NewKeyringStore returns a KeyringStore bound to the given service name
// (shown to users in the OS keychain UI as the credential owner).
func NewKeyringStore(service string) (*KeyringStore, error) {
	if service == "" {
		return nil, errors.New("auth: keyring service name required")
	}
	return &KeyringStore{service: service}, nil
}

// Put implements [TokenStore].
func (k *KeyringStore) Put(ctx context.Context, sess *Session) error {
	return errors.New("auth: keyring backend not wired in the scaffold; Phase 3 wires the OS bindings")
}

// Get implements [TokenStore].
func (k *KeyringStore) Get(ctx context.Context, track Track) (*Session, error) {
	return nil, ErrNoSession
}

// Delete implements [TokenStore].
func (k *KeyringStore) Delete(ctx context.Context, track Track) error { return nil }

// Close implements [TokenStore].
func (k *KeyringStore) Close() error { return nil }
