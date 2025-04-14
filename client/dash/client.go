package dash

import (
	"context"
	"errors"
	"os"
	"path"
	"sync"

	"github.com/bisoncraft/go-electrum-client/client"
	"github.com/bisoncraft/go-electrum-client/electrumx"
	"github.com/bisoncraft/go-electrum-client/electrumx/elxdash"
	"github.com/bisoncraft/go-electrum-client/wallet"
	"github.com/bisoncraft/go-electrum-client/wallet/bdb"
	"github.com/bisoncraft/go-electrum-client/wallet/db"
	"github.com/bisoncraft/go-electrum-client/wallet/wltdash"
)

// DashElectrumClient - implements ElectrumClient interface
type DashElectrumClient struct {
	// Cancel is the cancel func for the goele context
	Cancel context.CancelFunc
	// The Goele configuration
	ClientConfig *client.ClientConfig
	// Goele wallet
	Wallet wallet.ElectrumWallet
	// Interface tp ElectrumX servers for a coin network
	X electrumx.ElectrumX
	// Receive tip change notify channel from electrumx
	rcvTipChangeNotify <-chan int64
	// Forward tip change notify to external user if registered
	sendTipChangeNotify    chan int64
	sendTipChangeNotifyMtx sync.RWMutex
}

func NewDashElectrumClient(cfg *client.ClientConfig) client.ElectrumClient {
	ec := DashElectrumClient{
		Cancel:              nil,
		ClientConfig:        cfg,
		Wallet:              nil,
		X:                   nil,
		rcvTipChangeNotify:  nil,
		sendTipChangeNotify: nil,
	}
	return &ec
}

var _ = client.ElectrumClient(&DashElectrumClient{})

//////////////////////////////////////////////////////////////////////////////
// Interface impl
/////////////////

func (ec *DashElectrumClient) GetConfig() *client.ClientConfig {
	return ec.ClientConfig
}

func (ec *DashElectrumClient) GetWallet() wallet.ElectrumWallet {
	return ec.Wallet
}

func (ec *DashElectrumClient) GetX() electrumx.ElectrumX {
	return ec.X
}

func (ec *DashElectrumClient) walletExists() bool {
	cfg := ec.ClientConfig
	datadir := ec.ClientConfig.DataDir
	var walletName = ""
	switch cfg.DbType {
	case client.DbTypeBolt:
		walletName = "wallet.bdb"
	case client.DbTypeSqlite:
		walletName = "wallet.db"
	}
	if _, err := os.Stat(path.Join(datadir, walletName)); err != nil {
		return false
	}
	return true
}

func (ec *DashElectrumClient) getDatastore() error {
	cfg := ec.ClientConfig
	switch cfg.DbType {
	case client.DbTypeBolt:
		// Select a bbolt wallet datastore - false = RW database
		boltDatastore, err := bdb.Create(cfg.DataDir, false)
		if err != nil {
			return err
		}
		cfg.DB = boltDatastore
	case client.DbTypeSqlite:
		// Select a sqlite wallet datastore
		sqliteDatastore, err := db.Create(cfg.DataDir)
		if err != nil {
			return err
		}
		cfg.DB = sqliteDatastore
	default:
		return errors.New("unknown database type")
	}
	return nil
}

// createElectrumXInterface creates an ElectrumXInterface
func (ec *DashElectrumClient) createElectrumXInterface() error {
	elxCfg := ec.GetConfig().MakeElectrumXConfig()
	n, err := elxdash.NewElectrumXInterface(elxCfg)
	if err != nil {
		return err
	}
	ec.X = n
	return nil
}

// client interface implementation

func (ec *DashElectrumClient) Start(parentCtx context.Context) error {
	goeleCtx, goeleCancel := context.WithCancel(parentCtx)
	ec.Cancel = goeleCancel
	err := ec.createElectrumXInterface()
	if err != nil {
		return err
	}
	err = ec.X.Start(goeleCtx)
	if err != nil {
		return err
	}
	ec.rcvTipChangeNotify, err = ec.X.GetTipChangeNotify()
	if err != nil {
		return err
	}
	go ec.tipChange(goeleCtx)
	return nil
}

func (ec *DashElectrumClient) Stop() {
	ec.Cancel()
}

// CreateWallet makes a new wallet with a new seed. The password is to encrypt
// stored xpub, xprv and other sensitive data.
func (ec *DashElectrumClient) CreateWallet(pw string) error {
	if ec.walletExists() {
		return errors.New("wallet already exists")
	}
	err := ec.getDatastore()
	if err != nil {
		return err
	}

	walletCfg := ec.ClientConfig.MakeWalletConfig()

	w, err := wltdash.NewDashElectrumWallet(walletCfg, pw)
	if err != nil {
		return err
	}
	ec.Wallet = w
	return nil
}

// RecreateWallet recreates a wallet from an existing mnemonic seed.
// The password is to encrypt the stored xpub, xprv and other sensitive data
// and can be different from the original wallet's password.
func (ec *DashElectrumClient) RecreateWallet(ctx context.Context, pw, mnenomic string) error {
	if ec.walletExists() {
		//TODO: should we backup any wallet file that exists
		return errors.New("wallet already exists")
	}
	err := ec.getDatastore()
	if err != nil {
		return err
	}
	walletCfg := ec.ClientConfig.MakeWalletConfig()
	ec.Wallet, err = wltdash.RecreateElectrumWallet(walletCfg, pw, mnenomic)
	if err != nil {
		return err
	}
	// Do a rescan because alhough we have a wallet structure with a keychain
	// we do not have any transaction history
	return ec.RescanWallet(ctx)
}

// LoadWallet loads an existing wallet. The password is required to decrypt
// the stored xpub, xprv and other sensitive data
func (ec *DashElectrumClient) LoadWallet(pw string) error {
	if !ec.walletExists() {
		return errors.New("cannot find wallet")
	}
	err := ec.getDatastore()
	if err != nil {
		return err
	}
	walletCfg := ec.ClientConfig.MakeWalletConfig()
	w, err := wltdash.LoadDashElectrumWallet(walletCfg, pw)
	if err != nil {
		return err
	}
	ec.Wallet = w
	return nil
}

// Interface methods in blockchain.go
//
// Tip() (int64, bool)
// RegisterTipChangeNotify(tipChange func(height int64)) error
// UnegisterTipChangeNotify()
// GetBlockHeader(height int64) *wire.BlockHeader
// GetBlockHeaders(startHeight, count int64) ([]*wire.BlockHeader, error)

// Interface methods in client_wallet.go
//
// Spend(amount int64, toAddress string, feeLevel wallet.FeeLevel, broadcast bool) (string, string, error)
// GetPrivKeyForAddress(pw, addr string) (string, error)
// Broadcast(ctx context.Context, rawTx []byte) (string, error)
// FeeRate(ctx context.Context, confTarget int64) (int64, error)
// ListUnspent() ([]wallet.Utxo, error)
// UnusedAddress(ctx context.Context) (string, error)
// ChangeAddress(ctx context.Context) (string, error)
// Balance() (int64, int64, error)
// FreezeUTXO((txid string, out uint32) error
// UnFreezeUTXO((txid string, out uint32) error
// GetWalletTx(txid string) (int, bool, []byte, error)
// GetWalletSpents() ([]wallet.Stxo, error)

// Interface methods in client_node.go
//
// GetTransaction(ctx context.Context, txid string) (*electrumx.GetTransactionResult, error)
// GetRawTransaction(ctx context.Context,txid string) ([]byte, error)
// GetAddressHistory(ctx context.Context, addr string) (electrumx.HistoryResult, error)
// GetAddressUnspent(ctx context.Context, addr string) (electrumx.ListUnspentResult, error)
//
//////////////////////////////////////////////////////////////////////////////
