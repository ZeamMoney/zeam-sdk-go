package recipes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ZeamMoney/zeam-sdk-go/auth"
	"github.com/ZeamMoney/zeam-sdk-go/client/application"
)

// OneTimeSecrets carries the credentials that POST /v1/application
// returns exactly once. Partners MUST persist every field via their
// secret manager BEFORE the callback returns; the SDK zeroes the
// underlying struct immediately after.
type OneTimeSecrets struct {
	StellarSeed      string
	StellarPublicKey string
	ConnectSecret    string
	APIKey           string
	WebhookSecret    string
	WebhookID        string
}

// RegisterAppInput drives [RegisterApplication].
type RegisterAppInput struct {
	// Session is the Firebase-authenticated Business session holding
	// permission to register an application.
	Session *auth.Session
	// Payload is the registration body for POST /v1/application.
	Payload application.RegistrationInput
	// CaptureOneTimeSecrets is invoked exactly once with the four
	// credentials the gateway returns. If the callback returns an
	// error, the SDK zeros the secrets and returns that error.
	CaptureOneTimeSecrets func(ctx context.Context, s OneTimeSecrets) error
}

// RegisterAppClient is the subset of the top-level client required by
// the registration recipe.
type RegisterAppClient interface {
	Application() *application.Client
}

// RegisterAppResult is the non-sensitive shell returned to the caller
// after the one-time secrets have been captured.
type RegisterAppResult struct {
	IntegratorID     string
	ApplicationBlob  json.RawMessage
	StellarPublicKey string
	APIKeyID         string
	APIKeyLast4      string
	WebhookID        string
	WebhookLast4     string
	Warnings         []string
}

// RegisterApplication calls POST /v1/application, invokes the partner
// callback with the one-time secrets, and returns a
// [RegisterAppResult] that carries only the safe-to-log fields.
func RegisterApplication(ctx context.Context, c RegisterAppClient, in RegisterAppInput) (*RegisterAppResult, error) {
	if c == nil || c.Application() == nil {
		return nil, errors.New("recipes: RegisterApplication requires a configured client")
	}
	if in.Session == nil {
		return nil, errors.New("recipes: RegisterApplication requires an authenticated session")
	}
	if in.CaptureOneTimeSecrets == nil {
		return nil, errors.New("recipes: CaptureOneTimeSecrets callback is required")
	}

	resp, err := c.Application().Register(ctx, in.Session, in.Payload)
	if err != nil {
		return nil, fmt.Errorf("recipes: register application: %w", err)
	}

	secrets := OneTimeSecrets{
		StellarSeed:      resp.Stellar.Secret,
		StellarPublicKey: resp.Stellar.PublicKey,
		ConnectSecret:    resp.ConnectSecret,
		APIKey:           resp.APIKey.Secret,
		WebhookSecret:    resp.WebhookSecret.Secret,
		WebhookID:        resp.WebhookSecret.WebhookID,
	}
	captureErr := in.CaptureOneTimeSecrets(ctx, secrets)

	// Zero the in-memory copies regardless of capture outcome.
	resp.Stellar.Secret = ""
	resp.ConnectSecret = ""
	resp.APIKey.Secret = ""
	resp.WebhookSecret.Secret = ""
	secrets = OneTimeSecrets{}
	_ = secrets

	if captureErr != nil {
		return nil, fmt.Errorf("recipes: CaptureOneTimeSecrets: %w", captureErr)
	}

	return &RegisterAppResult{
		IntegratorID:     resp.IntegratorID,
		ApplicationBlob:  resp.Application,
		StellarPublicKey: resp.Stellar.PublicKey,
		APIKeyID:         resp.APIKey.KeyID,
		APIKeyLast4:      resp.APIKey.Last4,
		WebhookID:        resp.WebhookSecret.WebhookID,
		WebhookLast4:     resp.WebhookSecret.Last4,
		Warnings:         resp.Warnings,
	}, nil
}
