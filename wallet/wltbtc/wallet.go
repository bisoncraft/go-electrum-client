package wltbtc

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bisoncraft/go-electrum-client/wallet"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcwallet/wallet/txrules"
	"github.com/tyler-smith/go-bip39"
)

//////////////////////////////////////////////////////////////////////////////
//	ElectrumWallet

// BtcElectrumWallet implements ElectrumWallet
var _ = wallet.ElectrumWallet(&BtcElectrumWallet{})

const WalletVersion = "0.1.0"

var ErrEmptyPassword = errors.New("empty password")

type BtcElectrumWallet struct {
	params *chaincfg.Params

	feeProvider *wallet.FeeProvider

	repoPath string

	storageManager      *StorageManager
	txstore             *TxStore
	keyManager          *KeyManager
	subscriptionManager *SubscriptionManager

	mutex *sync.RWMutex

	creationDate time.Time

	blockchainTip int64

	running bool
}

// NewBtcElectrumWallet mskes new wallet with a new seed. The Mnemonic should
// be saved offline by the user.
func NewBtcElectrumWallet(config *wallet.WalletConfig, pw string) (*BtcElectrumWallet, error) {
	if pw == "" {
		return nil, ErrEmptyPassword
	}

	ent, err := bip39.NewEntropy(128)
	if err != nil {
		return nil, err
	}
	mnemonic, err := bip39.NewMnemonic(ent)
	if err != nil {
		return nil, err
	}

	seed := bip39.NewSeed(mnemonic, "")

	return makeBtcElectrumWallet(config, pw, seed)
}

// RecreateElectrumWallet makes new wallet with a mnenomic seed from an existing wallet.
// pw does not need to be the same as the old wallet
func RecreateElectrumWallet(config *wallet.WalletConfig, pw, mnemonic string) (*BtcElectrumWallet, error) {
	if pw == "" {
		return nil, ErrEmptyPassword
	}
	seed, err := bip39.NewSeedWithErrorChecking(mnemonic, "")
	if err != nil {
		return nil, err
	}

	return makeBtcElectrumWallet(config, pw, seed)
}

func makeBtcElectrumWallet(config *wallet.WalletConfig, pw string, seed []byte) (*BtcElectrumWallet, error) {

	mPrivKey, err := hdkeychain.NewMaster(seed, config.Params)
	if err != nil {
		return nil, err
	}
	mPubKey, err := mPrivKey.Neuter()
	if err != nil {
		return nil, err
	}
	w := &BtcElectrumWallet{
		repoPath:     config.DataDir,
		params:       config.Params,
		creationDate: time.Now(),
		feeProvider:  wallet.DefaultFeeProvider(),
		mutex:        new(sync.RWMutex),
	}

	sm := NewStorageManager(config.DB.Enc(), config.Params)
	sm.store.Version = "0.1"
	sm.store.Xprv = mPrivKey.String()
	sm.store.Xpub = mPubKey.String()
	sm.store.ShaPw = chainhash.HashB([]byte(pw))
	if config.StoreEncSeed {
		sm.store.Seed = bytes.Clone(seed)
	}
	err = sm.Put(pw)
	if err != nil {
		return nil, err
	}
	w.storageManager = sm

	w.keyManager, err = NewKeyManager(config.DB.Keys(), w.params, mPrivKey)
	mPrivKey.Zero()
	mPubKey.Zero()
	if err != nil {
		return nil, err
	}

	w.txstore, err = NewTxStore(w.params, config.DB, w.keyManager)
	if err != nil {
		return nil, err
	}

	w.subscriptionManager = NewSubscriptionManager(config.DB.Subscriptions(), w.params)

	err = config.DB.Cfg().PutCreationDate(w.creationDate)
	if err != nil {
		return nil, err
	}

	return w, nil
}

func LoadBtcElectrumWallet(config *wallet.WalletConfig, pw string) (*BtcElectrumWallet, error) {
	if pw == "" {
		return nil, ErrEmptyPassword
	}

	return loadBtcElectrumWallet(config, pw)
}

func loadBtcElectrumWallet(config *wallet.WalletConfig, pw string) (*BtcElectrumWallet, error) {

	sm := NewStorageManager(config.DB.Enc(), config.Params)

	err := sm.Get(pw)
	if err != nil {
		return nil, err
	}

	mPrivKey, err := hdkeychain.NewKeyFromString(sm.store.Xprv)
	if err != nil {
		return nil, err
	}

	w := &BtcElectrumWallet{
		repoPath:       config.DataDir,
		storageManager: sm,
		params:         config.Params,
		feeProvider:    wallet.DefaultFeeProvider(),
		mutex:          new(sync.RWMutex),
	}

	w.keyManager, err = NewKeyManager(config.DB.Keys(), w.params, mPrivKey)
	mPrivKey.Zero()
	if err != nil {
		return nil, err
	}

	w.txstore, err = NewTxStore(w.params, config.DB, w.keyManager)
	if err != nil {
		return nil, err
	}

	w.subscriptionManager = NewSubscriptionManager(config.DB.Subscriptions(), w.params)

	w.creationDate, err = config.DB.Cfg().GetCreationDate()
	if err != nil {
		return nil, err
	}

	return w, nil
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//
// API
//
//////////////

// /////////////////////
// start interface impl.

func (w *BtcElectrumWallet) Start() {
	w.running = true
}

func (w *BtcElectrumWallet) CreationDate() time.Time {
	return w.creationDate
}

func (w *BtcElectrumWallet) Params() *chaincfg.Params {
	return w.params
}

func (w *BtcElectrumWallet) CurrencyCode() string {
	if w.params.Name == chaincfg.MainNetParams.Name {
		return "btc"
	} else {
		return "tbtc"
	}
}

func (w *BtcElectrumWallet) IsDust(amount int64) bool {
	// This is a per mempool policy thing .. < 1000 sats for now
	return btcutil.Amount(amount) < txrules.DefaultRelayFeePerKb
}

// GetAddress gets an address given a KeyPath.
// It is used for Rescan and has no concept of gap-limit. It is expected that
// keys made here are just temporarily used to generate addresses for rescan.
func (w *BtcElectrumWallet) GetAddress(kp *wallet.KeyPath /*, addressType*/) (btcutil.Address, error) {
	key, err := w.keyManager.generateChildKey(kp.Change, uint32(kp.Index))
	if err != nil {
		return nil, err
	}
	address, err := key.Address(w.params)
	key.Zero()
	if err != nil {
		return nil, err
	}
	script := address.ScriptAddress()
	segwitAddress, err := btcutil.NewAddressWitnessPubKeyHash(script, w.params)
	if err != nil {
		return nil, err
	}
	return segwitAddress, nil
}

func (w *BtcElectrumWallet) GetUnusedAddress(purpose wallet.KeyChange) (btcutil.Address, error) {
	key, err := w.keyManager.GetUnusedKey(purpose)
	if err != nil {
		return nil, nil
	}
	address, err := key.Address(w.params)
	key.Zero()
	if err != nil {
		return nil, nil
	}
	script := address.ScriptAddress()
	segwitAddress, swerr := btcutil.NewAddressWitnessPubKeyHash(script, w.params)
	if swerr != nil {
		return nil, swerr
	}
	return segwitAddress, nil
}

// For receiving simple payments from legacy wallets only!
func (w *BtcElectrumWallet) GetUnusedLegacyAddress() (btcutil.Address, error) {
	key, err := w.keyManager.GetUnusedKey(wallet.RECEIVING)
	if err != nil {
		return nil, nil
	}
	addrP2PKH, err := key.Address(w.params)
	key.Zero()
	if err != nil {
		return nil, nil
	}
	return addrP2PKH, nil
}

func (w *BtcElectrumWallet) GetPrivKeyForAddress(pw string, address btcutil.Address) (string, error) {
	if ok := w.storageManager.IsValidPw(pw); !ok {
		return "", errors.New("invalid password")
	}
	hdKey, err := w.keyManager.GetKeyForScript(address.ScriptAddress())
	if err != nil {
		return "", err
	}
	privKey, err := hdKey.ECPrivKey()
	if err != nil {
		return "", err
	}
	wif, err := btcutil.NewWIF(privKey, w.params, true)
	if err != nil {
		return "", err
	}
	return wif.String(), nil
}

// Marks the address as used (involved in at least one transaction)
func (w *BtcElectrumWallet) MarkAddressUsed(address btcutil.Address) error {
	return w.txstore.Keys().MarkKeyAsUsed(address.ScriptAddress())
}

func (w *BtcElectrumWallet) DecodeAddress(addr string) (btcutil.Address, error) {
	return btcutil.DecodeAddress(addr, w.params)
}

func (w *BtcElectrumWallet) ScriptToAddress(script []byte) (btcutil.Address, error) {
	_, addresses, _, err := txscript.ExtractPkScriptAddrs(script, w.params)
	if err != nil {
		return &btcutil.AddressPubKeyHash{}, err
	}
	if len(addresses) == 0 {
		return &btcutil.AddressPubKeyHash{}, errors.New("unknown script")
	}
	return addresses[0], nil
}

func (w *BtcElectrumWallet) AddressToScript(address btcutil.Address) ([]byte, error) {
	return txscript.PayToAddrScript(address)
}

func (w *BtcElectrumWallet) AddSubscription(subcription *wallet.Subscription) error {
	return w.subscriptionManager.Put(subcription)
}

func (w *BtcElectrumWallet) RemoveSubscription(scriptPubKey string) {
	w.subscriptionManager.Delete(scriptPubKey)
}

func (w *BtcElectrumWallet) GetSubscription(scriptPubKey string) (*wallet.Subscription, error) {
	return w.subscriptionManager.Get(scriptPubKey)
}

func (w *BtcElectrumWallet) GetSubscriptionForElectrumScripthash(electrumScripthash string) (*wallet.Subscription, error) {
	return w.subscriptionManager.GetElectrumScripthash(electrumScripthash)
}

func (w *BtcElectrumWallet) ListSubscriptions() ([]*wallet.Subscription, error) {
	return w.subscriptionManager.GetAll()
}

func (w *BtcElectrumWallet) HasAddress(address btcutil.Address) bool {
	_, err := w.keyManager.GetKeyForScript(address.ScriptAddress())
	return err == nil
}

func (w *BtcElectrumWallet) ListAddresses() []btcutil.Address {
	keys := w.keyManager.GetKeys()
	addresses := []btcutil.Address{}
	for _, k := range keys {
		address, err := k.Address(w.params)
		if err != nil {
			continue
		}
		addresses = append(addresses, address)
	}
	return addresses
}

func (w *BtcElectrumWallet) IsMine(queryAddress btcutil.Address) bool {
	ourAddresses := w.ListAddresses()
	for _, address := range ourAddresses {
		if bytes.Equal(address.ScriptAddress(), queryAddress.ScriptAddress()) {
			return true
		}
	}
	return false
}

func (w *BtcElectrumWallet) Balance() (int64, int64, int64, error) {

	isStxoConfirmed := func(utxo wallet.Utxo, stxos []wallet.Stxo) bool {
		for _, stxo := range stxos {
			if stxo.Utxo.WatchOnly {
				continue
			}
			// utxo is prevout of stxo SpendTxid
			if stxo.SpendTxid.IsEqual(&utxo.Op.Hash) {
				return stxo.SpendHeight > 0
			}
			if stxo.Utxo.IsEqual(&utxo) {
				return stxo.Utxo.AtHeight > 0
			}
		}
		// no stxo so no spend
		return false
	}

	confirmed := int64(0)
	unconfirmed := int64(0)
	locked := int64(0)
	utxos, err := w.txstore.Utxos().GetAll()
	if err != nil {
		return 0, 0, 0, err
	}
	stxos, err := w.txstore.Stxos().GetAll()
	if err != nil {
		return 0, 0, 0, err
	}
	for _, utxo := range utxos {
		if utxo.WatchOnly {
			continue
		}
		if utxo.Frozen {
			locked += utxo.Value
		}
		if utxo.AtHeight > 0 {
			confirmed += utxo.Value
			continue
		}
		// height 0 so possibly spent in the mempool?
		if isStxoConfirmed(utxo, stxos) {
			confirmed += utxo.Value
			continue
		}
		unconfirmed += utxo.Value
	}
	return confirmed, unconfirmed, locked, nil
}

func (w *BtcElectrumWallet) ListTransactions() ([]wallet.Txn, error) {
	return w.txstore.Txns().GetAll(false)
}

func (w *BtcElectrumWallet) HasTransaction(txid string) (bool, *wallet.Txn) {
	txn, err := w.txstore.Txns().Get(txid)
	// errors only for 'no rows in rowset' (sqlite)
	if err != nil {
		return false, nil
	}
	return true, &txn
}

func (w *BtcElectrumWallet) GetTransaction(txid string) (*wallet.Txn, error) {
	txn, err := w.txstore.Txns().Get(txid)
	if err != nil {
		return nil, fmt.Errorf("no such transaction")
	}
	return &txn, nil
}

// Return the calculated confirmed txids and heights for an address in this
// wallet. Currently we get this info from the Node.
func (w *BtcElectrumWallet) GetWalletAddressHistory(address btcutil.Address) ([]wallet.AddressHistory, error) {
	var history []wallet.AddressHistory
	//TODO: if need a seperate and fast source of wallet only history.
	return history, nil
}

// Add a transaction to the database
func (w *BtcElectrumWallet) AddTransaction(tx *wire.MsgTx, height int64, timestamp time.Time) error {
	_, err := w.txstore.AddTransaction(tx, height, timestamp)
	return err
}

// List all unspent outputs in the wallet
func (w *BtcElectrumWallet) ListUnspent() ([]wallet.Utxo, error) {
	return w.txstore.Utxos().GetAll()
}

func (w *BtcElectrumWallet) ListConfirmedUnspent() ([]wallet.Utxo, error) {
	utxos, err := w.txstore.Utxos().GetAll()
	if err != nil {
		return nil, err
	}
	var confirmed = make([]wallet.Utxo, 0)
	for _, utxo := range utxos {
		if utxo.AtHeight > 0 {
			confirmed = append(confirmed, utxo)
		}
	}
	return confirmed, nil
}

func (w *BtcElectrumWallet) ListFrozenUnspent() ([]wallet.Utxo, error) {
	utxos, err := w.txstore.Utxos().GetAll()
	if err != nil {
		return nil, err
	}
	var frozen = make([]wallet.Utxo, 0)
	for _, utxo := range utxos {
		if utxo.Frozen {
			frozen = append(frozen, utxo)
		}
	}
	return frozen, nil
}

// List all spent
func (w *BtcElectrumWallet) ListSpent() ([]wallet.Stxo, error) {
	return w.txstore.Stxos().GetAll()
}

func (w *BtcElectrumWallet) FreezeUTXO(op *wire.OutPoint) error {
	utxos, err := w.txstore.Utxos().GetAll()
	if err != nil {
		return err
	}
	for _, utxo := range utxos {
		if utxo.Op.Hash.IsEqual(&op.Hash) && utxo.Op.Index == op.Index {
			return w.txstore.Utxos().Freeze(utxo)
		}
	}
	return errors.New("utxo not found")
}

func (w *BtcElectrumWallet) UnFreezeUTXO(op *wire.OutPoint) error {
	utxos, err := w.txstore.Utxos().GetAll()
	if err != nil {
		return err
	}
	for _, utxo := range utxos {
		if utxo.Op.Hash.IsEqual(&op.Hash) && utxo.Op.Index == op.Index {
			return w.txstore.Utxos().UnFreeze(utxo)
		}
	}
	return errors.New("utxo not found")
}

// Update the wallet's view of the blockchain
func (w *BtcElectrumWallet) UpdateTip(newTip int64) {
	w.blockchainTip = newTip
}

/////////////////////////////
// implementations in send.go

// // Send bitcoins to an external wallet
// Spend(amount int64, toAddress btcutil.Address, feeLevel wallet.FeeLevel) ([]byte, error) {

// // Calculates the estimated size of the transaction and returns the total fee
// // for the given feePerByte
// EstimateFee(ins []wallet.TransactionInput, outs []wallet.TransactionOutput, feePerByte uint64) int64

//////////////////////////////
// implementations in sweep.go

// SweepAddress(utxos []wallet.Utxo, address btcutil.Address, key *hdkeychain.ExtendedKey, redeemScript *[]byte, feeLevel wallet.FeeLevel) ([]byte, error)
// Build a transaction that sweeps all coins from a non-wallet private key
// SweepCoins(ins []TransactionInput, feeLevel FeeLevel) (int, *wire.MsgTx, error)

////////////////////////////////
// implementations in bumpfee.go

// CPFP logic - No rbf and never will be here!
// func (w *BtcElectrumWallet) BumpFee(txid *chainhash.Hash) (*chainhash.Hash, error)

// end interface impl
/////////////////////
