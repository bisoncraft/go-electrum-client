package firo

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/bisoncraft/go-electrum-client/client"
	"github.com/bisoncraft/go-electrum-client/wallet"
)

// RescanWallet asks ElectrumX for info for our wallet keys back to latest
// checkpoint height.
// We need to do this for a recreated wallet.
func (ec *FiroElectrumClient) RescanWallet(ctx context.Context) error {
	w := ec.GetWallet()
	if w == nil {
		return ErrNoWallet
	}
	node := ec.GetX()
	if node == nil {
		return ErrNoElectrumX
	}

	// highest key index we will try for now
	highestKeyIndex := 100
	historyHitIndex := 0

	for keyIndex := 0; keyIndex <= highestKeyIndex; keyIndex++ {
		// flip-flop internal/external to improve locality
		for change := 0; change < 2; change++ {
			keyPath := &wallet.KeyPath{
				Change: wallet.KeyChange(change),
				Index:  keyIndex,
			}
			address, err := w.GetAddress(keyPath)
			if err != nil {
				fmt.Printf("bad address for: %d:%d\n", keyIndex, change)
				continue
			}
			scripthash, err := addressToElectrumScripthash(address)
			if err != nil {
				fmt.Printf("cannot make script hash for address: %s\n", address.String())
				continue
			}
			// fmt.Printf("%s %s  Index:change %d:%d\n", address.String(), scripthash, keyIndex, change)

			history, err := node.GetHistory(ctx, scripthash)
			if err != nil {
				fmt.Printf("error: %v - for scripthash %s\n", scripthash, err)
				continue
			}
			if len(history) == 0 {
				// fmt.Printf("No history for script hash from node: %s\n", scripthash)
				continue
			}
			// got history - update the highest hit index
			historyHitIndex = keyIndex
			// for _, h := range history {
			// 	fmt.Println(" Height:", h.Height)
			// 	fmt.Println(" TxHash: ", h.TxHash)
			// 	fmt.Println(" Fee: ", h.Fee)
			// }
			pkScriptBytes, err := w.AddressToScript(address)
			if err != nil {
				fmt.Printf("cannot make pkScript for address: %s\n", address.String())
				continue
			}
			subscription := &wallet.Subscription{
				PkScript:           hex.EncodeToString(pkScriptBytes),
				ElectrumScripthash: scripthash,
				Address:            address.String(),
			}
			err = w.AddSubscription(subscription)
			if err != nil {
				fmt.Printf("cannot add subscritpion for address: %s\n", address.String())
				// ec.dumpSubscription("failed to add", subscription)
				continue
			}
			// fmt.Printf("Added subscritpion for address: %s to wallet subscriptions\n", address.String())
		}

		// if no more history hits for another GAP_LIMIT tries consider the job done.
		if keyIndex > historyHitIndex+client.GAP_LIMIT {
			// fmt.Printf("keyIndex: %d greater than highest history found index %d by GAP_LIMIT %d\n\n",
			// 	keyIndex, historyHitIndex, client.GAP_LIMIT)
			break
		}
	}

	return nil
}
