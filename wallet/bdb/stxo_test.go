package bdb

import (
	"bytes"
	"os"
	"sync"
	"testing"

	"github.com/bisoncraft/go-electrum-client/wallet"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	bolt "go.etcd.io/bbolt"
)

var sxdb *StxoDB
var stxo wallet.Stxo

func setupSxdb() error {
	bdb, err := bolt.Open("test.bdb", 0600, nil)
	if err != nil {
		return nil
	}
	err = initDatabaseBuckets(bdb)
	if err != nil {
		return nil
	}
	sxdb = &StxoDB{
		db:   bdb,
		lock: new(sync.RWMutex),
	}
	sh1, _ := chainhash.NewHashFromStr("e941e1c32b3dd1a68edc3af9f7fe711f35aaca60f758c2dd49561e45ca2c41c0")
	sh2, _ := chainhash.NewHashFromStr("82998e18760a5f6e5573cd789269e7853e3ebaba07a8df0929badd69dc644c5f")
	outpoint := wire.NewOutPoint(sh1, 0)
	utxo := wallet.Utxo{
		Op:           *outpoint,
		AtHeight:     300000,
		Value:        100000000,
		ScriptPubkey: []byte("001427ba894b4ac84cf236608590efe12ee514692c32"),
		WatchOnly:    false,
	}
	stxo = wallet.Stxo{
		Utxo:        utxo,
		SpendHeight: 300100,
		SpendTxid:   *sh2,
	}
	return nil
}

func teardownSxdb() {
	if sxdb == nil {
		return
	}
	sxdb.db.Close()
	os.RemoveAll("test.bdb")
}

func TestStxoPut(t *testing.T) {
	if err := setupSxdb(); err != nil {
		t.Fatal(err)
	}
	defer teardownSxdb()
	err := sxdb.Put(stxo)
	if err != nil {
		t.Error(err)
	}

}

func TestStxoGetAll(t *testing.T) {
	if err := setupSxdb(); err != nil {
		t.Fatal(err)
	}
	defer teardownSxdb()
	err := sxdb.Put(stxo)
	if err != nil {
		t.Error(err)
	}

	stxos, err := sxdb.GetAll()
	if err != nil {
		t.Error(err)
	}
	if stxos == nil {
		t.Fatal("nil list")
	}

	if stxos[0].Utxo.Op.Hash.String() != stxo.Utxo.Op.Hash.String() {
		t.Error("Stxo db returned wrong outpoint hash")
	}
	if stxos[0].Utxo.Op.Index != stxo.Utxo.Op.Index {
		t.Error("Stxo db returned wrong outpoint index")
	}
	if stxos[0].Utxo.Value != stxo.Utxo.Value {
		t.Error("Stxo db returned wrong value")
	}
	if stxos[0].Utxo.AtHeight != stxo.Utxo.AtHeight {
		t.Error("Stxo db returned wrong height")
	}
	if !bytes.Equal(stxos[0].Utxo.ScriptPubkey, stxo.Utxo.ScriptPubkey) {
		t.Error("Stxo db returned wrong scriptPubKey")
	}
	if stxos[0].Utxo.WatchOnly != stxo.Utxo.WatchOnly {
		t.Error("Stxo db returned wrong watch only bool")
	}
	if stxos[0].Utxo.Frozen != stxo.Utxo.Frozen {
		t.Error("Stxo db returned wrong frozen bool")
	}
	if stxos[0].SpendHeight != stxo.SpendHeight {
		t.Error("Stxo db returned wrong spend height")
	}
	if stxos[0].SpendTxid.String() != stxo.SpendTxid.String() {
		t.Error("Stxo db returned wrong spend txid")
	}
}

func TestDeleteStxo(t *testing.T) {
	if err := setupSxdb(); err != nil {
		t.Fatal(err)
	}
	defer teardownSxdb()
	err := sxdb.Put(stxo)
	if err != nil {
		t.Error(err)
	}
	err = sxdb.Delete(stxo)
	if err != nil {
		t.Error(err)
	}
	stxos, err := sxdb.GetAll()
	if err != nil {
		t.Error(err)
	}
	if len(stxos) != 0 {
		t.Error("Stxo db delete failed")
	}
}
