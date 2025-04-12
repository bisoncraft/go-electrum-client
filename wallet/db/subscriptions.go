package db

import (
	"database/sql"
	"sync"

	"github.com/bisoncraft/go-electrum-client/wallet"
)

type SubscriptionsDB struct {
	db   *sql.DB
	lock *sync.RWMutex
}

func (s *SubscriptionsDB) Put(subscription *wallet.Subscription) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	tx, _ := s.db.Begin()
	stmt, err := tx.Prepare("insert or replace into subscriptions(scriptPubKey, electrumScripthash, address) values(?,?,?)")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(subscription.PkScript, subscription.ElectrumScripthash, subscription.Address)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (s *SubscriptionsDB) Get(scriptPubKey string) (*wallet.Subscription, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	var sub *wallet.Subscription
	stmt, err := s.db.Prepare("select electrumScripthash, address from subscriptions where scriptPubKey=?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	var electrumScripthash string
	var address string
	err = stmt.QueryRow(scriptPubKey).Scan(&electrumScripthash, &address)
	if err != nil {
		return nil, err
	}
	sub = &wallet.Subscription{
		PkScript:           scriptPubKey,
		ElectrumScripthash: electrumScripthash,
		Address:            address,
	}
	return sub, nil
}

func (s *SubscriptionsDB) GetElectrumScripthash(electrumScripthash string) (*wallet.Subscription, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	var sub *wallet.Subscription
	stmt, err := s.db.Prepare("select scriptPubKey, address from subscriptions where electrumScripthash=?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	var scriptPubKey string
	var address string
	err = stmt.QueryRow(electrumScripthash).Scan(&scriptPubKey, &address)
	if err != nil {
		return nil, err
	}
	sub = &wallet.Subscription{
		PkScript:           scriptPubKey,
		ElectrumScripthash: electrumScripthash,
		Address:            address,
	}
	return sub, nil
}

func (s *SubscriptionsDB) GetAll() ([]*wallet.Subscription, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	var subs []*wallet.Subscription
	stm := "select scriptPubKey, electrumScripthash, address from subscriptions"
	rows, err := s.db.Query(stm)
	if err != nil {
		return subs, err
	}
	defer rows.Close()
	for rows.Next() {
		var scriptPubKey string
		var electrumScripthash string
		var address string
		if err := rows.Scan(&scriptPubKey, &electrumScripthash, &address); err != nil {
			continue
		}
		sub := &wallet.Subscription{
			PkScript:           scriptPubKey,
			ElectrumScripthash: electrumScripthash,
			Address:            address,
		}
		subs = append(subs, sub)
	}
	return subs, nil
}

func (s *SubscriptionsDB) Delete(scriptPubKey string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	_, err := s.db.Exec("delete from subscriptions where scriptPubKey=?", scriptPubKey)
	return err
}
