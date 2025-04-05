// Copyright (C) 2015-2016 The Lightning Network Developers

package wltfiro

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/bisoncraft/go-electrum-client/wallet"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/coinset"
	"github.com/btcsuite/btcd/btcutil/txsort"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcwallet/wallet/txauthor"
)

// secretSource is used to locate keys and redemption scripts while signing a
// transaction. secretSource satisfies the txauthor.SecretsSource interface.
type secretSource struct {
	w *FiroElectrumWallet
}

// ChainParams returns the chain parameters.
func (ss *secretSource) ChainParams() *chaincfg.Params {
	return ss.w.params
}

// GetKey fetches a private key for the specified address.
func (ss *secretSource) GetKey(address btcutil.Address) (*btcec.PrivateKey, bool, error) {
	extKey, err := ss.w.keyManager.GetKeyForScript(address.ScriptAddress())
	if err != nil {
		return nil, false, err
	}
	privKey, err := extKey.ECPrivKey()
	if err != nil {
		return nil, false, err
	}
	return privKey, true, nil
}

// GetScript fetches the redemption script for the specified p2sh/p2wsh address.
func (ss *secretSource) GetScript(address btcutil.Address) ([]byte, error) {
	return txscript.PayToAddrScript(address)
}

// satisfies coinset.Coin
type unspentCoin struct {
	TxHash       *chainhash.Hash
	TxIndex      uint32
	TxValue      btcutil.Amount
	TxNumConfs   int64
	ScriptPubKey []byte
}

func (c *unspentCoin) Hash() *chainhash.Hash { return c.TxHash }
func (c *unspentCoin) Index() uint32         { return c.TxIndex }
func (c *unspentCoin) Value() btcutil.Amount { return c.TxValue }
func (c *unspentCoin) PkScript() []byte      { return c.ScriptPubKey }
func (c *unspentCoin) NumConfs() int64       { return c.TxNumConfs }
func (c *unspentCoin) ValueAge() int64       { return int64(c.TxValue) * c.TxNumConfs }

func newUnspentCoin(txHash *chainhash.Hash, index uint32, value btcutil.Amount, numConfs int64, scriptPubKey []byte) coinset.Coin {
	unspent := &unspentCoin{
		TxHash:       (*chainhash.Hash)(txHash.CloneBytes()),
		TxIndex:      index,
		TxValue:      value,
		TxNumConfs:   numConfs,
		ScriptPubKey: scriptPubKey,
	}
	return coinset.Coin(unspent)
}

// gatherCoins aggregates acceptable utxos into a alice of coinset.Coin's
func (w *FiroElectrumWallet) gatherCoins(excludeUnconfirmed bool) []coinset.Coin {
	tip := w.blockchainTip
	utxos, _ := w.txstore.Utxos().GetAll()
	var unspentCoins []coinset.Coin
	for _, u := range utxos {
		if u.WatchOnly {
			continue
		}
		if u.Frozen {
			continue
		}
		if excludeUnconfirmed && u.AtHeight <= 0 {
			continue
		}
		var confirmations int64
		if u.AtHeight > 0 {
			confirmations = tip - u.AtHeight
		}
		unspent := newUnspentCoin(&u.Op.Hash, u.Op.Index, btcutil.Amount(u.Value), confirmations, u.ScriptPubkey)
		unspentCoins = append(unspentCoins, unspent)
	}
	return unspentCoins
}

// Spend creates and signs a new transaction from wallet coins
func (w *FiroElectrumWallet) Spend(
	pw string,
	amount int64,
	address btcutil.Address,
	feeLevel wallet.FeeLevel) (int, *wire.MsgTx, error) {

	if ok := w.storageManager.IsValidPw(pw); !ok {
		return -1, nil, errors.New("invalid password")
	}

	changeIndex, tx, err := w.buildTx(amount, address, feeLevel)
	if err != nil {
		return -1, nil, err
	}
	return changeIndex, tx, nil
}

// buildTx builds a normal Pay to (witness) pubkey hash transaction.
func (w *FiroElectrumWallet) buildTx(
	amount int64,
	address btcutil.Address,
	feeLevel wallet.FeeLevel) (int, *wire.MsgTx, error) {

	// Check for dust
	if w.IsDust(amount) {
		return -1, nil, wallet.ErrDustAmount
	}

	// check payto address
	script, err := txscript.PayToAddrScript(address)
	if err != nil {
		return -1, nil, err
	}

	// create input source
	coins := w.gatherCoins(true)
	for i, coin := range coins {
		fmt.Println(i, coin.Hash().String(), coin.Index(), coin.PkScript())
	}

	var prevScripts map[wire.OutPoint]*wire.TxOut

	inputSource := func(target btcutil.Amount) (
		total btcutil.Amount,
		inputs []*wire.TxIn,
		inputValues []btcutil.Amount,
		scripts [][]byte, err error) {

		coinSelector := coinset.MaxValueAgeCoinSelector{MaxInputs: 10000, MinChangeAmount: btcutil.Amount(0)}
		coins, err := coinSelector.CoinSelect(target, coins)
		if err != nil {
			return total, inputs, []btcutil.Amount{}, scripts, wallet.ErrInsufficientFunds
		}
		prevScripts = make(map[wire.OutPoint]*wire.TxOut)
		for _, c := range coins.Coins() {
			total += c.Value()
			outpoint := wire.NewOutPoint(c.Hash(), c.Index())
			in := wire.NewTxIn(outpoint, []byte{}, [][]byte{})
			in.Sequence = uint32(0xffffffff)
			inputs = append(inputs, in)
			prevScripts[*outpoint] = wire.NewTxOut(int64(c.Value()), c.PkScript())
		}
		return total, inputs, []btcutil.Amount{}, scripts, nil
	}

	// Get the fee per kilobyte
	feePerKB := int64(w.GetFeePerByte(feeLevel)) * 1000

	// outputs
	out := wire.NewTxOut(amount, script)

	// create change source
	changeSource := func() ([]byte, error) {
		address, err := w.GetUnusedAddress(wallet.CHANGE)
		if err != nil {
			return []byte{}, err
		}
		script, err := txscript.PayToAddrScript(address)
		if err != nil {
			return []byte{}, err
		}
		return script, nil
	}
	var scriptSize int = 33 // max size, over-estimated
	changeOutputsSource := txauthor.ChangeSource{
		NewScript:  changeSource,
		ScriptSize: scriptSize,
	}

	outputs := []*wire.TxOut{out}

	authoredTx, err := txauthor.NewUnsignedTransaction(
		outputs,
		btcutil.Amount(feePerKB),
		inputSource,
		&changeOutputsSource)
	if err != nil {
		return -1, nil, err
	}

	// BIP 69 sorting
	txsort.InPlaceSort(authoredTx.Tx)

	b := make([]byte, 0, 300)
	br := bytes.NewBuffer(b)
	authoredTx.Tx.Serialize(br)
	fmt.Println("unsigned tx:", hex.EncodeToString(br.Bytes()))

	// Sign
	var prevPkScripts [][]byte
	var inputValues []btcutil.Amount
	for _, txIn := range authoredTx.Tx.TxIn {
		op := txIn.PreviousOutPoint
		prevOut := prevScripts[op]
		inputValues = append(inputValues, btcutil.Amount(prevOut.Value))
		prevPkScripts = append(prevPkScripts, prevOut.PkScript)
		// Zero the previous witness and signature script or else
		// AddAllInputScripts does some weird stuff.
		txIn.SignatureScript = nil
		txIn.Witness = nil
	}
	err = txauthor.AddAllInputScripts(authoredTx.Tx, prevPkScripts, inputValues, &secretSource{w})
	if err != nil {
		return -1, nil, err
	}
	return authoredTx.ChangeIndex, authoredTx.Tx, nil
}

func (w *FiroElectrumWallet) GetFeePerByte(feeLevel wallet.FeeLevel) int64 {
	return w.feeProvider.GetFeePerByte(feeLevel)
}

func (w *FiroElectrumWallet) EstimateFee(
	ins []wallet.InputInfo,
	outs []wallet.TransactionOutput,
	feePerByte int64) int64 {

	tx := new(wire.MsgTx)
	for _, out := range outs {
		scriptPubKey, _ := txscript.PayToAddrScript(out.Address)
		output := wire.NewTxOut(out.Value, scriptPubKey)
		tx.TxOut = append(tx.TxOut, output)
	}
	estimatedSize := EstimateSerializeSize(len(ins), tx.TxOut, false, P2PKH)
	fee := estimatedSize * int(feePerByte)
	return int64(fee)
}

//TODO: FUTURE:
// func LockTimeFromRedeemScript(redeemScript []byte) (uint32, error) {
// 	if len(redeemScript) < 113 {
// 		return 0, errors.New("redeem script invalid length")
// 	}
// 	if redeemScript[106] != 103 {
// 		return 0, errors.New("rnvalid redeem script")
// 	}
// 	if redeemScript[107] == 0 {
// 		return 0, nil
// 	}
// 	if 81 <= redeemScript[107] && redeemScript[107] <= 96 {
// 		return uint32((redeemScript[107] - 81) + 1), nil
// 	}
// 	var v []byte
// 	op := redeemScript[107]
// 	if 1 <= op && op <= 75 {
// 		for i := 0; i < int(op); i++ {
// 			v = append(v, []byte{redeemScript[108+i]}...)
// 		}
// 	} else {
// 		return 0, errors.New("too many bytes pushed for sequence")
// 	}
// 	var result int64
// 	for i, val := range v {
// 		result |= int64(val) << uint8(8*i)
// 	}

// 	return uint32(result), nil
// }
