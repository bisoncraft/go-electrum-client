package bdb

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/bisoncraft/go-electrum-client/wallet"
	bolt "go.etcd.io/bbolt"
)

type SubscriptionsDB struct {
	db   *bolt.DB
	lock *sync.RWMutex
}

func (s *SubscriptionsDB) Put(subscription *wallet.Subscription) error {
	srec := &subRec{
		ScriptPubKey:       subscription.PkScript,
		ElectrumScripthash: subscription.ElectrumScripthash,
		Address:            subscription.Address,
	}
	return s.put(srec)
}

func (s *SubscriptionsDB) Get(scriptPubKey string) (*wallet.Subscription, error) {
	srec, err := s.get(scriptPubKey)
	if err != nil {
		return nil, err
	}
	sub := &wallet.Subscription{
		PkScript:           srec.ScriptPubKey,
		ElectrumScripthash: srec.ElectrumScripthash,
		Address:            srec.Address,
	}
	return sub, nil
}

func (s *SubscriptionsDB) GetElectrumScripthash(electrumScripthash string) (*wallet.Subscription, error) {
	srecList, err := s.getAll()
	if err != nil {
		return nil, err
	}
	for _, srec := range srecList {
		if electrumScripthash == srec.ElectrumScripthash {
			return &wallet.Subscription{
				PkScript:           srec.ScriptPubKey,
				ElectrumScripthash: srec.ElectrumScripthash,
				Address:            srec.Address,
			}, nil
		}
	}
	return nil, errors.New("electrum script hash not found")
}

func (s *SubscriptionsDB) GetAll() ([]*wallet.Subscription, error) {
	var subs []*wallet.Subscription
	srecList, err := s.getAll()
	if err != nil {
		return nil, err
	}
	for _, srec := range srecList {
		sub := &wallet.Subscription{
			PkScript:           srec.ScriptPubKey,
			ElectrumScripthash: srec.ElectrumScripthash,
			Address:            srec.Address,
		}
		subs = append(subs, sub)
	}
	return subs, nil
}

func (s *SubscriptionsDB) Delete(scriptPubKey string) error {
	return s.delete(scriptPubKey)
}

// DB access record
type subRec struct {
	// Unique key - Used as K & V[ScriptPubkey]
	ScriptPubKey       string `json:"pkscript"`
	ElectrumScripthash string `json:"electrum_scripthash"`
	Address            string `json:"address"`
}

func (s *SubscriptionsDB) put(srec *subRec) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	key := []byte(srec.ScriptPubKey)
	value, err := json.Marshal(srec)
	if err != nil {
		return err
	}

	e := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(subscriptionsBkt)
		if b == nil {
			return ErrBucketNotFound
		}
		err := b.Put(key, value)
		return err
	})

	return e
}

func (s *SubscriptionsDB) get(scriptPubKey string) (*subRec, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	key := []byte(scriptPubKey)

	var srec subRec
	e := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(subscriptionsBkt)
		if b == nil {
			return ErrBucketNotFound
		}
		value := b.Get(key)
		err := json.Unmarshal(value, &srec)
		return err
	})

	return &srec, e
}

func (s *SubscriptionsDB) getAll() ([]subRec, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var srecList []subRec
	e := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(subscriptionsBkt)
		if b == nil {
			return ErrBucketNotFound
		}
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			var srec subRec
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

func (s *SubscriptionsDB) delete(scriptPubKey string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	key := []byte(scriptPubKey)

	e := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(subscriptionsBkt)
		if b == nil {
			return ErrBucketNotFound
		}
		err := b.Delete(key)
		return err
	})

	return e
}
