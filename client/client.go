package client

///////////////////////////////// Client interface ///////////////////////////
//
//	architecture
//
//	   Client
//
//	     /\
//	 (controller)
//	   /    \
//	  /      \
//	 /        \
//
// Wallet      X
//
// The client interface describes the behaviors of the client controller.
// It is implemented for each coin asset client.

import (
	"context"

	"github.com/bisoncraft/go-electrum-client/electrumx"
	"github.com/bisoncraft/go-electrum-client/wallet"
)

type NodeType int

const (
	// ElectrumX Server(s)
	SingleNode NodeType = iota
	MultiNode  NodeType = 1
)

const (
	// Electrum Wallet
	GAP_LIMIT = 10
)

type ElectrumClient interface {
	Start(ctx context.Context) error
	Stop()
	//
	GetConfig() *ClientConfig
	GetWallet() wallet.ElectrumWallet
	GetX() electrumx.ElectrumX
	//
	RegisterTipChangeNotify() (<-chan int64, error)
	UnregisterTipChangeNotify()
	//
	CreateWallet(pw string) error
	LoadWallet(pw string) error
	RecreateWallet(ctx context.Context, pw, mnenomic string) error
	//
	SyncWallet(ctx context.Context) error
	RescanWallet(ctx context.Context) error
	ImportAndSweep(ctx context.Context, keyPairs []string) error
	//
	// Subset of electrum-like methods
	Tip() int64
	Synced() bool
	GetBlockHeader(height int64) (*electrumx.ClientBlockHeader, error)
	GetBlockHeaders(startHeight, count int64) ([]*electrumx.ClientBlockHeader, error)
	Spend(pw string, amount int64, toAddress string, feeLevel wallet.FeeLevel) (int, string, string, error)
	GetPrivKeyForAddress(pw, addr string) (string, error)
	ListUnspent() ([]wallet.Utxo, error)
	ListConfirmedUnspent() ([]wallet.Utxo, error)
	ListFrozenUnspent() ([]wallet.Utxo, error)
	FreezeUTXO(txid string, out uint32) error
	UnfreezeUTXO(txid string, out uint32) error
	UnusedAddress(ctx context.Context) (string, error)
	ChangeAddress(ctx context.Context) (string, error)
	ValidateAddress(addr string) (bool, bool, error)
	SignTx(pw string, txBytes []byte) ([]byte, error)
	GetWalletTx(txid string) (int, bool, []byte, error)
	GetWalletSpents() ([]wallet.Stxo, error)
	Balance() (int64, int64, int64, error)

	// adapt and pass thru to electrumx
	Broadcast(ctx context.Context, rawTx []byte) (string, error)
	FeeRate(ctx context.Context, confTarget int64) (int64, error)

	//pass thru directly to electrumx
	GetTransaction(ctx context.Context, txid string) (*electrumx.GetTransactionResult, error)
	GetRawTransaction(ctx context.Context, txid string) ([]byte, error)
	GetAddressHistory(ctx context.Context, addr string) (electrumx.HistoryResult, error)
	GetAddressUnspent(ctx context.Context, addr string) (electrumx.ListUnspentResult, error)

	//coin specific extra for server protocol - use dummy method for non-implmenting coins.
	// firo EXX addresses
}
