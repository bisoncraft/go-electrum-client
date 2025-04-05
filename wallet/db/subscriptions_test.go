package db

import (
	"database/sql"
	"sync"
	"testing"

	"github.com/bisoncraft/go-electrum-client/wallet"
)

var ssdb SubscriptionsDB

func init() {
	conn, _ := sql.Open("sqlite3", ":memory:")
	initDatabaseTables(conn)
	ssdb = SubscriptionsDB{
		db:   conn,
		lock: new(sync.RWMutex),
	}
}

func TestSubscriptionsDB_Put(t *testing.T) {
	sub := &wallet.Subscription{
		PkScript:           "spk",
		ElectrumScripthash: "esh",
		Address:            "add",
	}
	err := ssdb.Put(sub)
	if err != nil {
		t.Error(err)
	}
	stmt, _ := ssdb.db.Prepare("select scriptPubKey, electrumScripthash, address from subscriptions")
	defer stmt.Close()
	var scriptPubKey string
	var electrumScripthash string
	var address string
	err = stmt.QueryRow().Scan(&scriptPubKey, &electrumScripthash, &address)
	if err != nil {
		t.Error(err)
	}
	subOut := &wallet.Subscription{
		PkScript:           scriptPubKey,
		ElectrumScripthash: electrumScripthash,
		Address:            address,
	}
	if !sub.IsEqual(subOut) {
		t.Error("Failed to insert subscription into db")
	}
}

func TestSubscriptionDB_Get(t *testing.T) {
	sub := &wallet.Subscription{
		PkScript:           "spk",
		ElectrumScripthash: "esh",
		Address:            "add",
	}
	err := ssdb.Put(sub)
	if err != nil {
		t.Error(err)
	}
	sub2 := &wallet.Subscription{
		PkScript:           "spk2",
		ElectrumScripthash: "esh2",
		Address:            "add2",
	}
	err = ssdb.Put(sub2)
	if err != nil {
		t.Error(err)
	}
	sub, err = ssdb.Get("spk")
	if err != nil {
		if err != sql.ErrNoRows {
			t.Error(err)
		}
	}
	if sub != nil {
		if sub.ElectrumScripthash != "esh" {
			t.Error("invalid return")
		}
	}
	sub, err = ssdb.Get("spk2")
	if err != nil {
		if err != sql.ErrNoRows {
			t.Error(err)
		}
	}
	if sub != nil {
		if sub.ElectrumScripthash != "esh2" {
			t.Error("invalid return")
		}
	}
}

func TestSubscriptionDB_GetElectrumScripthash(t *testing.T) {
	sub := &wallet.Subscription{
		PkScript:           "spk",
		ElectrumScripthash: "esh",
		Address:            "add",
	}
	err := ssdb.Put(sub)
	if err != nil {
		t.Error(err)
	}
	sub2 := &wallet.Subscription{
		PkScript:           "spk2",
		ElectrumScripthash: "esh2",
		Address:            "add2",
	}
	err = ssdb.Put(sub2)
	if err != nil {
		t.Error(err)
	}
	sub, err = ssdb.GetElectrumScripthash("esh")
	if err != nil {
		if err != sql.ErrNoRows {
			t.Error(err)
		}
	}
	if sub != nil {
		if sub.PkScript != "spk" {
			t.Error("invalid return")
		}
	}
	sub, err = ssdb.GetElectrumScripthash("esh2")
	if err != nil {
		if err != sql.ErrNoRows {
			t.Error(err)
		}
	}
	if sub != nil {
		if sub.PkScript != "spk2" {
			t.Error("invalid return 2")
		}
	}
}

func TestSubscriptionDB_GetAll(t *testing.T) {
	sub := &wallet.Subscription{
		PkScript:           "spk",
		ElectrumScripthash: "esh",
		Address:            "add",
	}
	err := ssdb.Put(sub)
	if err != nil {
		t.Error(err)
	}
	sub2 := &wallet.Subscription{
		PkScript:           "spk2",
		ElectrumScripthash: "esh2",
		Address:            "add2",
	}
	err = ssdb.Put(sub2)
	if err != nil {
		t.Error(err)
	}
	subs, err := ssdb.GetAll()
	if err != nil {
		t.Error(err)
	}
	if len(subs) != 2 {
		t.Error("incorrect number of subscribe scripts")
	}
	if !subs[0].IsEqual(sub) {
		t.Error("incorrect subscribe script")
	}
	if !subs[1].IsEqual(sub2) {
		t.Error("incorrect subscribe script")
	}
}

func TestSubscriptionsDB_Delete(t *testing.T) {
	sub := &wallet.Subscription{
		PkScript:           "spk",
		ElectrumScripthash: "esh",
		Address:            "add",
	}
	err := ssdb.Put(sub)
	if err != nil {
		t.Error(err)
	}
	sub2 := &wallet.Subscription{
		PkScript:           "spk2",
		ElectrumScripthash: "esh2",
		Address:            "add2",
	}
	err = ssdb.Put(sub2)
	if err != nil {
		t.Error(err)
	}

	err = ssdb.Delete("spk")
	if err != nil {
		t.Error(err)
	}
	subs, err := ssdb.GetAll()
	if err != nil {
		t.Error(err)
	}
	if len(subs) != 1 {
		t.Error("wrong length after delete spk")
	}
	err = ssdb.Delete("spk2")
	if err != nil {
		t.Error(err)
	}
	subs, err = ssdb.GetAll()
	if err != nil {
		t.Error(err)
	}
	if len(subs) != 0 {
		t.Error("wrong length after delete spk2")
	}
}
