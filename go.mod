module github.com/ZeamMoney/zeam-sdk-go

go 1.26.2

require (
	github.com/google/uuid v1.6.0
	github.com/stellar/go-stellar-sdk v0.5.0
	golang.org/x/sync v0.19.0
)

require (
	github.com/klauspost/compress v1.17.6 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/stellar/go-xdr v0.0.0-20260312225820-cc2b0611aabf // indirect
	golang.org/x/exp v0.0.0-20231006140011-7918f672742d // indirect
)

// Wired during Phase 1 when the stellar/ wrapper activates the upstream keypair.
// require github.com/stellar/go-stellar-sdk v0.5.0
