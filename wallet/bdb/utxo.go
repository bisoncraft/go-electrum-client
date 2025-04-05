package bdb

import (
	"encoding/json"
	"sync"

	"github.com/bisoncraft/go-electrum-client/wallet"
	"github.com/btcsuite/btcd/wire"
	bolt "go.etcd.io/bbolt"
)

type UtxoDB struct {
	db   *bolt.DB
	lock *sync.RWMutex
}

// Put adds a utxo to the database. If the utxo already exists Put updates
// the utxo record.
func (u *UtxoDB) Put(utxo wallet.Utxo) error {
	strOutPoint := utxo.Op.String()
	urec := &utxoRec{
		OutPoint:     strOutPoint,
		AtHeight:     utxo.AtHeight,
		Value:        utxo.Value,
		ScriptPubkey: utxo.ScriptPubkey,
		WatchOnly:    utxo.WatchOnly,
		Frozen:       utxo.Frozen,
	}
	err := u.put(urec)
	if err != nil {
		return err
	}
	return nil
}

// Get gets the utxo given an outpoint. Not part of the Utxo interface.
func (u *UtxoDB) Get(op *wire.OutPoint) (wallet.Utxo, error) {
	var utxo = wallet.Utxo{}
	outPointStr := op.String()
	urec, err := u.get(outPointStr)
	if err != nil {
		return utxo, err
	}
	outPoint, err := wallet.NewOutPointFromString(urec.OutPoint)
	if err != nil {
		return utxo, err
	}
	utxo.Op = *outPoint
	utxo.AtHeight = urec.AtHeight
	utxo.Value = urec.Value
	utxo.ScriptPubkey = urec.ScriptPubkey
	utxo.WatchOnly = urec.WatchOnly
	utxo.Frozen = urec.Frozen
	return utxo, nil
}

// Get gets all the utxos in the utxo bucket in the database.
func (u *UtxoDB) GetAll() ([]wallet.Utxo, error) {
	var ret []wallet.Utxo
	urecList, err := u.getAll()
	if err != nil {
		return nil, err
	}
	for _, urec := range urecList {
		outPoint, err := wallet.NewOutPointFromString(urec.OutPoint)
		if err != nil {
			return nil, err
		}
		utxo := wallet.Utxo{
			Op:           *outPoint,
			AtHeight:     urec.AtHeight,
			Value:        urec.Value,
			ScriptPubkey: urec.ScriptPubkey,
			WatchOnly:    urec.WatchOnly,
			Frozen:       urec.Frozen,
		}
		ret = append(ret, utxo)
	}
	return ret, nil
}

// SetWatchOnly sets this utxo as watch only. It cannot be unset.
func (u *UtxoDB) SetWatchOnly(utxo wallet.Utxo) error {
	strOutPoint := utxo.Op.String()
	urec, err := u.get(strOutPoint)
	if err != nil {
		return err
	}
	urec.WatchOnly = true
	return u.put(urec)
}

// Freeze sets this utxo as frozen. It cannot be used in a transaction.
func (u *UtxoDB) Freeze(utxo wallet.Utxo) error {
	strOutPoint := utxo.Op.String()
	urec, err := u.get(strOutPoint)
	if err != nil {
		return err
	}
	urec.Frozen = true
	return u.put(urec)
}

// UnFreeze resets this utxo as unfrozen. It can again be used in a transaction.
func (u *UtxoDB) UnFreeze(utxo wallet.Utxo) error {
	strOutPoint := utxo.Op.String()
	urec, err := u.get(strOutPoint)
	if err != nil {
		return err
	}
	urec.Frozen = false
	return u.put(urec)
}

// Delete deletes a utxo from the database.
func (u *UtxoDB) Delete(utxo wallet.Utxo) error {
	strOutPoint := utxo.Op.String()
	return u.delete(strOutPoint)
}

// DB access record
type utxoRec struct {
	// Unique key - Used as K & V[OutPoint]
	OutPoint     string `json:"outpoint"`
	AtHeight     int64  `json:"height"`
	Value        int64  `json:"value"`
	ScriptPubkey []byte `json:"pkscript"`
	WatchOnly    bool   `json:"watch_only"`
	Frozen       bool   `json:"frozen"`
}

func (u *UtxoDB) put(urec *utxoRec) error {
	u.lock.Lock()
	defer u.lock.Unlock()

	key := []byte(urec.OutPoint)
	value, err := json.Marshal(urec)
	if err != nil {
		return err
	}

	e := u.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(utxosBkt)
		if b == nil {
			return ErrBucketNotFound
		}
		err := b.Put(key, value)
		return err
	})

	return e
}

func (u *UtxoDB) get(outPoint string) (*utxoRec, error) {
	u.lock.RLock()
	defer u.lock.RUnlock()

	key := []byte(outPoint)

	var urec utxoRec
	e := u.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(utxosBkt)
		if b == nil {
			return ErrBucketNotFound
		}
		value := b.Get(key)
		err := json.Unmarshal(value, &urec)
		return err
	})

	return &urec, e
}

func (u *UtxoDB) getAll() ([]utxoRec, error) {
	u.lock.RLock()
	defer u.lock.RUnlock()

	var urecList []utxoRec
	e := u.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(utxosBkt)
		if b == nil {
			return ErrBucketNotFound
		}
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			var urec utxoRec
			err := json.Unmarshal(v, &urec)
			if err != nil {
				return err
			}
			urecList = append(urecList, urec)
		}
		return nil
	})

	return urecList, e
}

func (u *UtxoDB) delete(outPoint string) error {
	u.lock.Lock()
	defer u.lock.Unlock()

	key := []byte(outPoint)

	e := u.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(utxosBkt)
		if b == nil {
			return ErrBucketNotFound
		}
		err := b.Delete(key)
		return err
	})

	return e
}
