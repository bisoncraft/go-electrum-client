package bdb

import (
	"encoding/json"
	"sync"

	"github.com/bisoncraft/go-electrum-client/wallet"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	bolt "go.etcd.io/bbolt"
)

type StxoDB struct {
	db   *bolt.DB
	lock *sync.RWMutex
}

func (s *StxoDB) Put(stxo wallet.Stxo) error {
	utxo := stxo.Utxo
	outPointstr := utxo.Op.String()
	srec := stxoRec{
		OutPoint:     outPointstr,
		AtHeight:     utxo.AtHeight,
		Value:        utxo.Value,
		ScriptPubkey: utxo.ScriptPubkey,
		WatchOnly:    utxo.WatchOnly,
		Frozen:       utxo.Frozen,
		SpendHeight:  stxo.SpendHeight,
		SpendTxid:    stxo.SpendTxid.String(),
	}
	return s.put(&srec)
}

func (s *StxoDB) GetAll() ([]wallet.Stxo, error) {
	srecList, err := s.getAll()
	if err != nil {
		return nil, err
	}
	ret := []wallet.Stxo{}
	for _, srec := range srecList {
		outPoint, err := wallet.NewOutPointFromString(srec.OutPoint)
		if err != nil {
			return nil, err
		}
		spendTxid, err := chainhash.NewHashFromStr(srec.SpendTxid)
		if err != nil {
			return nil, err
		}
		stxo := wallet.Stxo{
			Utxo: wallet.Utxo{
				Op:           *outPoint,
				AtHeight:     srec.AtHeight,
				Value:        srec.Value,
				ScriptPubkey: srec.ScriptPubkey,
				WatchOnly:    srec.WatchOnly,
				Frozen:       srec.Frozen,
			},
			SpendHeight: srec.SpendHeight,
			SpendTxid:   *spendTxid,
		}
		ret = append(ret, stxo)
	}
	return ret, nil
}

func (s *StxoDB) Delete(stxo wallet.Stxo) error {
	utxo := stxo.Utxo
	outPointstr := utxo.Op.String()
	return s.delete(outPointstr)
}

// DB access record
type stxoRec struct {
	// UTXO
	// Unique key - Used as K & V[OutPoint]
	OutPoint     string `json:"outpoint"`
	AtHeight     int64  `json:"height"`
	Value        int64  `json:"value"`
	ScriptPubkey []byte `json:"pkscript"`
	WatchOnly    bool   `json:"watch_only"`
	Frozen       bool   `json:"frozen"`
	// STXO
	SpendHeight int64  `json:"spend_height"`
	SpendTxid   string `json:"spend_txid"`
}

func (s *StxoDB) put(srec *stxoRec) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	key := []byte(srec.OutPoint)
	value, err := json.Marshal(srec)
	if err != nil {
		return err
	}

	e := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(stxosBkt)
		if b == nil {
			return ErrBucketNotFound
		}
		err := b.Put(key, value)
		return err
	})

	return e
}

func (s *StxoDB) getAll() ([]stxoRec, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var srecList []stxoRec
	e := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(stxosBkt)
		if b == nil {
			return ErrBucketNotFound
		}
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			var srec stxoRec
			err := json.Unmarshal(v, &srec)
			if err != nil {
				return err
			}
			srecList = append(srecList, srec)
		}
		return nil
	})

	return srecList, e
}

func (s *StxoDB) delete(outPoint string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	key := []byte(outPoint)

	e := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(stxosBkt)
		if b == nil {
			return ErrBucketNotFound
		}
		return b.Delete(key)
	})

	return e
}
