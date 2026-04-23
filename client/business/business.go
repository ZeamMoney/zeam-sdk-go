// Package business wraps the gateway's /v1/business/* surface.
//
// All endpoints require a Firebase-backed [auth.TrackBusiness] session.
// The gateway forwards the Authorization header verbatim to the Business
// API, which scopes by firebase_uid.
package business

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ZeamMoney/zeam-sdk-go/auth"
	"github.com/ZeamMoney/zeam-sdk-go/client"
	"github.com/ZeamMoney/zeam-sdk-go/stellar"
)

// Client is the /v1/business/* wrapper.
type Client struct{ D client.Doer }

// New constructs a business client.
func New(d client.Doer) *Client { return &Client{D: d} }

// Association is the minimal association record returned by the Business API.
type Association struct {
	ID              string          `json:"id"`
	AssociationName string          `json:"association_name"`
	Raw             json.RawMessage `json:"-"`
}

// Wallet is a Stellar wallet belonging to an association.
type Wallet struct {
	ID        string          `json:"id"`
	PublicKey string          `json:"publicKey"`
	Type      string          `json:"type"`
	Balances  json.RawMessage `json:"balances"`
	Raw       json.RawMessage `json:"-"`
}

// Beneficiary carries enough fields for the ConnectPayment recipe. Full
// field matrix lives in docs/BENEFICIARIES.md upstream.
type Beneficiary struct {
	ID                  string               `json:"id"`
	PaymentDestinations []PaymentDestination `json:"payment_destinations"`
	Raw                 json.RawMessage      `json:"-"`
}

// PaymentDestination is the chosen destination on a beneficiary.
type PaymentDestination struct {
	ID         string          `json:"id"`
	Method     string          `json:"method"`
	CountryISO string          `json:"country_iso"`
	IsPrimary  bool            `json:"is_primary"`
	Raw        json.RawMessage `json:"raw"`
}

// StellarQuoteInput is the body of POST /v1/business/stellar/quote.
type StellarQuoteInput struct {
	FromAsset string `json:"fromAsset"`
	ToAsset   string `json:"toAsset"`
	Amount    string `json:"amount"`
}

// StellarQuote is the strict-receive quote response.
type StellarQuote struct {
	SendMax string          `json:"sendMax"`
	Path    json.RawMessage `json:"path"`
	FX      json.RawMessage `json:"fx"`
	Raw     json.RawMessage `json:"-"`
}

// WalletTransactionInput is the body of POST /v1/business/wallet/{id}/transaction.
type WalletTransactionInput struct {
	ToPublicKey string `json:"toPublicKey"`
	FromAsset   string `json:"fromAsset"`
	ToAsset     string `json:"toAsset,omitempty"`
	Amount      string `json:"amount"`
	SendMax     string `json:"sendMax,omitempty"`
	Memo        string `json:"memo,omitempty"`
	MemoType    string `json:"memoType,omitempty"`
}

// WalletTransactionResult carries the Horizon submission result.
type WalletTransactionResult struct {
	TxHash string          `json:"txHash"`
	XDR    string          `json:"xdr"`
	Raw    json.RawMessage `json:"-"`
}

// ListAssociations calls GET /v1/business/association/all.
func (c *Client) ListAssociations(ctx context.Context, sess *auth.Session) ([]Association, error) {
	var out []Association
	err := client.Call(ctx, c.D, http.MethodGet, "/v1/business/association/all", nil, sess, auth.TrackBusiness, "", nil, &out)
	return out, err
}

// ListWalletsByAssociation calls GET /v1/business/wallet/association/{id}.
func (c *Client) ListWalletsByAssociation(ctx context.Context, sess *auth.Session, associationID string) ([]Wallet, error) {
	path := fmt.Sprintf("/v1/business/wallet/association/%s", associationID)
	var out []Wallet
	err := client.Call(ctx, c.D, http.MethodGet, path, nil, sess, auth.TrackBusiness, "", nil, &out)
	return out, err
}

// GetBeneficiary calls GET /v1/business/beneficiaries/{associationId}/{id}.
func (c *Client) GetBeneficiary(ctx context.Context, sess *auth.Session, associationID, beneficiaryID string) (*Beneficiary, error) {
	path := fmt.Sprintf("/v1/business/beneficiaries/%s/%s", associationID, beneficiaryID)
	var out Beneficiary
	err := client.Call(ctx, c.D, http.MethodGet, path, nil, sess, auth.TrackBusiness, "", nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// StellarQuote calls POST /v1/business/stellar/quote. Returns
// [stellar.ErrInvalidAsset] (wrapped) when either asset fails local
// validation before the network call.
func (c *Client) StellarQuote(ctx context.Context, sess *auth.Session, in StellarQuoteInput) (*StellarQuote, error) {
	if _, err := stellar.ParseAsset(in.FromAsset); err != nil {
		return nil, err
	}
	if _, err := stellar.ParseAsset(in.ToAsset); err != nil {
		return nil, err
	}
	var out StellarQuote
	err := client.Call(ctx, c.D, http.MethodPost, "/v1/business/stellar/quote", nil, sess, auth.TrackBusiness, "", in, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ExecuteWalletTransaction calls POST /v1/business/wallet/{id}/transaction.
func (c *Client) ExecuteWalletTransaction(
	ctx context.Context,
	sess *auth.Session,
	walletID string,
	in WalletTransactionInput,
) (*WalletTransactionResult, error) {
	path := fmt.Sprintf("/v1/business/wallet/%s/transaction", walletID)
	var out WalletTransactionResult
	err := client.Call(ctx, c.D, http.MethodPost, path, nil, sess, auth.TrackBusiness, "", in, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
