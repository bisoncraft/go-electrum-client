package bdb

import (
	"errors"
	"fmt"
	"log"
	"path"
	"sync"

	"github.com/bisoncraft/go-electrum-client/wallet"
	bolt "go.etcd.io/bbolt"
)

// This database is a BoltDb implementation of Datastore.

var (
	keysBkt          = []byte("keys")
	utxosBkt         = []byte("utxos")
	stxosBkt         = []byte("stxos")
	txnsBkt          = []byte("txns")
	subscriptionsBkt = []byte("subscriptions")
	configBkt        = []byte("config")
	encBkt           = []byte("enc")
)

var ErrBucketNotFound = errors.New("cannot find bucket")

type BoltDatastore struct {
	keys          wallet.Keys
	utxos         wallet.Utxos
	stxos         wallet.Stxos
	txns          wallet.Txns
	subscriptions wallet.Subscriptions
	cfg           wallet.Cfg
	enc           wallet.Enc
	db            *bolt.DB
	lock          *sync.RWMutex
}

func (s *BoltDatastore) Close() {
	s.db.Close()
}

func Create(dbPath string, readOnly bool) (*BoltDatastore, error) {
	dbPath = path.Join(dbPath, "wallet.bdb")
	options := *bolt.DefaultOptions
	if readOnly {
		options.ReadOnly = true
	}
	bdb, err := bolt.Open(dbPath, 0600, &options)
	if err != nil {
		log.Fatal(err)
	}
	return dbSetup(bdb)
}

func dbSetup(bdb *bolt.DB) (*BoltDatastore, error) {
	l := new(sync.RWMutex)
	boltDB := &BoltDatastore{
		cfg: &CfgDB{
			db:   bdb,
			lock: l,
		},
		enc: &EncDB{
			db:   bdb,
			lock: l,
		},
		keys: &KeysDB{
			db:   bdb,
			lock: l,
		},
		utxos: &UtxoDB{
			db:   bdb,
			lock: l,
		},
		stxos: &StxoDB{
			db:   bdb,
			lock: l,
		},
		txns: &TxnsDB{
			db:   bdb,
			lock: l,
		},
		subscriptions: &SubscriptionsDB{
			db:   bdb,
			lock: l,
		},
		db:   bdb,
		lock: l,
	}
	initDatabaseBuckets(bdb)
	return boltDB, nil
}

func (db *BoltDatastore) Cfg() wallet.Cfg {
	return db.cfg
}
func (db *BoltDatastore) Enc() wallet.Enc {
	return db.enc
}
func (db *BoltDatastore) Keys() wallet.Keys {
	return db.keys
}
func (db *BoltDatastore) Utxos() wallet.Utxos {
	return db.utxos
}
func (db *BoltDatastore) Stxos() wallet.Stxos {
	return db.stxos
}
func (db *BoltDatastore) Txns() wallet.Txns {
	return db.txns
}
func (db *BoltDatastore) Subscriptions() wallet.Subscriptions {
	return db.subscriptions
}

//		create table if not exists keys (scriptAddress text primary key not null, purpose integer, keyIndex integer, used integer);
//		create table if not exists utxos (outpoint text primary key not null, value integer, height integer, scriptPubKey text, watchOnly integer, frozen integer);
//		create table if not exists stxos (outpoint text primary key not null, value integer, height integer, scriptPubKey text, watchOnly integer, spendHeight integer, spendTxid text);
//		create table if not exists txns (txid text primary key not null, value integer, height integer, timestamp integer, watchOnly integer, tx blob);
//		create table if not exists subscriptions (scriptPubKey text primary key not null, electrumScripthash text, address text);
//		create table if not exists config(key text primary key not null, value blob);
//		create table if not exists enc(key text primary key not null, value blob);

func initDatabaseBuckets(db *bolt.DB) error {
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(keysBkt)
		if err != nil {
			return fmt.Errorf("create keys bucket: %s", err)
		}
		return nil
	})
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(utxosBkt)
		if err != nil {
			return fmt.Errorf("create utxos bucket: %s", err)
		}
		return nil
	})
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(stxosBkt)
		if err != nil {
			return fmt.Errorf("create stxos bucket: %s", err)
		}
		return nil
	})
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(txnsBkt)
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(subscriptionsBkt)
		if err != nil {
			return fmt.Errorf("create subscriptions bucket: %s", err)
		}
		return nil
	})
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(configBkt)
		if err != nil {
			return fmt.Errorf("create config bucket: %s", err)
		}
		return nil
	})
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(encBkt)
		if err != nil {
			return fmt.Errorf("create enc bucket: %s", err)
		}
		return nil
	})
	return nil
}
