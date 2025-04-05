package bdb

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/bisoncraft/go-electrum-client/wallet"
	bolt "go.etcd.io/bbolt"
)

type TxnsDB struct {
	db   *bolt.DB
	lock *sync.RWMutex
}

func (t *TxnsDB) Put(txn []byte, txid string, value int64, height int64, timestamp time.Time, watchOnly bool) error {
	tsb, err := timestamp.GobEncode()
	if err != nil {
		return err
	}
	trec := &txnRec{
		Txid:      txid,
		Height:    height,
		Value:     value,
		Timestamp: tsb,
		WatchOnly: watchOnly,
		RawTx:     txn,
	}
	return t.put(trec)
}

func (t *TxnsDB) Get(txid string) (wallet.Txn, error) {
	txn := wallet.Txn{}
	trec, err := t.get(txid)
	if err != nil {
		return txn, err
	}
	timestamp := time.Time{}
	err = timestamp.GobDecode(trec.Timestamp)
	if err != nil {
		return txn, err
	}
	txn.Txid = trec.Txid
	txn.Value = trec.Value
	txn.Height = trec.Height
	txn.Timestamp = timestamp
	txn.WatchOnly = trec.WatchOnly
	txn.Bytes = trec.RawTx

	return txn, nil
}

func (t *TxnsDB) GetAll(includeWatchOnly bool) ([]wallet.Txn, error) {
	var ret []wallet.Txn
	trecList, err := t.getAll()
	if err != nil {
		return nil, err
	}
	for _, trec := range trecList {
		if trec.WatchOnly && !includeWatchOnly {
			continue
		}
		timestamp := time.Time{}
		err = timestamp.GobDecode(trec.Timestamp)
		if err != nil {
			return nil, err
		}
		txn := wallet.Txn{
			Txid:      trec.Txid,
			Value:     trec.Value,
			Height:    trec.Height,
			Timestamp: timestamp,
			WatchOnly: trec.WatchOnly,
			Bytes:     trec.RawTx,
		}
		ret = append(ret, txn)
	}
	return ret, nil
}

func (t *TxnsDB) Delete(txid string) error {
	return t.delete(txid)
}

func (t *TxnsDB) UpdateHeight(txid string, height int, timestamp time.Time) error {
	trec, err := t.get(txid)
	if err != nil {
		return err
	}
	tsb, err := timestamp.GobEncode()
	if err != nil {
		return err
	}
	trec.Timestamp = tsb
	trec.Height = int64(height)
	return t.put(trec)
}

// DB access record
type txnRec struct {
	// Unique key - Used as K & V[Txid]
	Txid      string `json:"txid"`
	Height    int64  `json:"height"`
	Value     int64  `json:"value"`
	Timestamp []byte `json:"timestamp"`
	WatchOnly bool   `json:"watch_only"`
	RawTx     []byte `json:"rawtx,omitempty"`
}

func (t *TxnsDB) put(trec *txnRec) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	key := []byte(trec.Txid)
	value, err := json.Marshal(trec)
	if err != nil {
		return err
	}

	e := t.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(txnsBkt)
		if b == nil {
			return ErrBucketNotFound
		}
		err := b.Put(key, value)
		return err
	})

	return e
}

func (t *TxnsDB) get(Txid string) (*txnRec, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	key := []byte(Txid)
	var trec txnRec
	e := t.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(txnsBkt)
		if b == nil {
			return ErrBucketNotFound
		}
		value := b.Get(key)
		err := json.Unmarshal(value, &trec)
		return err
	})

	return &trec, e
}

func (t *TxnsDB) getAll() ([]txnRec, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	var trecList []txnRec
	e := t.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(txnsBkt)
		if b == nil {
			return ErrBucketNotFound
		}
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			var trec txnRec
			err := json.Unmarshal(v, &trec)
			if err != nil {
				return err
			}
			trecList = append(trecList, trec)
		}
		return nil
	})

	return trecList, e
}

func (t *TxnsDB) delete(txid string) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	key := []byte(txid)

	e := t.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(txnsBkt)
		if b == nil {
			return ErrBucketNotFound
		}
		err := b.Delete(key)
		return err
	})

	return e
}
