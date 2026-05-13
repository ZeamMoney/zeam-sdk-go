package auth

import "errors"

// Track identifies which authentication track issued a [Session].
// Business sessions are issued by the OTP/Firebase path and must NOT be
// sent to /v1/connect-* endpoints; Connect sessions are issued by SEP-10
// and must NOT be sent to Business endpoints. Mixing tracks is rejected
// at the upstream with 401 invalid_token.
type Track int

const (
	// TrackUnknown is the zero value; a Session in this track is invalid.
	TrackUnknown Track = iota
	// TrackBusiness is the Business OTP / Firebase track.
	TrackBusiness
	// TrackConnect is the Connect SEP-10 track.
	TrackConnect
)

// String returns the track's canonical name.
func (t Track) String() string {
	switch t {
	case TrackBusiness:
		return "business"
	case TrackConnect:
		return "connect"
	default:
		return "unknown"
	}
}

// ErrWrongTrack is returned when a call attempts to use a session against
// an endpoint belonging to a different track.
var ErrWrongTrack = errors.New("auth: session track mismatch (business ↔ connect)")

// RefreshRequest is the request body for POST /v1/public/auth/refresh
// and POST /v1/public/auth-connect/refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// SEP10SubmitRequest is the request body for POST /v1/public/auth-connect.
type SEP10SubmitRequest struct {
	Transaction string `json:"transaction"`
}
