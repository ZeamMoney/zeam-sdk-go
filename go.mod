module github.com/ZeamMoney/zeam-sdk-go

go 1.26.2

require (
	github.com/google/uuid v1.6.0
	golang.org/x/sync v0.19.0
)

// Wired during Phase 1 when the stellar/ wrapper activates the upstream keypair.
// require github.com/stellar/go-stellar-sdk v0.5.0
