package wltdash

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bisoncraft/go-electrum-client/wallet"
	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

type TxStore struct {
	adrs       []btcutil.Address
	txids      map[string]int64 // txid => height
	txidsMutex *sync.RWMutex
	addrMutex  *sync.Mutex
	cbMutex    *sync.Mutex

	keyManager *KeyManager

	params *chaincfg.Params

	wallet.Datastore
}

func NewTxStore(params *chaincfg.Params, db wallet.Datastore, keyManager *KeyManager) (*TxStore, error) {
	txs := &TxStore{
		params:     params,
		keyManager: keyManager,
		addrMutex:  new(sync.Mutex),
		cbMutex:    new(sync.Mutex),
		txidsMutex: new(sync.RWMutex),
		txids:      make(map[string]int64),
		Datastore:  db,
	}
	txs.PopulateAdrs()
	return txs, nil
}

// PopulateAdrs makes a list of addresses from the key-index pairs we have in the
// database. The key-pairs we have stored are returned in index order. PopulateAdrs
// also makes an up to date list of txs in the database. It never mutates the db.
func (ts *TxStore) PopulateAdrs() {
	keys := ts.keyManager.GetKeys()
	ts.addrMutex.Lock()
	ts.adrs = []btcutil.Address{}
	for _, k := range keys {
		address, err := k.Address(ts.params)
		if err != nil {
			continue
		}
		script := address.ScriptAddress()
		segwitAddress, swerr := btcutil.NewAddressWitnessPubKeyHash(script, ts.params)
		if swerr != nil {
			fmt.Println(swerr)
			continue
		}
		ts.adrs = append(ts.adrs, segwitAddress)
		k.Zero()
	}
	ts.addrMutex.Unlock()

	txns, _ := ts.Txns().GetAll(true)
	ts.txidsMutex.Lock()
	for _, t := range txns {
		ts.txids[t.Txid] = t.Height
	}
	ts.txidsMutex.Unlock()
}

// AddTransaction puts a tx into the database.
func (ts *TxStore) AddTransaction(tx *wire.MsgTx, height int64, timestamp time.Time) (uint32, error) {
	var hits uint32
	var err error
	// transaction has been Okay'd by ElectrumX; check tx sanity.
	utilTx := btcutil.NewTx(tx)
	err = blockchain.CheckTransactionSanity(utilTx)
	if err != nil {
		return hits, err
	}

	// check if we've already processed this tx up to confirmed state.
	ts.txidsMutex.RLock()
	sh, ok := ts.txids[tx.TxHash().String()]
	ts.txidsMutex.RUnlock()
	if ok && (sh > 0 || (sh == 0 && height == 0)) {
		return 1, nil
	}

	// check to see if this is a double spend
	doubleSpends, err := ts.CheckDoubleSpends(tx)
	if err != nil {
		return hits, err
	}
	if len(doubleSpends) > 0 {
		// First seen rule
		if height == 0 {
			return 0, nil
		} else {
			// mark any unconfirmed doubles as dead
			for _, double := range doubleSpends {
				ts.markAsDead(*double)
			}
		}
	}

	// generate pubkey hash scripts for all addresses in our list
	ts.addrMutex.Lock()
	pkScripts := make([][]byte, len(ts.adrs))
	for i := range ts.adrs {
		pkScripts[i], err = txscript.PayToAddrScript(ts.adrs[i])
		if err != nil {
			ts.addrMutex.Unlock()
			return hits, err
		}
	}
	ts.addrMutex.Unlock()

	// look through all outputs of this tx to see if we increase balance
	thisTxHash := tx.TxHash()
	value := int64(0)
	matchesWatchOnly := false
	for i, txout := range tx.TxOut {
		for _, script := range pkScripts {
			if bytes.Equal(txout.PkScript, script) {
				// new tx output paying to us
				scriptAddress, _ := ts.extractScriptAddress(txout.PkScript)
				ts.keyManager.MarkKeyAsUsed(scriptAddress)
				newop := wire.OutPoint{
					Hash:  thisTxHash,
					Index: uint32(i),
				}
				newu := wallet.Utxo{
					AtHeight:     height,
					Value:        txout.Value,
					ScriptPubkey: txout.PkScript,
					Op:           newop,
					WatchOnly:    false,
				}
				value += newu.Value
				ts.Utxos().Put(newu)
				hits++
				break
			}
		}
	}

	// get updated list with any new utxos added
	utxos, err := ts.Utxos().GetAll()
	if err != nil {
		return 0, err
	}

	// look through all inputs of this tx to see if we decrease balance
	for _, txin := range tx.TxIn {
		for i, u := range utxos {
			if outPointsEqual(txin.PreviousOutPoint, u.Op) {
				// tx input spends our utxo
				st := wallet.Stxo{
					Utxo:        u,
					SpendHeight: height,
					SpendTxid:   thisTxHash,
				}
				ts.Stxos().Put(st)
				ts.Utxos().Delete(u)
				utxos = append(utxos[:i], utxos[i+1:]...)
				if !u.WatchOnly {
					value -= u.Value
					hits++
				} else {
					matchesWatchOnly = true
				}
				break
			}
		}
	}

	// update height of any stxos
	if height > 0 {
		stxos, err := ts.Stxos().GetAll()
		if err != nil {
			return 0, err
		}
		for _, stxo := range stxos {
			if stxo.SpendTxid == thisTxHash {
				stxo.SpendHeight = height
				ts.Stxos().Put(stxo)
				if !stxo.Utxo.WatchOnly {
					hits++
				} else {
					matchesWatchOnly = true
				}
				break
			}
		}
	}

	// if hits is nonzero it's a relevant tx and we should store it
	if hits > 0 || matchesWatchOnly {
		ts.cbMutex.Lock()
		ts.txidsMutex.Lock()
		txn, err := ts.Txns().Get(tx.TxHash().String())
		if err != nil {
			// new txn{}
			txn.Timestamp = timestamp
			var buf bytes.Buffer
			tx.BtcEncode(&buf, wire.ProtocolVersion, wire.WitnessEncoding)
			ts.Txns().Put(buf.Bytes(), tx.TxHash().String(), value, height, txn.Timestamp, hits == 0)
			ts.txids[tx.TxHash().String()] = height
		}
		// check the height before committing so we don't allow rogue electrumX servers
		// to send us a loose tx that resets our height to zero.
		if err == nil && txn.Height <= 0 {
			ts.Txns().UpdateHeight(tx.TxHash().String(), int(height), txn.Timestamp)
			ts.txids[tx.TxHash().String()] = height
		}
		ts.txidsMutex.Unlock()
		ts.cbMutex.Unlock()
		ts.PopulateAdrs()
		hits++
	}
	return hits, err
}

func (ts *TxStore) markAsDead(txid chainhash.Hash) error {
	stxos, err := ts.Stxos().GetAll()
	if err != nil {
		return err
	}
	markStxoAsDead := func(s wallet.Stxo) error {
		err := ts.Stxos().Delete(s)
		if err != nil {
			return err
		}
		err = ts.Txns().UpdateHeight(s.SpendTxid.String(), -1, time.Now())
		if err != nil {
			return err
		}
		return nil
	}
	for _, s := range stxos {
		// if an stxo is marked dead, move it back into the utxo table/bucket
		if txid == s.SpendTxid {
			if err := markStxoAsDead(s); err != nil {
				return err
			}
			if err := ts.Utxos().Put(s.Utxo); err != nil {
				return err
			}
		}
		// if a dependency of the spend is dead then mark the spend as dead
		if txid.IsEqual(&s.Utxo.Op.Hash) {
			if err := markStxoAsDead(s); err != nil {
				return err
			}
			if err := ts.markAsDead(s.SpendTxid); err != nil {
				return err
			}
		}
	}
	utxos, err := ts.Utxos().GetAll()
	if err != nil {
		return err
	}
	// dead utxos should just be deleted
	for _, u := range utxos {
		if txid.IsEqual(&u.Op.Hash) {
			err := ts.Utxos().Delete(u)
			if err != nil {
				return err
			}
		}
	}
	ts.Txns().UpdateHeight(txid.String(), -1, time.Now())
	return nil
}

// CheckDoubleSpends takes a transaction and compares it with all transactions
// in the db. It returns a slice of all txids in the db which are double spent
// by the received tx.
func (ts *TxStore) CheckDoubleSpends(argTx *wire.MsgTx) ([]*chainhash.Hash, error) {
	var dubs []*chainhash.Hash // slice of all double-spent txids
	argTxid := argTx.TxHash()
	txs, err := ts.Txns().GetAll(true)
	if err != nil {
		return dubs, err
	}
	for _, compTx := range txs {
		if compTx.Height < 0 {
			continue
		}
		msgTx, err := newWireTx(compTx.Bytes, true)
		if err != nil {
			return dubs, err
		}
		compTxid := msgTx.TxHash()
		for _, argIn := range argTx.TxIn {
			// iterate through inputs of compTx
			for _, compIn := range msgTx.TxIn {
				if outPointsEqual(argIn.PreviousOutPoint, compIn.PreviousOutPoint) && !compTxid.IsEqual(&argTxid) {
					// found double spend
					dubs = append(dubs, &compTxid)
					break // back to argIn loop
				}
			}
		}
	}
	return dubs, nil
}

func (ts *TxStore) extractScriptAddress(script []byte) ([]byte, error) {
	_, addrs, _, err := txscript.ExtractPkScriptAddrs(script, ts.params)
	if err != nil {
		return nil, err
	}
	if len(addrs) == 0 {
		return nil, errors.New("unknown script")
	}
	return addrs[0].ScriptAddress(), nil
}
