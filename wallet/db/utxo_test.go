package db

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"strconv"
	"sync"
	"testing"

	"github.com/bisoncraft/go-electrum-client/wallet"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

var uxdb UtxoDB
var utxo wallet.Utxo

func init() {
	conn, _ := sql.Open("sqlite3", ":memory:")
	initDatabaseTables(conn)
	uxdb = UtxoDB{
		db:   conn,
		lock: new(sync.RWMutex),
	}
	sh1, _ := chainhash.NewHashFromStr("e941e1c32b3dd1a68edc3af9f7fe711f35aaca60f758c2dd49561e45ca2c41c0")
	outpoint := wire.NewOutPoint(sh1, 0)
	utxo = wallet.Utxo{
		Op:           *outpoint,
		AtHeight:     300000,
		Value:        100000000,
		ScriptPubkey: []byte("scriptpubkey"),
		WatchOnly:    false,
		Frozen:       true,
	}
}

func TestUtxoPut(t *testing.T) {
	err := uxdb.Put(utxo)
	if err != nil {
		t.Error(err)
	}
	stmt, _ := uxdb.db.Prepare("select outpoint, value, height, scriptPubKey, frozen from utxos where outpoint=?")
	defer stmt.Close()

	var outpoint string
	var value int
	var height int
	var scriptPubkey string
	var frozenInt int
	o := utxo.Op.Hash.String() + ":" + strconv.Itoa(int(utxo.Op.Index))
	err = stmt.QueryRow(o).Scan(&outpoint, &value, &height, &scriptPubkey, &frozenInt)
	if err != nil {
		t.Error(err)
	}
	bFrozen := frozenInt == 1
	if outpoint != o {
		t.Error("Utxo db returned wrong outpoint")
	}
	if value != int(utxo.Value) {
		t.Error("Utxo db returned wrong value")
	}
	if height != int(utxo.AtHeight) {
		t.Error("Utxo db returned wrong height")
	}
	if scriptPubkey != hex.EncodeToString(utxo.ScriptPubkey) {
		t.Error("Utxo db returned wrong scriptPubKey")
	}
	if bFrozen != utxo.Frozen {
		t.Error("Utxo db returned wrong scriptPubKey")

	}
}

func TestUtxoGetAll(t *testing.T) {
	err := uxdb.Put(utxo)
	if err != nil {
		t.Error(err)
	}
	utxos, err := uxdb.GetAll()
	if err != nil {
		t.Error(err)
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
	if utxos[0].WatchOnly != utxo.WatchOnly {
		t.Error("Utxo db returned wrong watchOnly value")
	}
	if utxos[0].Frozen != utxo.Frozen {
		t.Error("Utxo db returned wrong frozen value")
	}
}

func TestSetWatchOnlyUtxo(t *testing.T) {
	err := uxdb.Put(utxo)
	if err != nil {
		t.Error(err)
	}
	err = uxdb.SetWatchOnly(utxo)
	if err != nil {
		t.Error(err)
	}
	stmt, _ := uxdb.db.Prepare("select watchOnly from utxos where outpoint=?")
	defer stmt.Close()

	var watchOnlyInt int
	o := utxo.Op.Hash.String() + ":" + strconv.Itoa(int(utxo.Op.Index))
	err = stmt.QueryRow(o).Scan(&watchOnlyInt)
	if err != nil {
		t.Error(err)
	}
	if watchOnlyInt != 1 {
		t.Error("Utxo freeze failed")
	}

}

func TestFreezeUnFreezeUtxo(t *testing.T) {
	err := uxdb.Put(utxo)
	if err != nil {
		t.Error(err)
	}
	err = uxdb.Freeze(utxo)
	if err != nil {
		t.Error(err)
	}
	stmt, _ := uxdb.db.Prepare("select frozen from utxos where outpoint=?")
	defer stmt.Close()

	var frozenInt int
	o := utxo.Op.Hash.String() + ":" + strconv.Itoa(int(utxo.Op.Index))
	err = stmt.QueryRow(o).Scan(&frozenInt)
	if err != nil {
		t.Error(err)
	}
	if frozenInt != 1 {
		t.Error("Utxo freeze failed")
	}
	err = uxdb.UnFreeze(utxo)
	if err != nil {
		t.Error(err)
	}
	stmt1, _ := uxdb.db.Prepare("select frozen from utxos where outpoint=?")
	defer stmt1.Close()

	err = stmt.QueryRow(o).Scan(&frozenInt)
	if err != nil {
		t.Error(err)
	}
	if frozenInt != 0 {
		t.Error("Utxo freeze failed")
	}
}

func TestDeleteUtxo(t *testing.T) {
	err := uxdb.Put(utxo)
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
