package wallet

import (
	"bytes"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

type Coin interface {
	String() string
	CurrencyCode() string
}

type CoinType uint32

const (
	Bitcoin     CoinType = 0
	Litecoin    CoinType = 1
	Dash        CoinType = 5
	Zcash       CoinType = 133
	Firo        CoinType = 136
	BitcoinCash CoinType = 145
	Ethereum    CoinType = 60

	TestnetBitcoin     = 1000000
	TestnetLitecoin    = 1000001
	TestnetZcash       = 1000133
	TestnetFiro        = 1000136
	TestnetBitcoinCash = 1000145
	TestnetEthereum    = 1000060
)

func (c *CoinType) String() string {
	switch *c {
	case Bitcoin:
		return "Bitcoin"
	case BitcoinCash:
		return "Bitcoin Cash"
	case Zcash:
		return "Zcash"
	case Firo:
		return "Firo"
	case Litecoin:
		return "Litecoin"
	case Ethereum:
		return "Ethereum"
	case TestnetBitcoin:
		return "Testnet Bitcoin"
	case TestnetBitcoinCash:
		return "Testnet Bitcoin Cash"
	case TestnetZcash:
		return "Testnet Zcash"
	case TestnetLitecoin:
		return "Testnet Litecoin"
	case TestnetEthereum:
		return "Testnet Ethereum"
	default:
		return ""
	}
}

func (c *CoinType) CurrencyCode() string {
	switch *c {
	case Bitcoin:
		return "BTC"
	case BitcoinCash:
		return "BCH"
	case Zcash:
		return "ZEC"
	case Firo:
		return "FIRO"
	case Litecoin:
		return "LTC"
	case Ethereum:
		return "ETH"
	case TestnetBitcoin:
		return "TBTC"
	case TestnetBitcoinCash:
		return "TBCH"
	case TestnetZcash:
		return "TZEC"
	case TestnetFiro:
		return "TFIRO"
	case TestnetLitecoin:
		return "TLTC"
	case TestnetEthereum:
		return "TETH"
	default:
		return ""
	}
}

type Datastore interface {
	Cfg() Cfg
	Enc() Enc
	Utxos() Utxos
	Stxos() Stxos
	Txns() Txns
	Keys() Keys
	Subscriptions() Subscriptions
}

type Cfg interface {
	PutCreationDate(date time.Time) error
	GetCreationDate() (time.Time, error)
}

type Enc interface {
	PutEncrypted(b []byte, pw string) error
	GetDecrypted(pw string) ([]byte, error)
}

type Utxos interface {
	// Put a utxo into the database
	Put(utxo Utxo) error

	// Fetch all utxos from the db
	GetAll() ([]Utxo, error)

	// Make a utxo watch-only because we have no key for it but want to watch
	// it's status. [Implemented in DB. Not used by the wallet at this time]
	SetWatchOnly(utxo Utxo) error

	// Make a utxo unspendable - we do have the key
	Freeze(utxo Utxo) error

	// Make a frozen utxo spendable again - we do have the key
	UnFreeze(utxo Utxo) error

	// Delete a utxo from the db
	Delete(utxo Utxo) error
}

type Stxos interface {
	// Put a stxo to the database
	Put(stxo Stxo) error

	// Fetch all stxos from the db
	GetAll() ([]Stxo, error)

	// Delete a stxo from the db
	Delete(stxo Stxo) error
}

type Txns interface {
	// Put a new transaction to the database
	Put(raw []byte, txid string, value int64, height int64, timestamp time.Time, watchOnly bool) error

	// Fetch a tx and it's metadata given a hash
	Get(txid string) (Txn, error)

	// Fetch all transactions from the db
	GetAll(includeWatchOnly bool) ([]Txn, error)

	// Update the height of a transaction
	UpdateHeight(txid string, height int, timestamp time.Time) error

	// Delete a transaction from the db
	Delete(txid string) error
}

// Keys provides a database interface for the wallet to:
// - Track used keys by key path
// - Manage the look ahead window.
//
// No HD keys are stored in the database. All HD keys are derived 'on the fly'
type Keys interface {
	// Put a bip32 key path into the database
	Put(hash160 []byte, keyPath KeyPath) error

	// Mark the key as used
	MarkKeyAsUsed(scriptAddress []byte) error

	// Fetch the last index for the given key purpose
	// The bool should state whether the key has been used or not
	GetLastKeyIndex(purpose KeyChange) (int, bool, error)

	// Returns the path for the given key
	GetPathForKey(scriptAddress []byte) (KeyPath, error)

	// Get a list of unused key indexes for the given purpose
	GetUnused(purpose KeyChange) ([]int, error)

	// Fetch all key paths
	GetAll() ([]KeyPath, error)

	// Get the number of unused keys following the last used key
	// for each key purpose.
	GetLookaheadWindows() map[KeyChange]int

	// Debug dump
	GetDbg() string
}

// Subscriptions is used to track ElectrumX scriptHash status change
// subscriptions made to ElectrumX nodes.
type Subscriptions interface {
	// Add a subscription.
	Put(subscription *Subscription) error

	// Return the subscription for the scriptPubKey
	Get(scriptPubKey string) (*Subscription, error)

	// Return the subscription for the electrumScripthash (not indexed)
	GetElectrumScripthash(electrumScripthash string) (*Subscription, error)

	// Return all subscriptions
	GetAll() ([]*Subscription, error)

	// Delete a subscription
	Delete(scriptPubkey string) error
}

type Subscription struct {
	// wallet subscribe watch list public key script; hex string
	PkScript string
	// electrum 1.4 protocol 'scripthash'; hex string
	ElectrumScripthash string
	// address
	Address string // encoded legacy or bech address
}

func (s *Subscription) IsEqual(alt *Subscription) bool {
	if alt == nil {
		return s == nil
	}
	if alt.PkScript != s.PkScript {
		return false
	}
	if alt.ElectrumScripthash != s.ElectrumScripthash {
		return false
	}
	if alt.Address != s.Address {
		return false
	}
	return true
}

type Utxo struct {
	// Previous txid and output index
	Op wire.OutPoint

	// Block height where this tx was confirmed, 0 for unconfirmed
	AtHeight int64

	// Coin value
	Value int64

	// Output script
	ScriptPubkey []byte

	// The primary purpose is track multisig UTXOs which must have
	// separate handling to spend. Currently unused.
	// [multisig ideas in separate branch on github]
	//
	// Keeping for some future external Tx ideas. External meaning tx's created
	// external to this wallet that we may want to ask the Electrum server to
	// watch out for..
	WatchOnly bool

	// If true this utxo has been used in a new input by software outside the
	// wallet; in an HTLC contract perhaps. Utxo will not be selected for a new
	// wallet transaction while frozen.
	//
	// It is the outside software's responsibility to set this.
	Frozen bool
}

func (utxo *Utxo) IsEqual(alt *Utxo) bool {
	if alt == nil {
		return utxo == nil
	}

	if !utxo.Op.Hash.IsEqual(&alt.Op.Hash) {
		return false
	}

	if utxo.Op.Index != alt.Op.Index {
		return false
	}

	if utxo.AtHeight != alt.AtHeight {
		return false
	}

	if utxo.Value != alt.Value {
		return false
	}

	if !bytes.Equal(utxo.ScriptPubkey, alt.ScriptPubkey) {
		return false
	}

	return true
}

type Stxo struct {
	// When it used to be a UTXO
	Utxo Utxo

	// The height at which it was spent
	SpendHeight int64

	// The tx that consumed it
	SpendTxid chainhash.Hash
}

func (stxo *Stxo) IsEqual(alt *Stxo) bool {
	if alt == nil {
		return stxo == nil
	}

	if !stxo.Utxo.IsEqual(&alt.Utxo) {
		return false
	}

	if stxo.SpendHeight != alt.SpendHeight {
		return false
	}

	if !stxo.SpendTxid.IsEqual(&alt.SpendTxid) {
		return false
	}

	return true
}

type Txn struct {
	// Transaction ID
	Txid string

	// The value relevant to the wallet
	Value int64

	// The height at which it was mined
	Height int64

	// The time the transaction was first seen
	Timestamp time.Time

	// This transaction only involves a watch only address
	WatchOnly bool

	// The number of confirmations on a transaction. This does not need to be saved in
	// the database but should be calculated when the Transactions() method is called.
	Confirmations int64

	// The state of the transaction (confirmed, unconfirmed, dead, etc). Implementations
	// have some flexibility in describing their transactions. Like confirmations, this
	// is best calculated when the Transactions() method is called.
	Status StatusCode

	// If the Status is Error the ErrorMessage should describe the problem
	ErrorMessage string

	// Raw transaction bytes
	Bytes []byte

	FromAddress string
	ToAddress   string

	Outputs []TransactionOutput
}

type StatusCode string

const (
	StatusUnconfirmed StatusCode = "UNCONFIRMED"
	StatusPending     StatusCode = "PENDING"
	StatusConfirmed   StatusCode = "CONFIRMED"
	StatusStuck       StatusCode = "STUCK"
	StatusDead        StatusCode = "DEAD"
	StatusError       StatusCode = "ERROR"
)

type KeyPath struct {
	Change KeyChange
	Index  int
}
