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

// ── Quote (POST /v1/connect-quote) ────────────────────────────────────
// Source of truth: connect.zeam.dotnet QuoteRequestDto / QuoteResponseDto

// QuoteInput is the request body for POST /v1/connect-quote.
type QuoteInput struct {
	ConnectorID     string    `json:"connectorId"`
	Amount          float64   `json:"amount"`
	TransactionType string    `json:"transactionType,omitempty"` // e.g. "C2C"
	QR              *QRCode   `json:"qr,omitempty"`
}

// QRCode is an optional QR code payload for quote requests.
type QRCode struct {
	RawCode string `json:"rawCode"`
}

// QuoteResponse is the 402 response from POST /v1/connect-quote.
type QuoteResponse struct {
	QuoteID             string               `json:"quoteId"`
	ConnectorID         string               `json:"connectorId"`
	Direction           string               `json:"direction"`
	SendAmount          float64              `json:"sendAmount"`
	SendCurrency        string               `json:"sendCurrency"`
	ReceiveAmount       float64              `json:"receiveAmount"`
	ReceiveCurrency     string               `json:"receiveCurrency"`
	Rate                float64              `json:"rate"`
	Fee                 float64              `json:"fee"`
	FeeCurrency         string               `json:"feeCurrency"`
	Total               float64              `json:"total"`
	CreatedAt           string               `json:"createdAt"`
	ExpiresAt           string               `json:"expiresAt"`
	TransactionType     string               `json:"transactionType"`
	FundingInstructions *QuoteFunding        `json:"fundingInstructions"`
	Metadata            json.RawMessage      `json:"metadata,omitempty"`
}

// QuoteFunding contains the Stellar payment instructions from a quote.
type QuoteFunding struct {
	Network            string          `json:"network"`
	DestinationAccount string          `json:"destinationAccount"`
	Asset              QuoteFundingAsset `json:"asset"`
	MemoType           string          `json:"memoType"`
	Memo               string          `json:"memo"`
}

// QuoteFundingAsset identifies the Stellar asset for funding.
type QuoteFundingAsset struct {
	Code   string `json:"code"`
	Issuer string `json:"issuer"`
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

// ── Execute / OffRamp (POST /v1/connect-execute) ────────────────────
// Source of truth: connect.zeam.dotnet OffRampRequestDto / OffRampResponseDto

// ExecuteInput is the request body for POST /v1/connect-execute.
// Required fields: Reference, QuoteID, RefundAccount, TransactionHash.
// The remaining fields are method-specific — populate the ones that match
// the connector's payment method (e.g. Bank for BANK, Cash for CASH).
type ExecuteInput struct {
	Reference           string               `json:"reference"`
	QuoteID             string               `json:"quoteId"`
	RefundAccount       RefundAccount        `json:"refundAccount"`
	TransactionHash     string               `json:"transactionHash"`
	Purpose             string               `json:"purpose,omitempty"`
	Sender              *Sender              `json:"sender,omitempty"`
	BusinessSender      *BusinessSender      `json:"businessSender,omitempty"`
	Beneficiary         *Beneficiary         `json:"beneficiary,omitempty"`
	BusinessBeneficiary *BusinessBeneficiary `json:"businessBeneficiary,omitempty"`
	MobileMoney         *MobileMoney         `json:"mobileMoney,omitempty"`
	Bank                *Bank                `json:"bank,omitempty"`
	Voucher             *Voucher             `json:"voucher,omitempty"`
	Cash                *Cash                `json:"cash,omitempty"`
	QR                  *QRPayment           `json:"qr,omitempty"`
}

// RefundAccount identifies the Stellar account for refunds.
type RefundAccount struct {
	Account string `json:"account"`
}

// Sender is individual sender KYC data.
type Sender struct {
	FirstName                string `json:"firstName,omitempty"`
	Middlename               string `json:"middlename,omitempty"`
	Lastname                 string `json:"lastname,omitempty"`
	NationalityCc            string `json:"nationalityCc,omitempty"`
	Dob                      string `json:"dob,omitempty"`
	CountryOfBirth           string `json:"countryOfBirth,omitempty"`
	Gender                   string `json:"gender,omitempty"`
	Address                  string `json:"address,omitempty"`
	PostalCode               string `json:"postalCode,omitempty"`
	City                     string `json:"city,omitempty"`
	CountryIsoCode           string `json:"countryIsoCode,omitempty"`
	Msisdn                   string `json:"msisdn"`
	Email                    string `json:"email,omitempty"`
	IdType                   string `json:"idType,omitempty"`
	IdCountryIsoCode         string `json:"idCountryIsoCode,omitempty"`
	IdNumber                 string `json:"idNumber,omitempty"`
	IdExpiryDate             string `json:"idExpiryDate,omitempty"`
	Occupation               string `json:"occupation,omitempty"`
	ProvinceState            string `json:"provinceState,omitempty"`
	BeneficiaryRelationship  string `json:"beneficiaryRelationship,omitempty"`
	SourceOfFunds            string `json:"sourceOfFunds,omitempty"`
}

// BusinessSender is business sender KYC data.
type BusinessSender struct {
	RegisteredName                     string `json:"registered_name"`
	TradingName                        string `json:"trading_name"`
	Address                            string `json:"address"`
	PostalCode                         string `json:"postal_code"`
	City                               string `json:"city"`
	CountryIsoCode                     string `json:"country_iso_code"`
	Msisdn                             string `json:"msisdn"`
	Email                              string `json:"email"`
	DateOfIncorporation                string `json:"date_of_incorporation"`
	RepresentativeLastname             string `json:"representative_lastname"`
	RepresentativeFirstname            string `json:"representative_firstname"`
	RepresentativeIdCountryIsoCode     string `json:"representative_id_country_iso_code"`
	ProvinceState                      string `json:"province_state,omitempty"`
	RegistrationNumber                 string `json:"registration_number,omitempty"`
	Code                               string `json:"code,omitempty"`
	TaxId                              string `json:"tax_id,omitempty"`
}

// Beneficiary is individual beneficiary KYC data.
type Beneficiary struct {
	FirstName                string `json:"firstName,omitempty"`
	Middlename               string `json:"middlename,omitempty"`
	Lastname                 string `json:"lastname,omitempty"`
	NationalityCc            string `json:"nationalityCc,omitempty"`
	Dob                      string `json:"dob,omitempty"`
	CountryOfBirth           string `json:"countryOfBirth,omitempty"`
	Gender                   string `json:"gender,omitempty"`
	Address                  string `json:"address,omitempty"`
	PostalCode               string `json:"postalCode,omitempty"`
	City                     string `json:"city,omitempty"`
	CountryIsoCode           string `json:"countryIsoCode,omitempty"`
	Msisdn                   string `json:"msisdn"`
	Email                    string `json:"email,omitempty"`
	IdType                   string `json:"idType,omitempty"`
	IdCountryIsoCode         string `json:"idCountryIsoCode,omitempty"`
	IdNumber                 string `json:"idNumber,omitempty"`
	IdExpiryDate             string `json:"idExpiryDate,omitempty"`
	Occupation               string `json:"occupation,omitempty"`
	ProvinceState            string `json:"provinceState,omitempty"`
	BeneficiaryRelationship  string `json:"beneficiaryRelationship,omitempty"`
	SourceOfFunds            string `json:"sourceOfFunds,omitempty"`
}

// BusinessBeneficiary is business beneficiary KYC data.
type BusinessBeneficiary struct {
	RegisteredName                     string `json:"registered_name"`
	TradingName                        string `json:"trading_name"`
	Address                            string `json:"address"`
	PostalCode                         string `json:"postal_code"`
	City                               string `json:"city"`
	CountryIsoCode                     string `json:"country_iso_code"`
	Msisdn                             string `json:"msisdn"`
	Email                              string `json:"email"`
	DateOfIncorporation                string `json:"date_of_incorporation"`
	RepresentativeLastname             string `json:"representative_lastname"`
	RepresentativeFirstname            string `json:"representative_firstname"`
	RepresentativeIdCountryIsoCode     string `json:"representative_id_country_iso_code"`
	ProvinceState                      string `json:"province_state,omitempty"`
	RegistrationNumber                 string `json:"registration_number,omitempty"`
	TaxId                              string `json:"tax_id,omitempty"`
}

// MobileMoney is the mobile money payment method details.
type MobileMoney struct {
	Msisdn        string `json:"msisdn,omitempty"`
	AccountNumber string `json:"accountNumber,omitempty"`
}

// Bank is the bank transfer payment method details.
type Bank struct {
	AccountNumber    string `json:"accountNumber"`
	BranchCode       string `json:"branchCode,omitempty"`
	AccountName      string `json:"accountName,omitempty"`
	AccountType      string `json:"accountType,omitempty"`
	SwiftCode        string `json:"swiftCode,omitempty"`
	Iban             string `json:"iban,omitempty"`
	Clabe            string `json:"clabe,omitempty"`
	Cbu              string `json:"cbu,omitempty"`
	CbuAlias         string `json:"cbu_alias,omitempty"`
	BikCode          string `json:"bik_code,omitempty"`
	IfsCode          string `json:"ifs_code,omitempty"`
	SortCode         string `json:"sort_code,omitempty"`
	AbaRoutingNumber string `json:"aba_routing_number,omitempty"`
	BsbNumber        string `json:"bsb_number,omitempty"`
	RoutingCode      string `json:"routing_code,omitempty"`
	EntityTtId       string `json:"entity_tt_id,omitempty"`
	Email            string `json:"email,omitempty"`
	CardNumber       string `json:"card_number,omitempty"`
	QrCode           string `json:"qr_code,omitempty"`
	BankId           *int   `json:"bankId,omitempty"`
}

// Voucher is the voucher payment method details.
type Voucher struct {
	Product string `json:"product"`
}

// Cash is the cash payment method details.
type Cash struct {
	Reference string `json:"reference"`
}

// QRPayment is the QR payment method details.
type QRPayment struct {
	Amount float64 `json:"amount"`
	Tip    float64 `json:"tip"`
}

// ExecuteResponse is the response from POST /v1/connect-execute.
type ExecuteResponse struct {
	Status            string `json:"status"`
	TransactionID     string `json:"transactionId"`
	ExternalReference string `json:"externalReference"`
	ConnectorID       string `json:"connectorId"`
	CreatedAt         string `json:"createdAt"`
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
