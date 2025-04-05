package firo

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

	"github.com/bisoncraft/go-electrum-client/electrumx"
	"github.com/bisoncraft/go-electrum-client/wallet"
)

// Here is the client interface between the node & wallet for monitoring the
// status of wallet 'scripthashes'
//
// https://electrumx.readthedocs.io/en/latest/protocol-basics.html
//
// It can get confusing! Here 'scripthash' is an electrum value. But the
// ScriptHash from (btcutl.Address).ScriptHash() is the normal RIPEMD160
// hash.
//
// An electrum scripthash is the full output payment script which is then
// sha256 hashed. The result has bytes reversed for network send. It is sent
// to ElectrumX as a string.

var ErrNoSubscriptionFoundInDb = errors.New("no subscription found in db")

// pkScriptToElectrumScripthash takes a public key script and makes an electrum
// 1.4 protocol 'scripthash'
func pkScriptToElectrumScripthash(pkScript []byte) string {
	revBytes := func(b []byte) []byte {
		size := len(b)
		buf := make([]byte, size)
		var i int
		for i = 0; i < size; i++ {
			buf[i] = b[size-i-1]
		}
		return buf
	}
	pkScriptHashBytes := chainhash.HashB(pkScript)
	revScriptHashBytes := revBytes(pkScriptHashBytes)
	return hex.EncodeToString(revScriptHashBytes)
}

// addrToScripthash takes a btcutil.Address and makes an electrum 1.4 protocol 'scripthash'
func addressToElectrumScripthash(address btcutil.Address) (string, error) {
	pkScript, err := txscript.PayToAddrScript(address)
	if err != nil {
		return "", err
	}
	return pkScriptToElectrumScripthash(pkScript), nil
}

// addrToElectrumScripthash takes a bech or legacy bitcoin address and makes an electrum
// 1.4 protocol 'scripthash'
func addrToElectrumScripthash(addr string, network *chaincfg.Params) (string, error) {
	address, err := btcutil.DecodeAddress(addr, network)
	if err != nil {
		return "", err
	}
	return addressToElectrumScripthash(address)
}

// addSubscription adds subscription details to the wallet db
func (ec *FiroElectrumClient) addSubscription(subscription *wallet.Subscription) error {
	w := ec.GetWallet()
	if w == nil {
		return ErrNoWallet
	}
	return w.AddSubscription(subscription)
}

// getSubscriptionForScripthash gets subscription details from the wallet db keyed
// on electrum scripthash
func (ec *FiroElectrumClient) getSubscriptionForScripthash(scripthash string) (*wallet.Subscription, error) {
	w := ec.GetWallet()
	if w == nil {
		return nil, ErrNoWallet
	}
	subscription, err := w.GetSubscriptionForElectrumScripthash(scripthash)
	if err != nil {
		return nil, err
	}
	if subscription == nil {
		return nil, ErrNoSubscriptionFoundInDb
	}
	return subscription, nil
}

// getSubscriptionForScripthash
func (ec *FiroElectrumClient) getSubscription(scriptPubKey string) (*wallet.Subscription, error) {
	w := ec.GetWallet()
	if w == nil {
		return nil, ErrNoWallet
	}
	subscription, err := w.GetSubscription(scriptPubKey)
	if err != nil {
		return nil, err
	}
	if subscription == nil {
		return nil, ErrNoSubscriptionFoundInDb
	}
	return subscription, nil
}

// isSubscribed tests if an address pub key script has subscription details stored
// the wallet db
func (ec *FiroElectrumClient) isSubscribed(pkScript string) (bool, error) {
	w := ec.GetWallet()
	if w == nil {
		return false, ErrNoWallet
	}
	subscription, err := w.GetSubscription(pkScript)
	if err != nil {
		return false, err
	}
	if subscription == nil {
		return false, nil
	}
	return true, nil
}

// removeSubscription removes subscription details stored the wallet db for an
// address pub key script
func (ec *FiroElectrumClient) removeSubscription(pkScript string) error {
	w := ec.GetWallet()
	if w == nil {
		return ErrNoWallet
	}
	w.RemoveSubscription(pkScript)
	return nil
}

//////////////////////////////////////////////////////////////////////////////
// wallet <-> client <-> node
/////////////////////////////

// addressStatusNotify listens for address status change notifications
func (ec *FiroElectrumClient) addressStatusNotify(ctx context.Context) error {
	node := ec.GetX()

	scripthashNotifyCh, err := node.GetScripthashNotify()
	if err != nil {
		return err
	}

	go func() {

		fmt.Println("=== Waiting for address change notifications ===")

		for {
			select {

			case <-ctx.Done():
				fmt.Println("ctx.Done - in client scripthash notify - exiting thread")
				return

			case status, ok := <-scripthashNotifyCh:
				if !ok {
					fmt.Println("scripthash notify channel closed - exiting thread")
					return
				}

				if status.Status == "" {
					// fmt.Println("status.Status is null no history yet; ignoring...")
					continue
				}

				// get wallet db subscription details
				sub, err := ec.getSubscriptionForScripthash(status.Scripthash)
				if err != nil { // db assert  'no rows in result set'
					panic(err)
				}
				if sub == nil { // db assert
					panic("no subscription for subscribed scripthash")
				}

				// get scripthash history
				history, err := ec.GetAddressHistoryFromNode(ctx, sub)
				if err != nil {
					continue
				}
				// ec.dumpHistory(sub, history)

				// add/update wallet db tx store
				ec.addTxHistoryToWallet(ctx, history)
			}
		}
	}()
	// serve until done
	return nil
}

// SubscribeAddressNotify subscribes to notifications from ElectrumX for a public
// key script address. It also adds the new subscription to the wallet database.
// It returns a subscribe status which is the hash of all address history known
// to the electrumX server and can be a zero length string if the subscription
// is new and has no history yet.
func (ec *FiroElectrumClient) SubscribeAddressNotify(ctx context.Context, newSub *wallet.Subscription) (string, error) {
	node := ec.GetX()
	if node == nil {
		return "", ErrNoElectrumX
	}
	subscribed, err := ec.isSubscribed(newSub.PkScript)
	if err != nil {
		return "", err
	}

	// subscribe to node and wallet database
	res, err := node.SubscribeScripthashNotify(ctx, newSub.ElectrumScripthash)
	if err != nil {
		return "", err
	}
	if res == nil {
		return "", errors.New("empty result")
	}
	// wallet
	if !subscribed {
		ec.addSubscription(newSub)
	}
	return res.Status, nil
}

// UnsubscribeAddressNotify both unsubscribes from notifications for an address
// _and_ removes the subscription details from the wallet database.
func (ec *FiroElectrumClient) UnsubscribeAddressNotify(ctx context.Context, pkScript string) {
	node := ec.GetX()
	if node == nil {
		return
	}
	subscription, err := ec.getSubscription(pkScript)
	if err != nil || subscription == nil {
		fmt.Println("not subscribed or db error")
		return
	}

	// unsubscribe from node and wallet db
	node.UnsubscribeScripthashNotify(ctx, subscription.ElectrumScripthash)
	err = ec.removeSubscription(pkScript)
	if err != nil {
		fmt.Println("removeSubscription", err)
		return
	}
}

// GetAddressHistoryFromNode requests address history from ElectrumX  for a
// subscribed address.
func (ec *FiroElectrumClient) GetAddressHistoryFromNode(ctx context.Context, subscription *wallet.Subscription) (electrumx.HistoryResult, error) {
	node := ec.GetX()
	if node == nil {
		return nil, ErrNoElectrumX
	}
	res, err := node.GetHistory(ctx, subscription.ElectrumScripthash)
	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		fmt.Println("empty history result for: ", subscription.PkScript)
		return nil, nil
	}

	return res, nil
}

// GetRawTransactionFromNode requests a raw hex transaction for a subscribed address
// from ElectrumX keyed on a txid. This txid is usually taken from an ElectrumX
// history list.
func (ec *FiroElectrumClient) GetRawTransactionFromNode(ctx context.Context, txid string) (*wire.MsgTx, time.Time, error) {
	node := ec.GetX()
	if node == nil {
		return nil, time.Time{}, ErrNoElectrumX
	}
	txres, err := node.GetRawTransaction(ctx, txid)
	if err != nil {
		return nil, time.Time{}, err
	}
	b, err := hex.DecodeString(txres)
	if err != nil {
		return nil, time.Time{}, err
	}
	msgTx, err := newWireTx(b, true)
	if err != nil {
		return nil, time.Time{}, err
	}
	txTime := time.Now()
	return msgTx, txTime, nil
}

// addTxHistoryToWallet adds new transaction details for an ElectrumX history list
func (ec *FiroElectrumClient) addTxHistoryToWallet(ctx context.Context, history electrumx.HistoryResult) {
	for _, h := range history {
		// does wallet already has a confirmed transaction?
		walletHasTx, txn := ec.GetWallet().HasTransaction(h.TxHash)
		if walletHasTx && txn.Height > 0 {
			// fmt.Println("** already got confirmed tx", h.TxHash)
			continue
		}
		// add or update the wallet transaction
		msgTx, txtime, err := ec.GetRawTransactionFromNode(ctx, h.TxHash)
		if err != nil {
			continue
		}
		// fmt.Printf("adding/updating transaction txid: %s, height: %d, fee %d\n", h.TxHash, h.Height, h.Fee)
		err = ec.GetWallet().AddTransaction(msgTx, h.Height, txtime)
		if err != nil {
			fmt.Println(err)
			continue
		}
	}
}

func (ec *FiroElectrumClient) pkScriptToAddressPubkeyHash(pkScript []byte) (btcutil.Address, string) {
	pks, err := txscript.ParsePkScript(pkScript)
	if err != nil {
		return nil, ""
	}
	apkh, err := pks.Address(ec.GetConfig().Params)
	if err != nil {
		return nil, ""
	}
	return apkh, apkh.String()
}

// updateWalletTip updates wallet's notion of the blockchain tip based on the
// latest electrumx headers' "maybe" tip. This would be used for example in
// calculating tx confirmations.
func (ec *FiroElectrumClient) updateWalletTip(tip int64) {
	w := ec.GetWallet()
	if w != nil {
		w.UpdateTip(tip)
	}
}

// //////////////////////////
// dbg dump
// /////////
// func (ec *FiroElectrumClient) dumpSubscription(title string, sub *wallet.Subscription) {
// 	fmt.Printf("%s\n PkScript: %s\n ElectrumScriptHash: %s\n Address: %s\n\n",
// 		title,
// 		sub.PkScript,
// 		sub.ElectrumScripthash,
// 		sub.Address)
// }

// func (ec *FiroElectrumClient) dumpHistory(sub *wallet.Subscription, history electrumx.HistoryResult) {
// 	ec.dumpSubscription("Address History for subscription:", sub)
// 	for _, h := range history {
// 		fmt.Println(" Height:", h.Height)
// 		fmt.Println(" TxHash: ", h.TxHash)
// 		fmt.Println(" Fee: ", h.Fee)
// 	}
// }
