package db

import (
	"database/sql"
	"sync"
	"time"

	"github.com/bisoncraft/go-electrum-client/wallet"
)

type TxnsDB struct {
	db   *sql.DB
	lock *sync.RWMutex
}

func (t *TxnsDB) Put(txn []byte, txid string, value int64, height int64, timestamp time.Time, watchOnly bool) error {
	t.lock.Lock()
	defer t.lock.Unlock()
	tx, err := t.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("insert or replace into txns(txid, value, height, timestamp, watchOnly, tx) values(?,?,?,?,?,?)")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()
	watchOnlyInt := 0
	if watchOnly {
		watchOnlyInt = 1
	}
	_, err = stmt.Exec(txid, value, height, int(timestamp.Unix()), watchOnlyInt, txn)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (t *TxnsDB) Get(txid string) (wallet.Txn, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	var txn wallet.Txn
	stmt, err := t.db.Prepare("select tx, value, height, timestamp, watchOnly from txns where txid=?")
	if err != nil {
		return txn, err
	}
	defer stmt.Close()
	var ret []byte
	var value int
	var height int
	var timestamp int
	var watchOnlyInt int
	err = stmt.QueryRow(txid).Scan(&ret, &value, &height, &timestamp, &watchOnlyInt)
	if err != nil {
		return txn, err
	}
	watchOnly := false
	if watchOnlyInt > 0 {
		watchOnly = true
	}
	txn = wallet.Txn{
		Txid:      txid,
		Value:     int64(value),
		Height:    int64(height),
		Timestamp: time.Unix(int64(timestamp), 0),
		WatchOnly: watchOnly,
		Bytes:     ret,
	}
	return txn, nil
}

func (t *TxnsDB) GetAll(includeWatchOnly bool) ([]wallet.Txn, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	var ret []wallet.Txn
	stm := "select txid, tx, value, height, timestamp, watchOnly from txns"
	rows, err := t.db.Query(stm)
	if err != nil {
		return ret, err
	}
	defer rows.Close()
	for rows.Next() {
		var txid string
		var tx []byte
		var value int
		var height int
		var timestamp int
		var watchOnlyInt int
		if err := rows.Scan(&txid, &tx, &value, &height, &timestamp, &watchOnlyInt); err != nil {
			continue
		}
		watchOnly := false
		if watchOnlyInt > 0 {
			if !includeWatchOnly {
				continue
			}
			watchOnly = true
		}

		txn := wallet.Txn{
			Txid:      txid,
			Value:     int64(value),
			Height:    int64(height),
			Timestamp: time.Unix(int64(timestamp), 0),
			WatchOnly: watchOnly,
			Bytes:     tx,
		}
		ret = append(ret, txn)
	}
	return ret, nil
}

func (t *TxnsDB) Delete(txid string) error {
	t.lock.Lock()
	defer t.lock.Unlock()
	_, err := t.db.Exec("delete from txns where txid=?", txid)
	if err != nil {
		return err
	}
	return nil
}

func (t *TxnsDB) UpdateHeight(txid string, height int, timestamp time.Time) error {
	t.lock.Lock()
	defer t.lock.Unlock()
	tx, err := t.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("update txns set height=?, timestamp=? where txid=?")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(height, int(timestamp.Unix()), txid)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}
