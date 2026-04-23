package recipes

import (
	"context"
	"errors"
	"fmt"

	"github.com/ZeamMoney/zeam-sdk-go/auth"
)

// OTPHint carries the non-sensitive fields returned by the OTP-request
// step so partner UX can show a masked destination and expiry.
type OTPHint struct {
	MaskedDestination string
	RequestID         string
}

// LoginOTPInput is the input to [LoginOTP].
type LoginOTPInput struct {
	// MobileNumber in E.164 form (+27…).
	MobileNumber string
	// AskCode is the partner-provided UX callback. It receives the hint
	// after the OTP has been delivered and must return the code the end
	// user typed.
	AskCode func(ctx context.Context, hint OTPHint) (string, error)
	// Subject is an optional human-meaningful tag stored on the Session.
	Subject string
}

// LoginOTPClient is the minimal auth surface LoginOTP needs. The top-
// level package provides an adapter.
type LoginOTPClient interface {
	OTP() *auth.OTPFlow
	Store() auth.TokenStore
}

// LoginOTP runs the Business OTP flow end-to-end. On success it returns
// a [*auth.Session] on TrackBusiness and persists it in the client's
// [auth.TokenStore].
func LoginOTP(ctx context.Context, c LoginOTPClient, in LoginOTPInput) (*auth.Session, error) {
	if c == nil || c.OTP() == nil || c.Store() == nil {
		return nil, errors.New("recipes: LoginOTP requires a configured client")
	}
	if in.MobileNumber == "" {
		return nil, errors.New("recipes: mobile number is required")
	}
	if in.AskCode == nil {
		return nil, errors.New("recipes: AskCode callback is required")
	}

	challenge, err := c.OTP().RequestOTP(ctx, auth.OTPRequest{MobileNumber: in.MobileNumber})
	if err != nil {
		return nil, fmt.Errorf("recipes: request OTP: %w", err)
	}
	code, err := in.AskCode(ctx, OTPHint{
		MaskedDestination: challenge.MaskedDestination,
		RequestID:         challenge.RequestID,
	})
	if err != nil {
		return nil, fmt.Errorf("recipes: AskCode: %w", err)
	}
	sess, err := c.OTP().VerifyOTP(ctx, auth.OTPVerify{RequestID: challenge.RequestID, Code: code}, in.Subject)
	if err != nil {
		return nil, fmt.Errorf("recipes: verify OTP: %w", err)
	}
	if err := c.Store().Put(ctx, sess); err != nil {
		sess.Erase()
		return nil, fmt.Errorf("recipes: persist session: %w", err)
	}
	return sess, nil
}
