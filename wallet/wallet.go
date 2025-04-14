package wallet

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

type WalletConfig struct {
	// Coin ticker to id the coin
	Coin string

	// The blockchain, btc, dash, etc
	CoinType CoinType

	// mainnet, testnet or regtest
	NetType string

	// Network parameters - can this be generalized for all coins?
	Params *chaincfg.Params

	// Store the seed in encrypted storage
	StoreEncSeed bool

	DbType string

	// Location of the data directory
	DataDir string

	// An implementation of the Datastore interface
	DB Datastore

	// The default fee-per-byte for each level
	LowFee    int64
	MediumFee int64
	HighFee   int64

	// The highest allowable fee-per-byte
	MaxFee int64

	// If not testing do not overwrite existing wallet files
	Testing bool
}

type ElectrumWallet interface {
	// Start the wallet
	Start()

	// Return the creation date of the wallet
	CreationDate() time.Time

	// Return the network parameters
	Params() *chaincfg.Params

	// Returns the type of crytocurrency this wallet implements
	CurrencyCode() string

	// Check if this amount is considered dust < 1000 sats/equivalent for now
	IsDust(amount int64) bool

	// GetAddress gets an address given a KeyPath. It is used for Rescan
	GetAddress(kp *KeyPath) (btcutil.Address, error)

	// GetUnusedAddress returns an address suitable for receiving payments.
	// `purpose` specifies whether the address should be internal or external.
	// This function will return the same address so long as that address is
	// not invloved in a transaction. Whenever the returned address has been
	// used in a broadcasted tx this function should start returning a new,
	// unused address.
	GetUnusedAddress(purpose KeyChange) (btcutil.Address, error)

	// GetUnusedLegacyAddress returns an address suitable for receiving payments
	// from legacy wallets, exchanges, etc. It will only give out external addr-
	// esses for receiving funds; not change addresses.
	GetUnusedLegacyAddress() (btcutil.Address, error)

	// GetPrivKeyForAddress gets the wallet private key-pair as a WIF given an
	// wallet address and the wallet password.
	GetPrivKeyForAddress(pw string, address btcutil.Address) (string, error)

	// Marks the address as used (involved in at least one transaction)
	MarkAddressUsed(address btcutil.Address) error

	// CreateNewAddress returns a new, never-before-returned address.
	// CAUTION: This will be outside the gap limit.     [deprecated]
	// CreateNewAddress(purpose KeyPurpose) btcutil.Address

	// DecodeAddress parses the address string and return an address.
	DecodeAddress(addr string) (btcutil.Address, error)

	// ScriptToAddress takes a raw output script (the full script, not just a
	// hash160) and returns the corresponding address.
	ScriptToAddress(script []byte) (btcutil.Address, error)

	// Turn the given address into an output script
	AddressToScript(address btcutil.Address) ([]byte, error)

	// Add a subscription to the wallet. These addresses will be used to
	// subscribe to ElectrumX and get notifications back from ElectrumX
	// when coins are received. If already stored this is a no-op.
	AddSubscription(subscription *Subscription) error

	// Remove a subscription from the wallet.
	RemoveSubscription(scriptPubKey string)

	// Returns the ElectrumX subscribed subscription in db that has scriptPubKey
	// as a key.
	GetSubscription(scriptPubKey string) (*Subscription, error)

	// Returns the ElectrumX subscribed subscription in db that has electrumScripthash
	// as a key.
	GetSubscriptionForElectrumScripthash(electrumScripthash string) (*Subscription, error)

	// Returns all the ElectrumX subscribed subscriptions in db.
	ListSubscriptions() ([]*Subscription, error)

	// Returns if the wallet has the HD key for the given address
	HasAddress(address btcutil.Address) bool

	// Returns a list of addresses for this wallet
	ListAddresses() []btcutil.Address

	//Returns true if the address is a wallet address
	IsMine(address btcutil.Address) bool

	// Balance returns the confirmed & unconfirmed aggregate balance for the wallet.
	// For utxo based wallets if a spend of confirmed coins is made, the resulting
	// "change" should be also counted as confirmed even if the spending transaction
	// is unconfirmed. The reason for this that if the spend never confirms, no coins
	// will be lost to the wallet.
	//
	// This command uses the local wallet. We can also get from ElectrumX but on a per
	// address basis.
	Balance() (int64, int64, int64, error)

	// Sign an unsigned transaction with the wallet and return signed tx and
	// the change output index
	SignTx(pw string, info *SigningInfo) ([]byte, error)

	// Returns a list of transactions for this wallet - currently unused
	ListTransactions() ([]Txn, error)

	// Does the wallet have a specific transaction?
	HasTransaction(txid string) (bool, *Txn)

	// Get info on a specific transaction - currently unused
	GetTransaction(txid string) (*Txn, error)

	// Return the calculated confirmed txids and heights for an address - unused
	GetWalletAddressHistory(address btcutil.Address) ([]AddressHistory, error)

	// Add a transaction to the database
	AddTransaction(tx *wire.MsgTx, height int64, timestamp time.Time) error

	// List all unspent outputs in the wallet irrespective of status
	ListUnspent() ([]Utxo, error)

	// List all unspent outputs in the wallet that have been mined once or more
	// times
	ListConfirmedUnspent() ([]Utxo, error)

	// List all unspent outputs that are frozen (temporarily) in the wallet
	ListFrozenUnspent() ([]Utxo, error)

	// List all spent
	ListSpent() ([]Stxo, error)

	// Set the utxo as temporarily unspendable
	FreezeUTXO(op *wire.OutPoint) error

	// Set the utxo as spendable again
	UnFreezeUTXO(op *wire.OutPoint) error

	// Make a new spending transaction
	Spend(pw string, amount int64, toAddress btcutil.Address, feeLevel FeeLevel) (int, *wire.MsgTx, error)

	// Calculates the estimated size of the transaction and returns the total fee for the given feePerByte
	EstimateFee(ins []InputInfo, outs []TransactionOutput, feePerByte int64) int64

	// Build a transaction that sweeps all coins from a non-wallet private key
	SweepCoins(coins []InputInfo, feeLevel FeeLevel, maxTxInputs int) ([]*wire.MsgTx, error)

	// CPFP logic; rbf not supported
	BumpFee(txid string) (*wire.MsgTx, error)

	// Update the height of the tip from the blockchain headers.
	UpdateTip(newTip int64)
}

// Errors
var (
	// ErrDustAmount is returned if an output amount is below the dust threshold
	ErrDustAmount error = errors.New("amount is below network dust threshold")

	// ErrInsufficientFunds is returned when the wallet is unable to send the
	// amount specified due to the balance being too low
	ErrInsufficientFunds = errors.New("ERROR_INSUFFICIENT_FUNDS")

	// ErrWalletFnNotImplemented is returned from some unimplemented functions.
	// This is due to a concrete wallet not implementing the functionality or
	// temporarily during development.
	ErrWalletFnNotImplemented = errors.New("wallet function is not implemented")
)

type FeeLevel int

const (
	PRIORITY       FeeLevel = 0
	NORMAL         FeeLevel = 1
	ECONOMIC       FeeLevel = 2
	FEE_BUMP       FeeLevel = 3
	SUPER_ECONOMIC FeeLevel = 4
)

// The end leaves on the HD wallet have only two possible values. External keys are those given
// to other people for the purpose of receiving transactions. These may include keys used for
// refund addresses. Internal keys are used only by the wallet, primarily for change addresses
// but could also be used for shuffling around UTXOs.
type KeyChange int

const (
	EXTERNAL KeyChange = 0
	INTERNAL KeyChange = 1
	// Aliases
	RECEIVING = EXTERNAL
	CHANGE    = INTERNAL
)

type AddressHistory struct {
	Height int64
	TxHash chainhash.Hash
}

type TransactionOutput struct {
	Address btcutil.Address
	Value   int64
	Index   uint32
}

type SigningInfo struct {
	UnsignedTx *wire.MsgTx
	VerifyTx   bool
}

type InputInfo struct {
	Outpoint      *wire.OutPoint
	Height        int64
	Value         int64
	KeyPair       *btcutil.WIF
	LinkedAddress btcutil.Address
	PkScript      []byte
}

func (info *InputInfo) String() string {
	var outPoint = ""
	var linkedAddress = ""
	var privkey string = ""
	var pubkey string = ""
	var redeemscript = ""
	if info.Outpoint != nil {
		outPoint = info.Outpoint.String()
	}
	if info.LinkedAddress != nil {
		linkedAddress = fmt.Sprintf("%s %x", info.LinkedAddress.String(), info.LinkedAddress.ScriptAddress())
	}
	if info.KeyPair != nil {
		privkey = hex.EncodeToString(info.KeyPair.PrivKey.Serialize())
		pubkey = hex.EncodeToString(info.KeyPair.SerializePubKey())
	}
	if len(info.PkScript) > 0 {
		redeemscript = hex.EncodeToString(info.PkScript)
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Outpoint:      %s\n", outPoint))
	sb.WriteString(fmt.Sprintf("Height:        %d\n", info.Height))
	sb.WriteString(fmt.Sprintf("Value:         %d sats\n", info.Value))
	sb.WriteString(fmt.Sprintf("LinkedAddress: %s\n", linkedAddress))
	sb.WriteString(fmt.Sprintf("KeyPair:       %s %s\n", privkey, pubkey))
	sb.WriteString(fmt.Sprintf("RedeemScript:  %s\n", redeemscript))
	return sb.String()
}
