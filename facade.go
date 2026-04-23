package zeam

import (
	"sync"

	"github.com/ZeamMoney/zeam-sdk-go/auth"
	"github.com/ZeamMoney/zeam-sdk-go/client/application"
	"github.com/ZeamMoney/zeam-sdk-go/client/business"
	"github.com/ZeamMoney/zeam-sdk-go/client/connect"
	"github.com/ZeamMoney/zeam-sdk-go/client/health"
	"github.com/ZeamMoney/zeam-sdk-go/client/payments"
	"github.com/ZeamMoney/zeam-sdk-go/client/reports"
	"github.com/ZeamMoney/zeam-sdk-go/stellar"
)

// lazy holds the lazily-initialised facade helpers. Created on first use
// to avoid constructing sub-clients a given process never touches.
type lazy struct {
	once sync.Once

	otp         *auth.OTPFlow
	sep10       *auth.SEP10Flow
	store       auth.TokenStore
	business    *business.Client
	application *application.Client
	connect     *connect.Client
	payments    *payments.Client
	reports     *reports.Client
	health      *health.Client
}

func (c *Client) lazyInit() {
	c.lz.once.Do(func() {
		if c.cfg.tokenStore != nil {
			c.lz.store = c.cfg.tokenStore
		} else {
			c.lz.store = auth.NewMemoryStore()
		}
		c.lz.otp = &auth.OTPFlow{HTTP: c.httpClient, BaseURL: c.baseURL}
		c.lz.sep10 = &auth.SEP10Flow{
			HTTP:    c.httpClient,
			BaseURL: c.baseURL,
			Signer:  stellar.NewSigner(c.cfg.stellarNetwork),
		}
		c.lz.health = health.New(c)
		c.lz.business = business.New(c)
		c.lz.application = application.New(c)
		c.lz.payments = payments.New(c)
		c.lz.reports = reports.New(c)
		// Connect requires a connector secret bound at call-time; the
		// facade returns a client preconfigured with an empty secret,
		// and WithConnectSecret wraps it. Callers can also construct
		// connect.New(c, secret) directly.
		c.lz.connect = connect.New(c, "")
	})
}

// OTP returns the Business OTP flow.
func (c *Client) OTP() *auth.OTPFlow {
	c.lazyInit()
	return c.lz.otp
}

// SEP10 returns the Connect SEP-10 flow.
func (c *Client) SEP10() *auth.SEP10Flow {
	c.lazyInit()
	return c.lz.sep10
}

// Store returns the configured token store (defaults to [auth.MemoryStore]).
func (c *Client) Store() auth.TokenStore {
	c.lazyInit()
	return c.lz.store
}

// Health returns the /healthz client.
func (c *Client) Health() *health.Client {
	c.lazyInit()
	return c.lz.health
}

// Business returns the /v1/business/* client.
func (c *Client) Business() *business.Client {
	c.lazyInit()
	return c.lz.business
}

// Application returns the /v1/application/* client.
func (c *Client) Application() *application.Client {
	c.lazyInit()
	return c.lz.application
}

// Payments returns the /v1/payments, /v1/quotes, /v1/orders client.
func (c *Client) Payments() *payments.Client {
	c.lazyInit()
	return c.lz.payments
}

// Reports returns the /v1/reports/*, /v1/market-prices client.
func (c *Client) Reports() *reports.Client {
	c.lazyInit()
	return c.lz.reports
}

// Connect returns the /v1/connect-* client configured without a
// connector secret. Use [Client.ConnectWithSecret] for a client bound to
// a specific integrator secret (the typical case).
func (c *Client) Connect() *connect.Client {
	c.lazyInit()
	return c.lz.connect
}

// ConnectWithSecret returns a new Connect client bound to the supplied
// connector secret.
func (c *Client) ConnectWithSecret(secret string) *connect.Client {
	return connect.New(c, secret)
}
