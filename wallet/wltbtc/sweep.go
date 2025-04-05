package wltbtc

import (
	"fmt"
	"os"

	"github.com/bisoncraft/go-electrum-client/wallet"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcwallet/wallet/txrules"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

var (
	verify = true
)

// Build (one or more) segwit transaction that sweeps all coins owned by a
// private key external to our wallet .. into our wallet.
// Each tx has many inputs but only one output. There are no change outputs.

const MAX_TX_INPUTS = 50 //TODO: enforce this

func (w *BtcElectrumWallet) SweepCoins(
	coins []wallet.InputInfo,
	feeLevel wallet.FeeLevel,
	maxTxInputs int) ([]*wire.MsgTx, error) {

	var sweepTxs = make([]*wire.MsgTx, 0)

	var totalValue int64
	var privKeyToSignOutputs map[int]*secp256k1.PrivateKey
	var prevOutScripts map[int][]byte
	var prevOutValues map[int]int64

	// segwit output which makes this always a segwit transaction
	walletAddressSegwit, err := w.GetUnusedAddress(wallet.RECEIVING)
	if err != nil {
		return nil, err
	}
	// make an output script that can spend the inputs to our wallet
	p2wpkhScript, err := txscript.PayToAddrScript(walletAddressSegwit)
	if err != nil {
		return nil, err
	}

	// make unsigned transaction

	sweepTx := wire.NewMsgTx(2 /*wire.TxVersion*/)

	privKeyToSignOutputs = make(map[int]*secp256k1.PrivateKey)
	prevOutScripts = make(map[int][]byte)
	prevOutValues = make(map[int]int64)

	for i, coin := range coins {
		scriptForInput, err := txscript.PayToAddrScript(coin.LinkedAddress)
		if err != nil {
			// we could change policy to just ignore .. see how
			return nil, err
		}
		totalValue += coin.Value
		prevOutValues[i] = coin.Value
		prevOutScripts[i] = scriptForInput
		privKeyToSignOutputs[i] = coin.KeyPair.PrivKey
		// make a basic input from coin
		input := wire.NewTxIn(coin.Outpoint, []byte{}, [][]byte{})
		sweepTx.TxIn = append(sweepTx.TxIn, input)
	}

	// get vsize estimate before adding the output
	sweepSize := txSerialSizeEst(sweepTx)

	// get the fee
	feePerKB := btcutil.Amount(w.GetFeePerByte(feeLevel)) * 1000
	fee := txrules.FeeForSerializeSize(feePerKB, sweepSize)

	// add single output
	outValue := totalValue - int64(fee)
	sweepTx.AddTxOut(wire.NewTxOut(outValue, p2wpkhScript))
	err = txrules.CheckOutput(sweepTx.TxOut[0], feePerKB)
	if err != nil {
		return nil, err
	}

	// sign

	// NewTxSigHashes uses the PrevOutFetcher only for detecting a taproot
	// output, so we can provide a dummy.
	prevOutFetcher := new(txscript.CannedPrevOutputFetcher)

	for idx, input := range sweepTx.TxIn {
		sigHashes := txscript.NewTxSigHashes(sweepTx, prevOutFetcher)
		prevOutScriptTy := txscript.GetScriptClass(prevOutScripts[idx])
		switch prevOutScriptTy {
		case txscript.WitnessV0PubKeyHashTy:
			sig, err := txscript.WitnessSignature(sweepTx, sigHashes, idx, prevOutValues[idx],
				prevOutScripts[idx], txscript.SigHashAll, privKeyToSignOutputs[idx], true)
			if err != nil {
				return nil, err
			}
			// add witness
			input.SignatureScript = nil
			input.Witness = append(input.Witness, sig...)
		case txscript.PubKeyHashTy:
			sig, err := txscript.SignatureScript(sweepTx, idx,
				prevOutScripts[idx], txscript.SigHashAll, privKeyToSignOutputs[idx], true)
			if err != nil {
				return nil, err
			}
			// add script sig
			input.SignatureScript = append(input.SignatureScript, sig...)
			input.Witness = nil
		default:
			return nil, fmt.Errorf("signing prev out script type %v unsupported",
				prevOutScriptTy)
		}
	}

	// verify

	if verify {
		for idx := range sweepTx.TxIn {
			e, err := txscript.NewEngine(
				// previous output pubkey script (from input)
				prevOutScripts[idx],
				// sweep transaction
				sweepTx,
				// transaction input index
				idx,
				txscript.StandardVerifyFlags,
				nil, //txscript.NewSigCache(10),
				nil, //txscript.NewTxSigHashes(sweepTx, prevOutFetcher),
				prevOutValues[idx],
				prevOutFetcher)
			if err != nil {
				panic(err)
			}
			err = e.Execute()
			if err != nil {
				fmt.Printf("Engine Error: %v\n", err)
				os.Exit(1)
			}
		}
	}
	sweepTxs = append(sweepTxs, sweepTx)
	return sweepTxs, nil
}
