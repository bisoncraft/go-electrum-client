package btc

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/dev-warrior777/go-electrum-client/client"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

/////////////////////////////////////////////////////////////////////////
// Here is the client interface between the node & wallet for transaction
// broadcast and wallet synchronization with ElectrumX's network 'view'
/////////////////////////////////////////////////////////////////////////

var ErrNoWallet error = errors.New("no wallet")
var ErrNoNode error = errors.New("no node")

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
			fmt.Println("no history for this script address .. yet")
			continue
		}

		// get address history to date for this address from ElectrumX
		history, err := ec.GetAddressHistoryFromNode(ctx, subscription)
		if err != nil {
			return err
		}
		ec.dumpHistory(subscription, history)

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
// output or -1 if none. The client needs to know the change address so it can
// set up notification of change address status after ElectrumX later broadcasts
// the resultant spend tx to the bitcoin network. This function does not broadcast
// the transaction.
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

	b := make([]byte, 0, wireTx.SerializeSize())
	buf := bytes.NewBuffer(b)
	err = wireTx.BtcEncode(buf, 0, wire.WitnessEncoding)
	if err != nil {
		return -1, "", "", err
	}
	rawTxHex := hex.EncodeToString(buf.Bytes())

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

func (ec *BtcElectrumClient) SignTx(ctx context.Context, pw string, txBytes []byte) (int, []byte, error) {
	w := ec.GetWallet()
	if w == nil {
		return -1, nil, ErrNoWallet
	}
	newWireTx := func(b []byte) (*wire.MsgTx, error) {
		tx := wire.NewMsgTx(wire.TxVersion)
		r := bytes.NewBuffer(b)
		err := tx.Deserialize(r)
		if len(tx.TxIn) == 0 {
			return nil, errors.New("tx: no inputs")
		}
		if len(tx.TxOut) == 0 {
			return nil, errors.New("tx: no outputs")
		}
		return tx, err
	}
	unsignedTx, err := newWireTx(txBytes)
	if err != nil {
		return -1, nil, err
	}
	var signInfo = &wallet.SigningInfo{
		UnsignedTx: unsignedTx,
		PrevOuts:   make([]*wallet.InputInfo, 0),
	}
	for i, in := range unsignedTx.TxIn {
		prevOut := &wallet.InputInfo{}
		txid := in.PreviousOutPoint.Hash.String()
		txn, err := w.GetTransaction(txid)
		if err == nil {
			// wallet tx
			wtx, err := newWireTx(txn.Bytes)
			if err != nil {
				return -1, nil, err
			}
			if uint32(len(wtx.TxOut)) < in.PreviousOutPoint.Index {
				return -1, nil, fmt.Errorf("tx: no prev out found for input[%d] vout", i)
			}
			prevOut.RedeemScript = wtx.TxOut[i].PkScript
			prevOut.Value = wtx.TxOut[i].Value
			signInfo.PrevOuts = append(signInfo.PrevOuts, prevOut)
			continue
		}
		// global tx
		grtBytes, err := ec.GetRawTransaction(ctx, txid)
		if err != nil {
			return -1, nil, err
		}
		gtx, err := newWireTx(grtBytes)
		if err != nil {
			return -1, nil, err
		}
		if uint32(len(gtx.TxOut)) <= in.PreviousOutPoint.Index {
			return -1, nil, fmt.Errorf("tx: no prev out found for input[%d] vout", i)
		}
		prevOut.RedeemScript = gtx.TxOut[i].PkScript
		prevOut.Value = gtx.TxOut[i].Value
		signInfo.PrevOuts = append(signInfo.PrevOuts, prevOut)
	}

	changeIndex, signedTx, err := w.SignTx(pw, unsignedTx, signInfo)

	fmt.Printf("signed Tx:\n%s\nchange index: %d\nerr:%v\n",
		hex.EncodeToString(signedTx),
		changeIndex,
		err,
	)
	//...

	return -1, nil, errors.New("not implemented on goele side")
}

func (ec *BtcElectrumClient) GetWalletTx(txid string) (int, []byte, error) {
	w := ec.GetWallet()
	if w == nil {
		return -1, nil, ErrNoWallet
	}
	txn, err := w.GetTransaction(txid)
	if err != nil {
		return -1, nil, err
	}

	tip, _ := ec.Tip()
	confirmations := tip - txn.Height

	return int(confirmations), txn.Bytes, nil
}

// RpcBroadcast sends a transaction to the server for broadcast on the bitcoin
// network. It is a test rpc server endpoint and it is thus not part of the
// ElectrumClient interface.
func (ec *BtcElectrumClient) RpcBroadcast(ctx context.Context, rawTx string, changeIndex int) (string, error) {
	txBytes, err := hex.DecodeString(rawTx)
	if err != nil {
		return "", err
	}
	r := bytes.NewBuffer(txBytes)
	wireMsgTx := wire.NewMsgTx(wire.TxVersion)
	wireMsgTx.BtcDecode(r, wire.ProtocolVersion, wire.WitnessEncoding)
	bc := client.BroadcastParams{
		Tx:          wireMsgTx,
		ChangeIndex: changeIndex,
	}
	return ec.Broadcast(ctx, &bc)
}

// Broadcast sends a transaction to the ElectrumX server for broadcast on the
// bitcoin network. It may also set up address status change notifications with
// ElectrumX and the wallet db for a change address belonging to the wallet.
func (ec *BtcElectrumClient) Broadcast(ctx context.Context, bc *client.BroadcastParams) (string, error) {
	w := ec.GetWallet()
	if w == nil {
		return "", ErrNoWallet
	}
	node := ec.GetNode()
	if node == nil {
		return "", ErrNoNode
	}
	if bc.Tx == nil {
		return "", errors.New("nil Tx")
	}

	// serialize msg tx
	wb := bytes.NewBuffer(make([]byte, 0))
	err := bc.Tx.BtcEncode(wb, 1, wire.WitnessEncoding)
	if err != nil {
		return "", err
	}
	rawTx := wb.Bytes()

	// check change index is valid
	hasChange := false
	fmt.Println("*** change output index", bc.ChangeIndex, "***")
	if bc.ChangeIndex >= 0 {
		txOuts := bc.Tx.TxOut
		if len(txOuts) < bc.ChangeIndex+1 {
			return "", errors.New("invalid change index")
		}
		hasChange = true
	}

	// Send tx to ElectrumX for broadcasting to the bitcoin network
	rawTxStr := hex.EncodeToString(rawTx)
	txid, err := node.Broadcast(ctx, rawTxStr)
	if err != nil {
		return "", err
	}

	// Subscribe for address status notification from ElectrumX for addresses
	// we might be interested in. This should also add the containing tx to the
	// wallet txns db in response to the first status change notification of the
	// subscribed address. In particular we almost always have a change script
	// address to watch paying back to our wallet after it's containing tx is
	// broadcasted to the network by ElectrumX.

	if hasChange {
		change := bc.Tx.TxOut[bc.ChangeIndex]

		scripthash := pkScriptToElectrumScripthash(change.PkScript)

		// wallet db
		pkScriptStr := hex.EncodeToString(change.PkScript)
		_, addr := ec.pkScriptToAddressPubkeyHash(change.PkScript)
		newSub := wallet.Subscription{
			PkScript:           pkScriptStr,
			ElectrumScripthash: scripthash,
			Address:            addr,
		}
		ec.dumpSubscription("adding change subscription", &newSub)
		err = w.AddSubscription(&newSub)
		if err != nil {
			// assert db store
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
	return w.ListUnspent()
}

// UnusedAddress gets a new unused wallet receive address and subscribes for
// ElectrumX address status notify events on the returned address.
func (ec *BtcElectrumClient) UnusedAddress(ctx context.Context) (string, error) {
	w := ec.GetWallet()
	if w == nil {
		return "", ErrNoWallet
	}
	node := ec.GetNode()
	if node == nil {
		return "", ErrNoNode
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
	ec.dumpSubscription("adding/updating get unused address subscription", newSub)
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
	node := ec.GetNode()
	if node == nil {
		return "", ErrNoNode
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
	ec.dumpSubscription("adding/updating get change address subscription", newSub)
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

func (ec *BtcElectrumClient) Balance() (int64, int64, error) {
	w := ec.GetWallet()
	if w == nil {
		return 0, 0, ErrNoWallet
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
	node := ec.GetNode()
	if node != nil {
		feeRate, _ := node.EstimateFeeRate(ctx, confTarget)
		if feeRate != -1 {
			return feeRate, nil
		}
	}
	// for now just return default fee rate of 1000
	return 1000, nil
}
