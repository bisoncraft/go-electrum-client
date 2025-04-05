# go-electrum-client

__Goele__ - Golang Electrum Client

## Electrum client wallet in Go

- Electrum client multi coin wallet library
- ElectrumX chain server interface
- No GUI

## Purpose

- Goele is a simplified functional version of an electrum wallet.

- Goele is golang code intended to be used directly by other golang projects; for example a lite trading wallet.

- Goele wallets __do__ have a simple RPC Server to call some of the python console-like methods such as ListUnspent, Spend & Broadcast, etc. But this is for testing only.

## Development example

The __goele__ example app can be found at `cmd/goele-btc/`. This is the tool I have been using for functional testing.

If running against regtest network then it is dependent upon the _BTC ElectrumX RegTest Simnet Harness_ found in `client/btc/test_harness`.

It will also run against btc testnet.

## Database Implementations

Goele uses `Bolt DB` by default. It can also use `sqlite3`.

There is a utility `bd` to dump the goele structures in `wallet.bdb`.

For `sqlite3` has it's own viewing for tables and records.

## BIP39

Goele uses BIP39 for wallet seeds. ELectrum uses it's own seed derivation.

## Rescan

There is code to rescan for wallet transactions when re-creating a wallet from seed.

The assumptions about GAP_LIMIT are _experimental_ so please do not use for mainnet.
Tested on regtest and testnet.
