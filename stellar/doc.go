// Package stellar is a narrow wrapper over the Stellar Go SDK, exposing
// only the surface the Zeam SDK needs (keypair handling, asset parsing,
// SEP-10 challenge signing). Partners should never need to import the
// upstream Stellar SDK directly.
//
// The wrapper pins the default network passphrase to the Stellar Public
// Main Network. Callers can override via the [Signer] constructor when
// targeting testnet or a futurenet environment.
package stellar

// PublicNetworkPassphrase is the passphrase used to sign transactions for
// the Stellar Public Main Network. Matches the value used by the gateway's
// SEP-10 proxy.
const PublicNetworkPassphrase = "Public Global Stellar Network ; September 2015"

// TestnetNetworkPassphrase targets the Stellar Test SDF Network.
const TestnetNetworkPassphrase = "Test SDF Network ; September 2015"
