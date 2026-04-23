package recipes

import (
	"context"
	"errors"
	"fmt"

	"github.com/ZeamMoney/zeam-sdk-go/auth"
	"github.com/ZeamMoney/zeam-sdk-go/stellar"
)

// ConnectLoginClient is the subset of the top-level client ConnectLogin
// needs.
type ConnectLoginClient interface {
	SEP10() *auth.SEP10Flow
	Store() auth.TokenStore
}

// ConnectLoginInput drives [ConnectLogin].
type ConnectLoginInput struct {
	// StellarSeed is the `stellar.secret` captured at application
	// registration. The recipe parses and zeroises it; the caller MUST
	// not retain a copy beyond the call.
	StellarSeed string
	// PublicKey is the matching Stellar public key.
	PublicKey string
}

// ConnectLogin runs the SEP-10 flow end-to-end. On success it returns a
// [*auth.Session] on TrackConnect and persists it in the client's
// [auth.TokenStore].
func ConnectLogin(ctx context.Context, c ConnectLoginClient, in ConnectLoginInput) (*auth.Session, error) {
	if c == nil || c.SEP10() == nil || c.Store() == nil {
		return nil, errors.New("recipes: ConnectLogin requires a configured client")
	}
	kp, err := stellar.NewKeypair(in.StellarSeed, in.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("recipes: parse keypair: %w", err)
	}
	defer kp.Erase()

	sess, err := c.SEP10().Login(ctx, kp)
	if err != nil {
		return nil, fmt.Errorf("recipes: SEP-10 login: %w", err)
	}
	if err := c.Store().Put(ctx, sess); err != nil {
		sess.Erase()
		return nil, fmt.Errorf("recipes: persist session: %w", err)
	}
	return sess, nil
}
