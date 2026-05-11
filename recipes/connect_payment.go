package recipes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ZeamMoney/zeam-sdk-go/auth"
	"github.com/ZeamMoney/zeam-sdk-go/client/business"
	"github.com/ZeamMoney/zeam-sdk-go/client/connect"
	"github.com/ZeamMoney/zeam-sdk-go/stellar"
)

// ConnectPaymentClient is the subset of the top-level client the
// flagship recipe needs.
type ConnectPaymentClient interface {
	Business() *business.Client
	Connect() *connect.Client
	SEP10() *auth.SEP10Flow
	Store() auth.TokenStore
}

// ConnectPaymentInput drives [ConnectPayment]. Mirrors the nine steps of
// docs/CONNECT_PAYMENT_FLOW.md §1 upstream.
type ConnectPaymentInput struct {
	// BusinessSession is an authenticated Business session (from LoginOTP
	// or a refreshed handle).
	BusinessSession *auth.Session

	// ApplicationSeed and ApplicationPublicKey are the Stellar credentials
	// captured at application registration. The recipe signs the SEP-10
	// challenge with them and zeroes the keypair immediately after.
	ApplicationSeed      string
	ApplicationPublicKey string

	// AssociationID is chosen from the list returned in Step 1.
	AssociationID string
	// WalletID is chosen from Step 2; typically the partner UI picks this.
	WalletID string
	// FundingAsset is the asset the partner's wallet holds and will debit.
	FundingAsset stellar.Asset
	// BeneficiaryID is looked up in Step 3.
	BeneficiaryID string
	// Method and CountryISO must match gateway ^[A-Z0-9_]+$ and the
	// beneficiary's selected destination.
	Method     string
	CountryISO string
	// SendAmount is the amount in the Connect-quoted accepted asset. Max
	// 7 fractional digits; > 0.
	SendAmount string
	// Memo / MemoType are carried verbatim on the Stellar transaction.
	// Memos are Connect-owned (the upstream returns a correlation memo in
	// the quote response); leave empty to let Step 6 decide.
	Memo     string
	MemoType string
}

// ConnectPaymentResult carries the outputs the caller is most likely to
// surface or persist.
type ConnectPaymentResult struct {
	ConnectTransactionID string
	ConnectStatus        string
	StellarTxHash        string
	QuoteID              string
	Connector            connect.Connector
	ConnectSession       *auth.Session
}

// ConnectPayment runs the full 9-step off-ramp orchestration.
//
// Named steps are exported on [*ConnectPayment] so callers can drive the
// flow manually when they need to pause between steps (e.g. show a
// confirmation UI after GetConnectQuote).
func NewConnectPayment(c ConnectPaymentClient, in ConnectPaymentInput) *ConnectPayment {
	return &ConnectPayment{client: c, input: in}
}

// ConnectPayment is a stateful recipe runner.
type ConnectPayment struct {
	client ConnectPaymentClient
	input  ConnectPaymentInput

	// Populated as steps run.
	Associations  []business.Association
	Wallets       []business.Wallet
	Beneficiary   *business.Beneficiary
	ConnectSess   *auth.Session
	Connectors    []connect.Connector
	Selected      connect.Connector
	ConnectQuote  *connect.QuoteResponse
	StellarQuote  *business.StellarQuote
	WalletTxHash  string
	ExecuteResult *connect.ExecuteResponse
}

// Do runs every step in order. Individual steps are exposed too; any
// caller can interleave UX between them.
func (f *ConnectPayment) Do(ctx context.Context) (*ConnectPaymentResult, error) {
	if err := f.validate(); err != nil {
		return nil, err
	}
	if _, err := f.ListAssociations(ctx); err != nil {
		return nil, err
	}
	if _, err := f.ListWallets(ctx); err != nil {
		return nil, err
	}
	if _, err := f.GetBeneficiary(ctx); err != nil {
		return nil, err
	}
	if _, err := f.SignInConnect(ctx); err != nil {
		return nil, err
	}
	if _, err := f.DiscoverConnectors(ctx); err != nil {
		return nil, err
	}
	if err := f.SelectConnector(); err != nil {
		return nil, err
	}
	if _, err := f.GetConnectQuote(ctx); err != nil {
		return nil, err
	}
	if f.RequiresStellarQuote() {
		if _, err := f.GetStellarQuote(ctx); err != nil {
			return nil, err
		}
	}
	if _, err := f.ExecuteStellarTransaction(ctx); err != nil {
		return nil, err
	}
	if _, err := f.ExecuteConnectPayment(ctx); err != nil {
		return nil, err
	}
	return &ConnectPaymentResult{
		ConnectTransactionID: f.ExecuteResult.TransactionID,
		ConnectStatus:        f.ExecuteResult.Status,
		StellarTxHash:        f.WalletTxHash,
		QuoteID:              f.ConnectQuote.QuoteID,
		Connector:            f.Selected,
		ConnectSession:       f.ConnectSess,
	}, nil
}

func (f *ConnectPayment) validate() error {
	in := f.input
	switch {
	case f.client == nil:
		return errors.New("recipes: ConnectPayment requires a configured client")
	case in.BusinessSession == nil || in.BusinessSession.Track() != auth.TrackBusiness:
		return errors.New("recipes: BusinessSession must be authenticated on TrackBusiness")
	case in.ApplicationSeed == "" || in.ApplicationPublicKey == "":
		return errors.New("recipes: ApplicationSeed and ApplicationPublicKey are required")
	case in.AssociationID == "" || in.WalletID == "" || in.BeneficiaryID == "":
		return errors.New("recipes: association, wallet, and beneficiary IDs are required")
	case in.Method == "" || in.CountryISO == "":
		return errors.New("recipes: Method and CountryISO are required")
	case in.SendAmount == "":
		return errors.New("recipes: SendAmount is required")
	}
	return nil
}

// ListAssociations runs Step 1.
func (f *ConnectPayment) ListAssociations(ctx context.Context) ([]business.Association, error) {
	a, err := f.client.Business().ListAssociations(ctx, f.input.BusinessSession)
	if err != nil {
		return nil, fmt.Errorf("step 1 list associations: %w", err)
	}
	f.Associations = a
	return a, nil
}

// ListWallets runs Step 2.
func (f *ConnectPayment) ListWallets(ctx context.Context) ([]business.Wallet, error) {
	w, err := f.client.Business().ListWalletsByAssociation(ctx, f.input.BusinessSession, f.input.AssociationID)
	if err != nil {
		return nil, fmt.Errorf("step 2 list wallets: %w", err)
	}
	f.Wallets = w
	return w, nil
}

// GetBeneficiary runs Step 3.
func (f *ConnectPayment) GetBeneficiary(ctx context.Context) (*business.Beneficiary, error) {
	b, err := f.client.Business().GetBeneficiary(ctx, f.input.BusinessSession, f.input.AssociationID, f.input.BeneficiaryID)
	if err != nil {
		return nil, fmt.Errorf("step 3 get beneficiary: %w", err)
	}
	f.Beneficiary = b
	return b, nil
}

// SignInConnect runs Step 4.
func (f *ConnectPayment) SignInConnect(ctx context.Context) (*auth.Session, error) {
	kp, err := stellar.NewKeypair(f.input.ApplicationSeed, f.input.ApplicationPublicKey)
	if err != nil {
		return nil, fmt.Errorf("step 4 parse keypair: %w", err)
	}
	defer kp.Erase()

	sess, err := f.client.SEP10().Login(ctx, kp)
	if err != nil {
		return nil, fmt.Errorf("step 4 SEP-10 login: %w", err)
	}
	if err := f.client.Store().Put(ctx, sess); err != nil {
		sess.Erase()
		return nil, fmt.Errorf("step 4 persist connect session: %w", err)
	}
	f.ConnectSess = sess
	return sess, nil
}

// DiscoverConnectors runs Step 5.
func (f *ConnectPayment) DiscoverConnectors(ctx context.Context) ([]connect.Connector, error) {
	cs, err := f.client.Connect().QueryConnectors(ctx, f.ConnectSess, connect.ConnectorQueryInput{
		CountryISO: f.input.CountryISO,
		Method:     f.input.Method,
	})
	if err != nil {
		return nil, fmt.Errorf("step 5 query connectors: %w", err)
	}
	f.Connectors = cs
	return cs, nil
}

// SelectConnector picks the first active connector matching the method.
// Partners preferring a different selection strategy can assign
// f.Selected directly before calling GetConnectQuote.
func (f *ConnectPayment) SelectConnector() error {
	if f.Selected.ID != "" {
		return nil
	}
	for _, c := range f.Connectors {
		if c.IsActive && c.Method == f.input.Method {
			f.Selected = c
			return nil
		}
	}
	return fmt.Errorf("step 5 no active connector for %s in %s", f.input.Method, f.input.CountryISO)
}

// GetConnectQuote runs Step 6.
func (f *ConnectPayment) GetConnectQuote(ctx context.Context) (*connect.QuoteResponse, error) {
	dest, err := f.buildDestination()
	if err != nil {
		return nil, err
	}
	q, err := f.client.Connect().GetQuote(ctx, f.ConnectSess, connect.QuoteInput{
		ConnectorID: f.Selected.ID,
		Amount:      f.input.SendAmount,
		Currency:    f.Selected.AcceptedAsset,
		Destination: dest,
	})
	if err != nil {
		return nil, fmt.Errorf("step 6 get quote: %w", err)
	}
	f.ConnectQuote = q
	return q, nil
}

// RequiresStellarQuote reports whether Step 7 must run. Matches the
// decision rule in docs/CONNECT_PAYMENT_FLOW.md §3.
func (f *ConnectPayment) RequiresStellarQuote() bool {
	if f.ConnectQuote == nil {
		return false
	}
	accepted, err := stellar.ParseAsset(f.ConnectQuote.SendCurrency)
	if err != nil {
		return false
	}
	return !f.input.FundingAsset.Equal(accepted)
}

// GetStellarQuote runs Step 7.
func (f *ConnectPayment) GetStellarQuote(ctx context.Context) (*business.StellarQuote, error) {
	q, err := f.client.Business().StellarQuote(ctx, f.input.BusinessSession, business.StellarQuoteInput{
		FromAsset: f.input.FundingAsset.String(),
		ToAsset:   f.ConnectQuote.SendCurrency,
		Amount:    fmt.Sprintf("%.7f", f.ConnectQuote.SendAmount),
	})
	if err != nil {
		return nil, fmt.Errorf("step 7 stellar quote: %w", err)
	}
	f.StellarQuote = q
	return q, nil
}

// ExecuteStellarTransaction runs Step 8.
func (f *ConnectPayment) ExecuteStellarTransaction(ctx context.Context) (string, error) {
	in := business.WalletTransactionInput{
		ToPublicKey: f.connectClearingAccount(),
		FromAsset:   f.input.FundingAsset.String(),
		Amount:      fmt.Sprintf("%.7f", f.ConnectQuote.SendAmount),
		Memo:        f.input.Memo,
		MemoType:    f.input.MemoType,
	}
	if f.RequiresStellarQuote() {
		in.ToAsset = f.ConnectQuote.SendCurrency
		in.SendMax = f.StellarQuote.SendMax
	}
	result, err := f.client.Business().ExecuteWalletTransaction(ctx, f.input.BusinessSession, f.input.WalletID, in)
	if err != nil {
		return "", fmt.Errorf("step 8 stellar transaction: %w", err)
	}
	f.WalletTxHash = result.TxHash
	return result.TxHash, nil
}

// ExecuteConnectPayment runs Step 9.
func (f *ConnectPayment) ExecuteConnectPayment(ctx context.Context) (*connect.ExecuteResponse, error) {
	dest, err := f.buildDestination()
	if err != nil {
		return nil, err
	}
	result, err := f.client.Connect().Execute(ctx, f.ConnectSess, connect.ExecuteInput{
		QuoteID:     f.ConnectQuote.QuoteID,
		TxHash:      f.WalletTxHash,
		Destination: dest,
		Memo:        f.input.Memo,
	})
	if err != nil {
		return nil, fmt.Errorf("step 9 connect execute: %w", err)
	}
	f.ExecuteResult = result
	return result, nil
}

// connectClearingAccount returns the Stellar public key the Connect
// clearing account currently publishes. The upstream's quote response
// normally includes it; when it doesn't the recipe asks the partner to
// supply it via Input.
func (f *ConnectPayment) connectClearingAccount() string {
	// The gateway + Connect may deliver this in details of the quote; the
	// caller can also patch it into the result before executing.
	// Falling back to Selected.DestinationAsset's issuer is unsafe; this
	// value MUST be sourced from the live quote in production.
	var probe struct {
		ClearingAccount string `json:"clearingAccount"`
	}
	if len(f.ConnectQuote.Raw) > 0 {
		_ = json.Unmarshal(f.ConnectQuote.Raw, &probe)
	}
	return probe.ClearingAccount
}

// buildDestination serialises the chosen beneficiary destination for
// Connect. The exact shape is upstream-owned.
func (f *ConnectPayment) buildDestination() (json.RawMessage, error) {
	if f.Beneficiary == nil {
		return nil, errors.New("beneficiary not loaded; call GetBeneficiary first")
	}
	for _, d := range f.Beneficiary.PaymentDestinations {
		if d.Method == f.input.Method && d.CountryISO == f.input.CountryISO {
			return d.Raw, nil
		}
	}
	return nil, fmt.Errorf("no beneficiary destination for method=%s country=%s", f.input.Method, f.input.CountryISO)
}
