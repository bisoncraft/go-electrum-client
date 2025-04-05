package firo

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/bisoncraft/go-electrum-client/wallet"
	"github.com/btcsuite/btcd/btcutil"
)

// Import UTXO's for a known privkey from another wallet from electrumX. Partially
// implemented (P2WPKH,P2PKH) as it is not the most important tool for this wallet.

func (ec *FiroElectrumClient) getWitnessScriptHashRedeemUtxos(_ context.Context /*keyPair*/, _ *btcutil.WIF) ([]wallet.InputInfo, error) {
	utxoList := make([]wallet.InputInfo, 0)
	return utxoList, nil
}

func (ec *FiroElectrumClient) getScriptHashRedeemUtxos(_ context.Context /*keyPair*/, _ *btcutil.WIF) ([]wallet.InputInfo, error) {
	utxoList := make([]wallet.InputInfo, 0)
	return utxoList, nil
}

func (ec *FiroElectrumClient) getPubKeyUtxos(_ context.Context, _ *btcutil.WIF) ([]wallet.InputInfo, error) {
	utxoList := make([]wallet.InputInfo, 0)
	return utxoList, nil
}

func (ec *FiroElectrumClient) getPubKeyHashUtxos(ctx context.Context, keyPair *btcutil.WIF) ([]wallet.InputInfo, error) {
	inputList := make([]wallet.InputInfo, 0, 1)
	pubKey := keyPair.SerializePubKey()

	node := ec.GetX()
	if node == nil {
		return inputList, ErrNoElectrumX
	}
	// make address p2pkh
	pkHash := hash160(pubKey)
	addressPubKeyHash, err := btcutil.NewAddressPubKeyHash(pkHash, ec.GetConfig().Params)
	if err != nil {
		return inputList, err
	}
	// make scripthash
	scripthash, err := addressToElectrumScripthash(addressPubKeyHash)
	if err != nil {
		return inputList, err
	}
	// ask electrumX
	listUnspent, err := node.GetListUnspent(ctx, scripthash)
	if err != nil {
		return inputList, err
	}
	for _, unspent := range listUnspent {
		op, err := wallet.NewOutPointFromString(
			fmt.Sprintf("%s:%d", unspent.TxHash, unspent.TxPos))
		if err != nil {
			return inputList, err
		}
		input := wallet.InputInfo{
			Outpoint:      op,
			Height:        unspent.Height,
			Value:         unspent.Value,
			LinkedAddress: addressPubKeyHash,
			PkScript:      []byte{},
			KeyPair:       keyPair,
		}
		inputList = append(inputList, input)
	}
	return inputList, nil
}

func (ec *FiroElectrumClient) getWitnessPubKeyHashUtxos(ctx context.Context, keyPair *btcutil.WIF) ([]wallet.InputInfo, error) {
	inputList := make([]wallet.InputInfo, 0, 1)
	pubKey := keyPair.SerializePubKey()

	node := ec.GetX()
	if node == nil {
		return inputList, ErrNoElectrumX
	}
	// make address p2wpkh
	pkHash := hash160(pubKey)
	addressWitnessPubKeyHash, err := btcutil.NewAddressWitnessPubKeyHash(pkHash, ec.GetConfig().Params)
	if err != nil {
		return inputList, err
	}
	// make scripthash
	scripthash, err := addressToElectrumScripthash(addressWitnessPubKeyHash)
	if err != nil {
		return inputList, err
	}
	// ask electrum
	listUnspent, err := node.GetListUnspent(ctx, scripthash)
	if err != nil {
		return inputList, err
	}
	for _, unspent := range listUnspent {
		op, err := wallet.NewOutPointFromString(
			fmt.Sprintf("%s:%d", unspent.TxHash, unspent.TxPos))
		if err != nil {
			return inputList, err
		}
		input := wallet.InputInfo{
			Outpoint:      op,
			Height:        unspent.Height,
			Value:         unspent.Value,
			LinkedAddress: addressWitnessPubKeyHash,
			PkScript:      []byte{},
			KeyPair:       keyPair,
		}
		inputList = append(inputList, input)
	}
	return inputList, nil
}

func (ec *FiroElectrumClient) getUtxos(ctx context.Context, keyPair *btcutil.WIF) ([]wallet.InputInfo, error) {
	inputList := make([]wallet.InputInfo, 0, 1)

	// P2WSH
	p2wshInputList, err := ec.getWitnessScriptHashRedeemUtxos(ctx, keyPair)
	if err != nil {
		return inputList, err
	}
	if len(p2wshInputList) > 0 {
		inputList = append(inputList, p2wshInputList...)
	}

	// P2SH - not yet implemented
	p2shInputList, err := ec.getScriptHashRedeemUtxos(ctx, keyPair)
	if err != nil {
		return inputList, err
	}
	if len(p2shInputList) > 0 {
		inputList = append(inputList, p2shInputList...)
	}

	// P2PK - including satoshi's coins maybe .. not yet implemented
	p2pkInputList, err := ec.getPubKeyUtxos(ctx, keyPair)
	if err != nil {
		return inputList, err
	}
	if len(p2pkInputList) > 0 {
		inputList = append(inputList, p2pkInputList...)
	}

	// P2PKH
	p2pkhInputList, err := ec.getPubKeyHashUtxos(ctx, keyPair)
	if err != nil {
		return inputList, err
	}
	if len(p2pkhInputList) > 0 {
		inputList = append(inputList, p2pkhInputList...)
	}

	// P2WPKH
	p2wpkhInputList, err := ec.getWitnessPubKeyHashUtxos(ctx, keyPair)
	if err != nil {
		return inputList, err
	}
	if len(p2wpkhInputList) > 0 {
		inputList = append(inputList, p2wpkhInputList...)
	}

	return inputList, nil
}

// ImportAndSweep imports privkeys from other wallets and builds a transaction that
// sweeps their funds into our wallet.
func (ec *FiroElectrumClient) ImportAndSweep(ctx context.Context, importedKeyPairs []string) error {
	w := ec.GetWallet()
	if w == nil {
		return ErrNoWallet
	}
	if len(importedKeyPairs) <= 0 {
		return errors.New("no keys")
	}
	var inputs []wallet.InputInfo = make([]wallet.InputInfo, 0)
	for _, k := range importedKeyPairs {
		wif, err := btcutil.DecodeWIF(k)
		if err != nil {
			fmt.Printf("warning cannot decode WIF from string: %s\n", k)
			continue
		}

		inputsForKey, err := ec.getUtxos(ctx, wif)
		if err != nil {
			fmt.Printf("warning cannot get utxos for pubkey: %s\n",
				hex.EncodeToString(wif.SerializePubKey()))
			continue
		}
		if len(inputsForKey) <= 0 {
			continue
		}
		inputs = append(inputs, inputsForKey...)
	}
	if len(inputs) <= 0 {
		return errors.New("no inputs found")
	}
	// wallet sweep []tx
	txs, err := w.SweepCoins(inputs, wallet.NORMAL, 50)
	if err != nil {
		return err
	}
	for _, tx := range txs {
		var sweepBuf bytes.Buffer
		sweepBuf.Grow(tx.SerializeSize())
		tx.Serialize(&sweepBuf)
	}

	return nil
}
