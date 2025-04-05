package btc

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"

	"github.com/bisoncraft/go-electrum-client/wallet"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
)

/////////////////////////////////////////////////////////////////////////
// Here is the client interface between the node & wallet for transaction
// broadcast and wallet synchronization with ElectrumX's network 'view'
/////////////////////////////////////////////////////////////////////////

var ErrNoWallet error = errors.New("no wallet")
var ErrNoElectrumX error = errors.New("no ElectrumX")

// SyncWallet sets up address notifications for subscribed addresses in the
// wallet db. This will update txns, utxos, stxos wallet db tables with any
// new address status history since the wallet was last open.
func (ec *BtcElectrumClient) SyncWallet(ctx context.Context) error {
	w := ec.GetWallet()
	if w == nil {
		return ErrNoWallet
	}
	subscriptions, err := w.ListSubscriptions()
	if err != nil {
		return err
	}
	// - get all subscribed receive/change/watched addresses in wallet db
	for _, subscription := range subscriptions {

		// for each:
		//   - subscribe for scripthash notifications from electrumX node
		//   - on sub the return is hash of all address history known to server
		//     i.e. the up to date history list of txid:height, if any
		//   - for each tx insert or update the wallet db

		status, err := ec.SubscribeAddressNotify(ctx, subscription)
		if err != nil {
			return err
		}
		if status == "" {
			// fmt.Println("no history for this script address .. yet")
			continue
		}
		// get address history to date for this address from ElectrumX
		history, err := ec.GetAddressHistoryFromNode(ctx, subscription)
		if err != nil {
			return err
		}
		// ec.dumpHistory(subscription, history)

		// update wallet txstore if needed
		ec.addTxHistoryToWallet(ctx, history)
	}
	// start goroutine to listen for scripthash status change notifications arriving
	err = ec.addressStatusNotify(ctx)
	if err != nil {
		return err
	}
	return nil
}

//////////////////////////////////////////////////////////////////////////////
// Python console-like subset
/////////////////////////////

// Spend tries to create a new transaction to pay an amount from the wallet to
// toAddress. It returns Tx & Txid as hex strings and the index of any change
// output or -1 if none.
// The wallet password is required in order to sign the tx.
func (ec *BtcElectrumClient) Spend(
	pw string,
	amount int64,
	toAddress string,
	feeLevel wallet.FeeLevel) (int, string, string, error) {

	w := ec.GetWallet()
	if w == nil {
		return -1, "", "", ErrNoWallet
	}
	w.UpdateTip(ec.Tip())
	address, err := btcutil.DecodeAddress(toAddress, ec.ClientConfig.Params)
	if err != nil {
		return -1, "", "", err
	}
	changeIndex, wireTx, err := w.Spend(pw, amount, address, feeLevel)
	if err != nil {
		return -1, "", "", err
	}
	txidHex := wireTx.TxHash().String()
	b, err := serializeWireTx(wireTx)
	if err != nil {
		return -1, "", "", err
	}
	rawTxHex := hex.EncodeToString(b)
	return changeIndex, rawTxHex, txidHex, nil
}

// GetPrivKeyForAddress
func (ec *BtcElectrumClient) GetPrivKeyForAddress(pw, addr string) (string, error) {
	w := ec.GetWallet()
	if w == nil {
		return "", ErrNoWallet
	}
	address, err := btcutil.DecodeAddress(addr, w.Params())
	if err != nil {
		return "", err
	}
	return w.GetPrivKeyForAddress(pw, address)
}

func (ec *BtcElectrumClient) SignTx(pw string, txBytes []byte) ([]byte, error) {
	w := ec.GetWallet()
	if w == nil {
		return nil, ErrNoWallet
	}
	w.UpdateTip(ec.Tip())
	// Note: this errors if no inputs or no outputs or both
	unsignedTx, err := newWireTx(txBytes, true /*checkIo*/)
	if err != nil {
		return nil, err
	}
	var signInfo = &wallet.SigningInfo{
		UnsignedTx: unsignedTx,
		VerifyTx:   true,
	}
	return w.SignTx(pw, signInfo)
}

// GetWalletTx gets a tx from this wallet if it exists. If it does not esist then
// we return error.
// Edge cases:
//
//   - If the tx in the db is unmined we return it with no error. But if it has been
//     mined then we compare the height it was mined with 'maybeTip' and return an
//     error if the wallet tip is obviously behind electrumx; and we also return a
//     flag indicating that.
//     Software calling this API can then choose to directly ask electrumx for tx
//     information using the GetTransaction API.
//
//   - if wallet tx height is less than or equal to 'maybeTip' we return the tx with
//     no error even though we may be behind electrumx.
func (ec *BtcElectrumClient) GetWalletTx(txid string) (int, bool, []byte, error) {
	w := ec.GetWallet()
	if w == nil {
		return -1, false, nil, ErrNoWallet
	}
	txn, err := w.GetTransaction(txid)
	if err != nil {
		// 'no such transaction'
		return -1, false, nil, err
	}
	if txn.Height < 0 {
		return -1, false, nil, errors.New("txns db error")
	}
	// not mined yet? return valid tx with no confs
	if txn.Height == 0 {
		return 0, false, txn.Bytes, nil
	}
	// mined
	maybeTip := ec.Tip()
	conf := maybeTip - txn.Height
	// definitely client tip is behind electrumx
	if conf < 0 {
		return int(conf), true, txn.Bytes, errors.New("tip behind electrumx")
	}
	return int(conf), false, txn.Bytes, nil
}

func (ec *BtcElectrumClient) GetWalletSpents() ([]wallet.Stxo, error) {
	w := ec.GetWallet()
	if w == nil {
		return nil, ErrNoWallet
	}
	return w.ListSpent()
}

// RpcBroadcast sends a transaction to the server for broadcast on the bitcoin
// network. It is a test rpc server endpoint and it is thus not part of the
// ElectrumClient interface.
func (ec *BtcElectrumClient) RpcBroadcast(ctx context.Context, tx string) (string, error) {
	rawTx, err := hex.DecodeString(tx)
	if err != nil {
		return "", err
	}
	return ec.Broadcast(ctx, rawTx)
}

// Broadcast sends a transaction to the ElectrumX server for broadcast on the
// bitcoin network. It may also set up address status change notifications with
// ElectrumX in the wallet db for addresses such as change address belonging to
// the wallet.
func (ec *BtcElectrumClient) Broadcast(ctx context.Context, rawTx []byte) (string, error) {
	params := ec.ClientConfig.Params
	w := ec.GetWallet()
	if w == nil {
		return "", ErrNoWallet
	}
	node := ec.GetX()
	if node == nil {
		return "", ErrNoElectrumX
	}
	if rawTx == nil {
		return "", errors.New("nil Tx")
	}
	tx, err := newWireTx(rawTx, true)
	if err != nil {
		return "", err
	}
	// Find any outputs that pay back to this wallet. In particular we almost
	// always have a change scriptaddress to watch paying back to this	 wallet
	// after it's containing tx is broadcasted to the network by ElectrumX
	ourAddresses := w.ListAddresses()
	isOurs := func(address btcutil.Address) bool {
		for _, ourAddress := range ourAddresses {
			if bytes.Equal(address.ScriptAddress(), ourAddress.ScriptAddress()) {
				return true
			}
		}
		return false
	}
	var backToWallet = make(map[int][]byte)
	for idx, txOut := range tx.TxOut {
		pkScript := txOut.PkScript
		_, addresses, _, err := txscript.ExtractPkScriptAddrs(pkScript, params)
		if err != nil {
			return "", err
		}
		if len(addresses) != 1 {
			return "", err
		}
		if isOurs(addresses[0]) {
			backToWallet[idx] = pkScript
		}
	}
	// fmt.Printf("found %d address(es) back to our wallet\n", len(backToWallet))

	// Send tx to ElectrumX for broadcasting to the bitcoin network
	rawTxStr := hex.EncodeToString(rawTx)
	txid, err := node.Broadcast(ctx, rawTxStr)
	if err != nil {
		return "", err
	}

	// Subscribe for address status notification from ElectrumX for addresses
	// paying back to our wallet. This will also add the containing tx to the
	// wallet txns db in response to the first status change notification of
	// the subscribed address.
	for idx := range tx.TxOut {
		pkScript, ok := backToWallet[idx]
		if !ok {
			continue
		}
		// make wallet subscription for this output pkScript
		scripthash := pkScriptToElectrumScripthash(pkScript)
		_, addr := ec.pkScriptToAddressPubkeyHash(pkScript)
		newSub := wallet.Subscription{
			PkScript:           hex.EncodeToString(pkScript),
			ElectrumScripthash: scripthash,
			Address:            addr,
		}
		// add to db
		err = w.AddSubscription(&newSub)
		if err != nil {
			// assert db store .. stop here before things get more messed up
			panic(err)
		}

		// request notifications from node
		res, err := node.SubscribeScripthashNotify(ctx, scripthash)
		if err != nil {
			w.RemoveSubscription(newSub.PkScript)
			return "", err
		}
		if res == nil { // network error
			w.RemoveSubscription(newSub.PkScript)
			return "", errors.New("network: empty result")
		}
	}
	return txid, nil
}

// ListUnspent returns a list of all utxos in the wallet db.
func (ec *BtcElectrumClient) ListUnspent() ([]wallet.Utxo, error) {
	w := ec.GetWallet()
	if w == nil {
		return nil, ErrNoWallet
	}
	w.UpdateTip(ec.Tip())
	return w.ListUnspent()
}

// ListConfirmedUnspent returns a list of all utxos in the wallet db with height > 0.
func (ec *BtcElectrumClient) ListConfirmedUnspent() ([]wallet.Utxo, error) {
	w := ec.GetWallet()
	if w == nil {
		return nil, ErrNoWallet
	}
	w.UpdateTip(ec.Tip())
	return w.ListConfirmedUnspent()
}

// ListFrozenUnspent returns a list of all utxos in the wallet db that are temporarily frozen.
func (ec *BtcElectrumClient) ListFrozenUnspent() ([]wallet.Utxo, error) {
	w := ec.GetWallet()
	if w == nil {
		return nil, ErrNoWallet
	}
	w.UpdateTip(ec.Tip())
	return w.ListFrozenUnspent()
}

// UnusedAddress gets a new unused wallet receive address and subscribes for
// ElectrumX address status notify events on the returned address.
func (ec *BtcElectrumClient) UnusedAddress(ctx context.Context) (string, error) {
	w := ec.GetWallet()
	if w == nil {
		return "", ErrNoWallet
	}
	node := ec.GetX()
	if node == nil {
		return "", ErrNoElectrumX
	}
	address, err := w.GetUnusedAddress(wallet.RECEIVING)
	if err != nil {
		return "", err
	}
	payToAddrScript, err := txscript.PayToAddrScript(address)
	if err != nil {
		return "", err
	}
	// wallet db
	newSub := &wallet.Subscription{
		PkScript:           hex.EncodeToString(payToAddrScript),
		ElectrumScripthash: pkScriptToElectrumScripthash(payToAddrScript),
		Address:            address.String(),
	}
	// ec.dumpSubscription("adding/updating get unused address subscription", newSub)
	// insert or update
	err = w.AddSubscription(newSub)
	if err != nil {
		return "", err
	}
	// request notifications from node
	res, err := node.SubscribeScripthashNotify(ctx, newSub.ElectrumScripthash)
	if err != nil {
		w.RemoveSubscription(newSub.PkScript)
		return "", err
	}
	if res == nil { // network error
		w.RemoveSubscription(newSub.PkScript)
		return "", errors.New("network: empty result")
	}
	return address.String(), nil
}

// ChangeAddress gets a new unused wallet change address and subscribes for
// ElectrumX address status notify events on the returned address.
func (ec *BtcElectrumClient) ChangeAddress(ctx context.Context) (string, error) {
	w := ec.GetWallet()
	if w == nil {
		return "", ErrNoWallet
	}
	node := ec.GetX()
	if node == nil {
		return "", ErrNoElectrumX
	}
	address, err := w.GetUnusedAddress(wallet.CHANGE)
	if err != nil {
		return "", err
	}
	payToAddrScript, err := txscript.PayToAddrScript(address)
	if err != nil {
		return "", err
	}
	// wallet db
	newSub := &wallet.Subscription{
		PkScript:           hex.EncodeToString(payToAddrScript),
		ElectrumScripthash: pkScriptToElectrumScripthash(payToAddrScript),
		Address:            address.String(),
	}
	// ec.dumpSubscription.("adding/updating get change address subscription", newSub)
	// insert or update
	err = w.AddSubscription(newSub)
	if err != nil {
		return "", err
	}
	// request notifications from node
	res, err := node.SubscribeScripthashNotify(ctx, newSub.ElectrumScripthash)
	if err != nil {
		w.RemoveSubscription(newSub.PkScript)
		return "", err
	}
	if res == nil { // network error
		w.RemoveSubscription(newSub.PkScript)
		return "", errors.New("network: empty result")
	}
	return address.String(), nil
}

// ValidateAddress returns if the address is valid and if it does or does not
// belong to this wallet
func (ec *BtcElectrumClient) ValidateAddress(addr string) (bool, bool, error) {
	w := ec.GetWallet()
	if w == nil {
		return false, false, ErrNoWallet
	}
	queryAddress, err := btcutil.DecodeAddress(addr, ec.ClientConfig.Params)
	if err != nil {
		return false, false, err
	}
	// so it is a valid address according to btcutil
	mine := w.IsMine(queryAddress)
	return true, mine, nil
}

// Balance returns the confirmed and unconfirmed balances of this wallet.
// This is a simple wallet and once a transaction has been mined it is
// considered confirmed.
func (ec *BtcElectrumClient) Balance() (int64, int64, int64, error) {
	w := ec.GetWallet()
	if w == nil {
		return 0, 0, 0, ErrNoWallet
	}
	return w.Balance()
}

func (ec *BtcElectrumClient) FreezeUTXO(txid string, out uint32) error {
	w := ec.GetWallet()
	if w == nil {
		return ErrNoWallet
	}
	op, err := wallet.NewOutPoint(txid, out)
	if err != nil {
		return err
	}
	return w.FreezeUTXO(op)
}

func (ec *BtcElectrumClient) UnfreezeUTXO(txid string, out uint32) error {
	w := ec.GetWallet()
	if w == nil {
		return ErrNoWallet
	}
	op, err := wallet.NewOutPoint(txid, out)
	if err != nil {
		return err
	}
	return w.UnFreezeUTXO(op)
}

func (ec *BtcElectrumClient) FeeRate(ctx context.Context, confTarget int64) (int64, error) {
	// Maye we can use oracle like https://blockexplorer.one/bitcoin/mainnet (/testnet)
	// https://rest.cryptoapis.io/v2/blockchain-data/bitcoin/testnet/mempool/fees
	// I do not like this!
	// node := ec.GetNode()
	// if node != nil {
	// 	feeRate, _ := node.EstimateFeeRate(ctx, confTarget)
	// 	if feeRate != -1 {
	// 		return feeRate, nil
	// 	}
	// }

	// static
	switch ec.ClientConfig.Params {
	case &chaincfg.MainNetParams:
		return 30000, nil
	case &chaincfg.TestNet3Params:
		return 1500, nil
	case &chaincfg.RegressionNetParams:
		return 1500, nil
	default:
		return 1000, nil
	}
}
