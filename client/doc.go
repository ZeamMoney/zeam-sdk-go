// Package client houses the typed 1:1 wrappers for gateway endpoints.
// Each sub-package mirrors a gateway route group so a gateway change
// touches at most one SDK sub-package.
//
// All sub-packages take an auth.Session whose [auth.Track] matches their
// protected surface:
//
//   - client/business, client/application, client/payments
//     require [auth.TrackBusiness].
//   - client/connect requires [auth.TrackConnect].
//   - client/health is unauthenticated.
//
// A call against a mis-matched track returns [auth.ErrWrongTrack] without
// contacting the gateway.
package client
