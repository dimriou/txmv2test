# TXMv2 Test

## Overview
This is a test repo for TXMv2. Clone this repo, and run `go mod tidy` to install the necessary dependencies.
Go to `main.go` and update the following variables:
- `rpc`: RPC endpoint of the preferred network for this test (recommended network is Ethereum Sepolia).
- `privateKeyString`: your private key without the 0x prefx.
- `fromAddressString`: the address the private key owns.

*Note1: you'll need some funding for the address you're going to use.*

*Note2: you can also update the `GasEstimator` configs if you want to but it's not necessary as the default
values are already set.*

This test repo imports Transaction Manager v2 from Core node (`github.com/smartcontractkit/chainlink/v2/core/chains/evm/txm`).

## Required Components
To run the Transaction Manager we initialize a few required components first:
- Logger
- Client
- Gas Estimator/Attempt Builder
- Keystore
- InMemory Layer

## How to use
In the test case, there is a section called `Add your logic here`. Inside that section you can implement your logic.
To help with the process, there is already a dummy use case in which we create a simple transaction request and we
send it to the 0x00 address. There two core methods to interact with TXMv2 are `CreateTransaction` and `Trigger`.
The first one converts a transaction request into an unstarted transaction that eventually will be picked up by the
TXM. The latter is used when we want to instantly trigger the TXM, in case we don't want to wait for the next
broadcasting cycle.

Finally, another section is exists, called `Confirmation Loop`. This section is only necessary for this test, since
the TXM needs to be closed at the end of it. It periodically checks if all our transactions were confirmed by the TXM
and if they did, it cleans up and exits.