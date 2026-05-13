// Package payments wraps /v1/transaction/* — transaction initialisation,
// status polling, and load. Firebase authenticated.
package payments

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ZeamMoney/zeam-sdk-go/auth"
	"github.com/ZeamMoney/zeam-sdk-go/client"
)

// Client wraps /v1/transaction/*.
type Client struct{ D client.Doer }

// New constructs a payments client.
func New(d client.Doer) *Client { return &Client{D: d} }

// ── Transaction lifecycle ───────────────────────────────────────────────

// TransactionInitResponse is the response from POST /v1/transaction/init.
type TransactionInitResponse struct {
	RequestID string `json:"requestId"`
}

// TransactionInit creates a new transaction reference. The gateway injects
// the payload; callers must NOT send a body.
func (c *Client) TransactionInit(ctx context.Context, sess *auth.Session) (*TransactionInitResponse, error) {
	var out TransactionInitResponse
	err := client.Call(ctx, c.D, http.MethodPost, "/v1/transaction/init", nil, sess, auth.TrackUnknown, "", nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// TransactionStatusResponse is the response from GET /v1/transaction/status/{id}.
type TransactionStatusResponse struct {
	Status        string `json:"status"`
	TransactionID string `json:"transactionId"`
	TxHash        string `json:"transactionHash"`
	ResultXDR     string `json:"resultXdr"`
	Origin        string `json:"origin"`
}

// TransactionStatus returns the current status of a transaction.
func (c *Client) TransactionStatus(ctx context.Context, sess *auth.Session, transactionID string) (*TransactionStatusResponse, error) {
	path := fmt.Sprintf("/v1/transaction/status/%s", transactionID)
	var out TransactionStatusResponse
	err := client.Call(ctx, c.D, http.MethodGet, path, nil, sess, auth.TrackUnknown, "", nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// TransactionLoadInput is the request body for POST /v1/transaction/load.
type TransactionLoadInput struct {
	RequestID    string        `json:"request_id"`
	Instructions []Instruction `json:"instructions"`
	CustomMemo   string        `json:"custom_memo,omitempty"`
}

// Instruction is a single transfer instruction within a load request.
type Instruction struct {
	From        FromAccount         `json:"from"`
	To          ToAccount           `json:"to"`
	Amount      float64             `json:"amount"`
	PathPayment *PathPaymentOptions `json:"path_payment,omitempty"`
}

// FromAccount identifies the source account and asset.
type FromAccount struct {
	Account         string `json:"account"`
	AssetCode       string `json:"asset_code"`
	Issuer          string `json:"issuer"`
	AuthorizationID string `json:"authorization_id"`
}

// ToAccount identifies the destination account and asset.
type ToAccount struct {
	Account   string `json:"account"`
	AssetCode string `json:"asset_code"`
	Issuer    string `json:"issuer"`
}

// PathPaymentOptions configures cross-asset path payment behaviour.
type PathPaymentOptions struct {
	AmountType        string  `json:"amount_type,omitempty"`        // "StrictReceive" (default) or "StrictSend"
	SlippageTolerance float64 `json:"slippage_tolerance,omitempty"` // default 0.01
}

// TransactionLoadResponse is the response from POST /v1/transaction/load.
type TransactionLoadResponse struct {
	Status    string `json:"status"` // "Queued" or "Failed"
	RequestID string `json:"request_id"`
}

// TransactionLoad loads a transaction for execution via the transactions-acl service.
func (c *Client) TransactionLoad(ctx context.Context, sess *auth.Session, in TransactionLoadInput) (*TransactionLoadResponse, error) {
	var out TransactionLoadResponse
	err := client.Call(ctx, c.D, http.MethodPost, "/v1/transaction/load", nil, sess, auth.TrackUnknown, "", in, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
