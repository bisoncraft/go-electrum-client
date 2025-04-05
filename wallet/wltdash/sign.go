package wltdash

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"os"

	"github.com/bisoncraft/go-electrum-client/wallet"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

const scriptDebug = false

// Sign an unsigned transaction with the wallet
func (w *DashElectrumWallet) SignTx(pw string, info *wallet.SigningInfo) ([]byte, error) {
	if ok := w.storageManager.IsValidPw(pw); !ok {
		return nil, errors.New("invalid password")
	}
	// Note: maybe change this for future CPFP logic, tricky!
	confirmedUtxos, err := w.ListConfirmedUnspent()
	validConfirmedUtxo := func(op wire.OutPoint) (*wallet.Utxo, bool) {
		for _, utxo := range confirmedUtxos {
			if op.Hash.IsEqual(&utxo.Op.Hash) && op.Index == utxo.Op.Index {
				return &utxo, true
			}
		}
		return nil, false
	}
	if err != nil {
		return nil, err
	}
	tx := info.UnsignedTx
	prevOutFetcher := new(txscript.CannedPrevOutputFetcher)
	for idx, input := range tx.TxIn {
		op := input.PreviousOutPoint
		utxo, valid := validConfirmedUtxo(op)
		if !valid {
			return nil, fmt.Errorf("outpoint %s is not valid (maybe not confirmed?)", op.String())
		}
		pkScript, err := txscript.ParsePkScript(utxo.ScriptPubkey)
		if err != nil {
			return nil, err
		}
		address, err := pkScript.Address(w.params)
		if err != nil {
			return nil, err
		}
		key, err := w.txstore.keyManager.GetKeyForScript(address.ScriptAddress())
		if err != nil {
			return nil, err
		}
		defer key.Zero()
		privKey, err := key.ECPrivKey()
		if err != nil {
			return nil, err
		}
		sigHashes := txscript.NewTxSigHashes(tx, prevOutFetcher)
		prevOutScriptTy := txscript.GetScriptClass(pkScript.Script())
		switch prevOutScriptTy {
		case txscript.WitnessV0ScriptHashTy:
			return nil, errors.New("signing P2WSH not (yet) supported")
		case txscript.WitnessV0PubKeyHashTy:
			sig, err := txscript.WitnessSignature(tx, sigHashes, idx, utxo.Value,
				pkScript.Script(), txscript.SigHashAll, privKey, true)
			if err != nil {
				return nil, err
			}
			// add witness
			input.SignatureScript = nil
			input.Witness = append(input.Witness, sig...)
		case txscript.PubKeyHashTy:
			// note we do not really support P2PK for outbound txs
			sig, err := txscript.SignatureScript(tx, idx,
				pkScript.Script(), txscript.SigHashAll, privKey, true)
			if err != nil {
				return nil, err
			}
			// add script sig
			input.SignatureScript = append(input.SignatureScript, sig...)
			input.Witness = nil
		default:
			return nil, fmt.Errorf("signing for script type %v unsupported",
				prevOutScriptTy)
		}
		if info.VerifyTx {
			e, err := txscript.NewDebugEngine(
				// pubkey script
				pkScript.Script(),
				// refund transaction
				tx,
				// transaction input index
				idx,
				txscript.StandardVerifyFlags,
				txscript.NewSigCache(10),
				txscript.NewTxSigHashes(tx, prevOutFetcher),
				utxo.Value,
				prevOutFetcher,
				nil)
			if err != nil {
				panic(err)
			}
			if !scriptDebug {
				err = e.Execute()
				if err != nil {
					panic(err)
				}
			} else {
				stepDebugScript(e)
			}
		}
	}

	txBytes, err := serializeWireTx(tx)
	if err != nil {
		return nil, err
	}
	return txBytes, nil
}

func stepDebugScript(e *txscript.Engine) {
	fmt.Println("Script 0")
	fmt.Println(e.DisasmScript(0))
	fmt.Println("Script 1")
	fmt.Println(e.DisasmScript(1))
	fmt.Printf("End Scripts\n============\n\n")

	for {
		fmt.Println("---------------------------- STACK --------------------------")
		stk := e.GetStack()
		for i, item := range stk {
			if len(item) > 0 {
				fmt.Printf("%d %v\n", i, hex.EncodeToString(item))
			} else {
				fmt.Printf("%d %s\n", i, "<null>")
			}
		}
		fmt.Println("-------------------------- ALT STACK ------------------------")
		astk := e.GetAltStack()
		for i, item := range astk {
			if len(item) > 0 {
				fmt.Printf("%d %v\n", i, hex.EncodeToString(item))
			} else {
				fmt.Printf("%d %s\n", i, "<null>")
			}
		}
		fmt.Println("--------------------------- Next Op -------------------------")
		fmt.Println(e.DisasmPC())
		fmt.Println("===========")
		script, err := e.DisasmScript(2)
		if err == nil {
			fmt.Printf("script 2: \n%s\n", script)
		}
		fmt.Println("..........")

		// STEP
		done, err := e.Step()
		if err != nil {
			fmt.Printf("Engine Error: %v\n", err)
			os.Exit(2)
		}

		if done {
			fmt.Println("----------------------- Last STACK ------------------------")
			stkerr := false
			stkerrtxt := ""
			stk = e.GetStack()
			for i, item := range stk {
				fmt.Println(i, hex.EncodeToString(item))
				if i == 0 && !bytes.Equal(item, []byte{0x01}) {
					stkerr = true
					stkerrtxt += "ToS Not '1'"
				}
				if i > 0 {
					stkerr = true
					stkerrtxt += " too many stack items left on stack"
				}
			}
			if stkerr {
				fmt.Println(stkerrtxt)
				os.Exit(3)
			}
			fmt.Println("--------------------- End Last STACK ------------------------")

			// senang
			break
		}
	}
}
