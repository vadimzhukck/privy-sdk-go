module github.com/vadimzhukck/privy-sdk-go/chains/aptos

go 1.23.0

toolchain go1.24.7

require (
	github.com/aptos-labs/aptos-go-sdk v1.7.0
	github.com/vadimzhukck/privy-sdk-go v0.0.0
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/coder/websocket v1.8.12 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hasura/go-graphql-client v0.13.1 // indirect
	github.com/hdevalence/ed25519consensus v0.2.0 // indirect
	golang.org/x/crypto v0.37.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
)

replace github.com/vadimzhukck/privy-sdk-go => ../..
