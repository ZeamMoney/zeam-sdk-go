package recipes

import (
	"context"
	"errors"
	"fmt"

	"github.com/ZeamMoney/zeam-sdk-go/auth"
	"github.com/ZeamMoney/zeam-sdk-go/client/application"
)

// RotateCredentialInput drives [RotateCredential].
type RotateCredentialInput struct {
	// Session is the Firebase-authenticated owner of the application.
	Session *auth.Session
	// ApplicationID is the `{id}` path parameter.
	ApplicationID string
	// CaptureNew is invoked with the freshly minted API key material.
	// Partners MUST persist the new key via their secret manager before
	// the callback returns. The old credential stays valid until it is
	// revoked or the rotating window closes upstream.
	CaptureNew func(ctx context.Context, keyID, newSecret, last4 string) error
}

// RotateCredentialClient is the subset of the client the recipe needs.
type RotateCredentialClient interface {
	Application() *application.Client
}

// RotateCredential runs POST /v1/application/{id}/rotate-credential and
// hands the new API key material to the partner callback.
func RotateCredential(ctx context.Context, c RotateCredentialClient, in RotateCredentialInput) error {
	if c == nil || c.Application() == nil {
		return errors.New("recipes: RotateCredential requires a configured client")
	}
	if in.Session == nil {
		return errors.New("recipes: RotateCredential requires an authenticated session")
	}
	if in.ApplicationID == "" {
		return errors.New("recipes: ApplicationID is required")
	}
	if in.CaptureNew == nil {
		return errors.New("recipes: CaptureNew callback is required")
	}

	resp, err := c.Application().RotateCredential(ctx, in.Session, in.ApplicationID)
	if err != nil {
		return fmt.Errorf("recipes: rotate credential: %w", err)
	}
	captureErr := in.CaptureNew(ctx, resp.APIKey.KeyID, resp.APIKey.Secret, resp.APIKey.Last4)
	resp.APIKey.Secret = ""
	if captureErr != nil {
		return fmt.Errorf("recipes: CaptureNew: %w", captureErr)
	}
	return nil
}
