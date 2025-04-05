package bdb

import (
	"bytes"
	"encoding/hex"
	"sync"
	"testing"

	"github.com/bisoncraft/go-electrum-client/wallet"
	bolt "go.etcd.io/bbolt"
)

var uxdb *UtxoDB

func setupUxdb() error {
	bdb, err := bolt.Open("test.bdb", 0600, nil)
	if err != nil {
		return nil
	}
	err = initDatabaseBuckets(bdb)
	if err != nil {
		return nil
	}
	uxdb = &UtxoDB{
		db:   bdb,
		lock: new(sync.RWMutex),
	}
	return nil
}

func teardownUxdb() {
	if uxdb == nil {
		return
	}
	uxdb.db.Close()
}

func TestUtxoPut(t *testing.T) {
	if err := setupUxdb(); err != nil {
		t.Fatal(err)
	}
	defer teardownUxdb()
	outPointStr := "b721c368f9ddb1d6d0d225fdb22e8f2b4b9f0fed160ea6ae80270b7849c2d62e:0"
	outPoint, err := wallet.NewOutPointFromString(outPointStr)
	if err != nil {
		t.Fatal(err)
	}
	scriptPubKey, err := hex.DecodeString("001427ba894b4ac84cf236608590efe12ee514692c32")
	if err != nil {
		t.Fatal(err)
	}
	utxo := wallet.Utxo{
		Op:           *outPoint,
		AtHeight:     0,
		Value:        300000000,
		ScriptPubkey: scriptPubKey,
		WatchOnly:    false,
		Frozen:       false,
	}
	err = uxdb.Put(utxo)
	if err != nil {
		t.Error(err)
	}

	u, err := uxdb.Get(outPoint)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(u.Op.Hash[:], utxo.Op.Hash[:]) {
		t.Error("Utxo db returned wrong outpoint tx hash")
	}
	if u.Op.Index != utxo.Op.Index {
		t.Error("Utxo db returned wrong outpoint index")
	}
	if u.Value != utxo.Value {
		t.Error("Utxo db returned wrong value")
	}
	if u.AtHeight != utxo.AtHeight {
		t.Error("Utxo db returned wrong height")
	}
	if !bytes.Equal(u.ScriptPubkey, utxo.ScriptPubkey) {
		t.Error("Utxo db returned wrong scriptPubKey")
	}
}

func TestUtxoGetAll(t *testing.T) {
	if err := setupUxdb(); err != nil {
		t.Fatal(err)
	}
	defer teardownUxdb()
	outPointStr := "b721c368f9ddb1d6d0d225fdb22e8f2b4b9f0fed160ea6ae80270b7849c2d62e:0"
	outPoint, err := wallet.NewOutPointFromString(outPointStr)
	if err != nil {
		t.Fatal(err)
	}
	scriptPubKey, err := hex.DecodeString("001427ba894b4ac84cf236608590efe12ee514692c32")
	if err != nil {
		t.Fatal(err)
	}
	utxo := wallet.Utxo{
		Op:           *outPoint,
		AtHeight:     0,
		Value:        300000000,
		ScriptPubkey: scriptPubKey,
		WatchOnly:    false,
		Frozen:       false,
	}
	err = uxdb.Put(utxo)
	if err != nil {
		t.Error(err)
	}

	utxos, err := uxdb.GetAll()
	if err != nil {
		t.Fatal(err)
	}
	if utxos == nil {
		t.Fatal(err)
	}

	if !bytes.Equal(utxos[0].Op.Hash[:], utxo.Op.Hash[:]) {
		t.Error("hash bytes not equal")
	}
	if utxos[0].Op.Hash.String() != utxo.Op.Hash.String() {
		t.Error("Utxo db returned wrong outpoint hash")
	}
	if utxos[0].Op.Index != utxo.Op.Index {
		t.Error("Utxo db returned wrong outpoint index")
	}
	if utxos[0].Value != utxo.Value {
		t.Error("Utxo db returned wrong value")
	}
	if utxos[0].AtHeight != utxo.AtHeight {
		t.Error("Utxo db returned wrong height")
	}
	if !bytes.Equal(utxos[0].ScriptPubkey, utxo.ScriptPubkey) {
		t.Error("Utxo db returned wrong scriptPubKey")
	}
}

func TestSetWatchOnlyUtxo(t *testing.T) {
	if err := setupUxdb(); err != nil {
		t.Fatal(err)
	}
	defer teardownUxdb()
	outPointStr := "b721c368f9ddb1d6d0d225fdb22e8f2b4b9f0fed160ea6ae80270b7849c2d62e:0"
	outPoint, err := wallet.NewOutPointFromString(outPointStr)
	if err != nil {
		t.Fatal(err)
	}
	scriptPubKey, err := hex.DecodeString("001427ba894b4ac84cf236608590efe12ee514692c32")
	if err != nil {
		t.Fatal(err)
	}
	utxo := wallet.Utxo{
		Op:           *outPoint,
		AtHeight:     0,
		Value:        300000000,
		ScriptPubkey: scriptPubKey,
		WatchOnly:    false,
		Frozen:       false,
	}
	err = uxdb.Put(utxo)
	if err != nil {
		t.Error(err)
	}

	err = uxdb.SetWatchOnly(utxo)
	if err != nil {
		t.Error(err)
	}

	u, err := uxdb.Get(outPoint)
	if err != nil {
		t.Error(err)
	}
	if !u.WatchOnly {
		t.Error("failed to set watch only")
	}
}

func TestFreezeUnFreezeUtxo(t *testing.T) {
	if err := setupUxdb(); err != nil {
		t.Fatal(err)
	}
	defer teardownUxdb()
	outPointStr := "b721c368f9ddb1d6d0d225fdb22e8f2b4b9f0fed160ea6ae80270b7849c2d62e:0"
	outPoint, err := wallet.NewOutPointFromString(outPointStr)
	if err != nil {
		t.Fatal(err)
	}
	scriptPubKey, err := hex.DecodeString("001427ba894b4ac84cf236608590efe12ee514692c32")
	if err != nil {
		t.Fatal(err)
	}
	utxo := wallet.Utxo{
		Op:           *outPoint,
		AtHeight:     0,
		Value:        300000000,
		ScriptPubkey: scriptPubKey,
		WatchOnly:    false,
		Frozen:       false,
	}
	err = uxdb.Put(utxo)
	if err != nil {
		t.Error(err)
	}

	err = uxdb.Freeze(utxo)
	if err != nil {
		t.Error(err)
	}

	u, err := uxdb.Get(outPoint)
	if err != nil {
		t.Error(err)
	}
	if !u.Frozen {
		t.Error("failed to freeze utxo")
	}

	err = uxdb.UnFreeze(utxo)
	if err != nil {
		t.Error(err)
	}

	u, err = uxdb.Get(outPoint)
	if err != nil {
		t.Error(err)
	}
	if u.Frozen {
		t.Error("failed to un-freeze utxo")
	}
}

func TestDeleteUtxo(t *testing.T) {
	if err := setupUxdb(); err != nil {
		t.Fatal(err)
	}
	defer teardownUxdb()
	outPointStr := "b721c368f9ddb1d6d0d225fdb22e8f2b4b9f0fed160ea6ae80270b7849c2d62e:0"
	outPoint, err := wallet.NewOutPointFromString(outPointStr)
	if err != nil {
		t.Fatal(err)
	}
	scriptPubKey, err := hex.DecodeString("001427ba894b4ac84cf236608590efe12ee514692c32")
	if err != nil {
		t.Fatal(err)
	}
	utxo := wallet.Utxo{
		Op:           *outPoint,
		AtHeight:     0,
		Value:        300000000,
		ScriptPubkey: scriptPubKey,
		WatchOnly:    false,
		Frozen:       false,
	}
	err = uxdb.Put(utxo)
	if err != nil {
		t.Error(err)
	}

	err = uxdb.Delete(utxo)
	if err != nil {
		t.Error(err)
	}
	utxos, err := uxdb.GetAll()
	if err != nil {
		t.Error(err)
	}
	if len(utxos) != 0 {
		t.Error("Utxo db delete failed")
	}
}
