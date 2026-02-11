module github.com/vadimzhukck/privy-sdk-go/chains/sui

go 1.23.0

toolchain go1.24.7

require (
	github.com/vadimzhukck/privy-sdk-go v0.0.0
	golang.org/x/crypto v0.37.0
)

require golang.org/x/sys v0.32.0 // indirect

replace github.com/vadimzhukck/privy-sdk-go => ../..
