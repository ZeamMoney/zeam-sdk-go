// Package application wraps /v1/application/* — the Firebase-authenticated
// Association Applications surface. The registration call returns
// one-time-only credentials (stellar.secret, connectSecret, apiKey.secret,
// webhookSecret.secret) and is the single most security-sensitive call in
// the SDK; [OneTimeSecrets] must be captured by the partner before the
// SDK zeroes them.
package application

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ZeamMoney/zeam-sdk-go/auth"
	"github.com/ZeamMoney/zeam-sdk-go/client"
)

// Client wraps /v1/application/*.
type Client struct{ D client.Doer }

// New constructs an application client.
func New(d client.Doer) *Client { return &Client{D: d} }

// Stellar carries the Stellar seed returned at registration. The seed
// MUST be captured and persisted securely; subsequent reads will not
// return it.
type Stellar struct {
	PublicKey string `json:"publicKey"`
	Secret    string `json:"secret"`
	VaultID   string `json:"vaultId"`
}

// APIKey carries the API key material. keyId + last4 are safe for
// display; secret is only returned once.
type APIKey struct {
	KeyID  string `json:"keyId"`
	Secret string `json:"secret"`
	Last4  string `json:"last4"`
}

// WebhookSecret carries the HMAC signing secret for inbound webhooks.
type WebhookSecret struct {
	WebhookID string `json:"webhookId"`
	Secret    string `json:"secret"`
	Last4     string `json:"last4"`
}

// RegistrationResponse mirrors POST /v1/application. All fields tagged
// as "one-time only" in docs/CLI_AUTH_FLOWS.md §4 are captured here.
type RegistrationResponse struct {
	Application   json.RawMessage `json:"application"`
	IntegratorID  string          `json:"integratorId"`
	Stellar       Stellar         `json:"stellar"`
	APIKey        APIKey          `json:"apiKey"`
	WebhookSecret WebhookSecret   `json:"webhookSecret"`
	ConnectSecret string          `json:"connectSecret"`
	IDPAuthID     string          `json:"idpAuthId"`
	Warnings      []string        `json:"warnings"`
}

// RegistrationInput is the request body for POST /v1/application.
type RegistrationInput struct {
	AssociationID     string `json:"associationId"`
	ApplicationName   string `json:"applicationName"`
	WebhookURL        string `json:"webhookUrl,omitempty"`
	WebhookMethod     string `json:"webhookMethod,omitempty"` // "POST" or "PUT"
	ExpiresAt         string `json:"expiresAt,omitempty"`     // RFC3339
}

// Register calls POST /v1/application. The caller is responsible for
// persisting the one-time credentials BEFORE discarding the returned
// struct — subsequent reads will not return the `stellar.secret`,
// `apiKey.secret`, `webhookSecret.secret`, or `connectSecret` fields.
func (c *Client) Register(ctx context.Context, sess *auth.Session, payload RegistrationInput) (*RegistrationResponse, error) {
	var out RegistrationResponse
	err := client.Call(ctx, c.D, http.MethodPost, "/v1/application", nil, sess, auth.TrackBusiness, "", payload, &out)
	if err != nil {
		return nil, err
	}
	if out.ConnectSecret == "" || out.Stellar.Secret == "" {
		return nil, fmt.Errorf("application: registration response missing one-time credentials — partner will not be able to call Connect")
	}
	return &out, nil
}

// RotateCredentialResponse mirrors POST /v1/application/{id}/rotate-credential.
type RotateCredentialResponse struct {
	APIKey APIKey `json:"apiKey"`
}

// RotateCredential calls POST /v1/application/{id}/rotate-credential.
// The old credential is moved to `rotating` state by the upstream; the
// caller MUST migrate promptly.
func (c *Client) RotateCredential(ctx context.Context, sess *auth.Session, id string) (*RotateCredentialResponse, error) {
	path := fmt.Sprintf("/v1/application/%s/rotate-credential", id)
	var out RotateCredentialResponse
	err := client.Call(ctx, c.D, http.MethodPost, path, nil, sess, auth.TrackBusiness, "", nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// RotateWebhookSecretResponse mirrors POST /v1/application/{id}/webhook/{webhookId}/rotate-secret.
type RotateWebhookSecretResponse struct {
	WebhookSecret WebhookSecret `json:"webhookSecret"`
}

// RotateWebhookSecret rotates the HMAC signing secret for a webhook. The
// old secret enters `rotating` state upstream and remains valid until
// the caller migrates.
func (c *Client) RotateWebhookSecret(ctx context.Context, sess *auth.Session, id, webhookID string) (*RotateWebhookSecretResponse, error) {
	path := fmt.Sprintf("/v1/application/%s/webhook/%s/rotate-secret", id, webhookID)
	var out RotateWebhookSecretResponse
	err := client.Call(ctx, c.D, http.MethodPost, path, nil, sess, auth.TrackBusiness, "", nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
