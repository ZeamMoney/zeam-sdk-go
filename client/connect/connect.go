// Package connect wraps the gateway's /v1/connect-* surface. It requires
// a [auth.TrackConnect] session and a connector secret passed via the
// x-zeam-auth header.
package connect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/ZeamMoney/zeam-sdk-go/auth"
	"github.com/ZeamMoney/zeam-sdk-go/client"
)

// Client wraps /v1/connect-*.
type Client struct {
	D             client.Doer
	ConnectSecret string
}

// New constructs a connect client bound to the given integrator secret.
func New(d client.Doer, connectSecret string) *Client {
	return &Client{D: d, ConnectSecret: connectSecret}
}

// Connector is one row from the Connectors GraphQL query response.
type Connector struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Method           string          `json:"method"`
	IsActive         bool            `json:"isActive"`
	AcceptedAsset    string          `json:"acceptedAsset"`
	DestinationAsset string          `json:"destinationAsset"`
	RequiresQuote    bool            `json:"requiresQuote"`
	Fees             json.RawMessage `json:"fees"`
	Limits           json.RawMessage `json:"limits"`
	Execution        json.RawMessage `json:"execution"`
}

// ConnectorQueryInput is the validated input for [Client.QueryConnectors].
type ConnectorQueryInput struct {
	CountryISO string
	Method     string
}

// enumPattern mirrors the gateway's SSRF/injection guard: uppercase ASCII,
// digits, and underscores only.
var enumPattern = regexp.MustCompile(`^[A-Z0-9_]+$`)

// QueryConnectors calls POST /v1/connect-query. The gateway owns the
// GraphQL body; the SDK only transmits the two filters.
func (c *Client) QueryConnectors(ctx context.Context, sess *auth.Session, in ConnectorQueryInput) ([]Connector, error) {
	if !enumPattern.MatchString(in.CountryISO) {
		return nil, fmt.Errorf("connect: countryISO must match ^[A-Z0-9_]+$, got %q", in.CountryISO)
	}
	if !enumPattern.MatchString(in.Method) {
		return nil, fmt.Errorf("connect: method must match ^[A-Z0-9_]+$, got %q", in.Method)
	}
	body := map[string]string{"countryISO": in.CountryISO, "method": in.Method}

	// After transport.Unwrap strips the outer {data, status, message}
	// envelope, we receive: {connectors: {connectors: [...]}}.
	var raw struct {
		Connectors struct {
			Connectors []Connector `json:"connectors"`
		} `json:"connectors"`
	}
	err := client.Call(ctx, c.D, http.MethodPost, "/v1/connect-query", nil, sess, auth.TrackConnect, c.ConnectSecret, body, &raw)
	if err != nil {
		return nil, err
	}
	return raw.Connectors.Connectors, nil
}

// QuoteInput is the body of POST /v1/connect-quote. The fields map to
// the Connect owner's schema; partners pass destination-specific payloads
// as raw JSON via [QuoteInput.Destination].
type QuoteInput struct {
	ConnectorID string          `json:"connectorId"`
	Amount      string          `json:"amount"`
	Currency    string          `json:"currency"`
	Destination json.RawMessage `json:"destination"`
}

// QuoteResponse is the gateway's quote response, fields matching
// Connect's upstream contract.
type QuoteResponse struct {
	QuoteID             string          `json:"quoteId"`
	ConnectorID         string          `json:"connectorId"`
	Direction           string          `json:"direction"`
	SendAmount          float64         `json:"sendAmount"`
	SendCurrency        string          `json:"sendCurrency"`
	ReceiveAmount       float64         `json:"receiveAmount"`
	ReceiveCurrency     string          `json:"receiveCurrency"`
	Rate                float64         `json:"rate"`
	Fee                 float64         `json:"fee"`
	FeeCurrency         string          `json:"feeCurrency"`
	Total               float64         `json:"total"`
	ExpiresAt           string          `json:"expiresAt"`
	CreatedAt           string          `json:"createdAt"`
	TransactionType     string          `json:"transactionType"`
	FundingInstructions json.RawMessage `json:"fundingInstructions"`
	Raw                 json.RawMessage `json:"-"`
}

// GetQuote calls POST /v1/connect-quote.
func (c *Client) GetQuote(ctx context.Context, sess *auth.Session, in QuoteInput) (*QuoteResponse, error) {
	var out QuoteResponse
	err := client.Call(ctx, c.D, http.MethodPost, "/v1/connect-quote", nil, sess, auth.TrackConnect, c.ConnectSecret, in, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ExecuteInput is the body of POST /v1/connect-execute.
type ExecuteInput struct {
	QuoteID         string          `json:"quoteId"`
	TransactionHash string          `json:"transactionHash"`
	Reference       string          `json:"reference"`
	RefundAccount   RefundAccount   `json:"refundAccount"`
	Beneficiary     json.RawMessage `json:"beneficiary,omitempty"`
}

// RefundAccount identifies the Stellar account for refunds if the off-ramp fails.
type RefundAccount struct {
	Account string `json:"account"`
}

// ExecuteResponse is the result of the payout call.
type ExecuteResponse struct {
	TransactionID string          `json:"transactionId"`
	Status        string          `json:"status"`
	Raw           json.RawMessage `json:"-"`
}

// Execute calls POST /v1/connect-execute.
func (c *Client) Execute(ctx context.Context, sess *auth.Session, in ExecuteInput) (*ExecuteResponse, error) {
	var out ExecuteResponse
	err := client.Call(ctx, c.D, http.MethodPost, "/v1/connect-execute", nil, sess, auth.TrackConnect, c.ConnectSecret, in, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ConnectStatusResponse is the response from GET /v1/connect-status/{transaction_id}.
type ConnectStatusResponse struct {
	TransactionID string          `json:"transactionId"`
	Status        string          `json:"status"`
	TxHash        string          `json:"txHash"`
	Raw           json.RawMessage `json:"-"`
}

// GetStatus calls GET /v1/connect-status/{transaction_id}.
func (c *Client) GetStatus(ctx context.Context, sess *auth.Session, transactionID string) (*ConnectStatusResponse, error) {
	path := fmt.Sprintf("/v1/connect-status/%s", transactionID)
	var out ConnectStatusResponse
	err := client.Call(ctx, c.D, http.MethodGet, path, nil, sess, auth.TrackConnect, c.ConnectSecret, nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ExecPath is the guarded path regex
// `path/segments` / digits / underscores / hyphens only; rejects any
// scheme, host, or `..` segment.
var ExecPath = regexp.MustCompile(`^[a-zA-Z0-9/_\-]+$`)

// Exec is the escape hatch for metadata-driven Connect calls against
// /v1/connect-exec/{path}. The path must match [ExecPath]; absolute
// URIs, query strings, and traversal segments are rejected before the
// network call (SSRF guard mirroring the gateway's
// `absolute_path_not_allowed` error code).
func (c *Client) Exec(
	ctx context.Context,
	sess *auth.Session,
	method, path string,
	body any,
	out any,
) error {
	if path == "" {
		return errors.New("connect: exec path is required")
	}
	if strings.Contains(path, "..") {
		return errors.New("connect: exec path must not contain `..`")
	}
	if strings.Contains(path, "://") || strings.HasPrefix(path, "//") {
		return errors.New("connect: exec path must not be absolute")
	}
	if !ExecPath.MatchString(path) {
		return fmt.Errorf("connect: exec path must match %s, got %q", ExecPath.String(), path)
	}
	full := "/v1/connect-exec/" + strings.TrimLeft(path, "/")
	return client.Call(ctx, c.D, method, full, nil, sess, auth.TrackConnect, c.ConnectSecret, body, out)
}
