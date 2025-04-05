package bdb

import (
	"os"
	"sync"
	"testing"

	"github.com/bisoncraft/go-electrum-client/wallet"
	bolt "go.etcd.io/bbolt"
)

var ssdb *SubscriptionsDB

func setupSsdb() error {
	bdb, err := bolt.Open("test.bdb", 0600, nil)
	if err != nil {
		return nil
	}
	err = initDatabaseBuckets(bdb)
	if err != nil {
		return nil
	}
	ssdb = &SubscriptionsDB{
		db:   bdb,
		lock: new(sync.RWMutex),
	}
	return nil
}

func teardownSsdb() {
	if ssdb == nil {
		return
	}
	ssdb.db.Close()
	os.RemoveAll("test.bdb")
}

func TestSubscriptionsDB_Put(t *testing.T) {
	if err := setupSsdb(); err != nil {
		t.Fatal(err)
	}
	defer teardownSsdb()
	sub := &wallet.Subscription{
		PkScript:           "spk",
		ElectrumScripthash: "esh",
		Address:            "add",
	}
	err := ssdb.Put(sub)
	if err != nil {
		t.Error(err)
	}

	subOut, err := ssdb.Get("spk")
	if err != nil {
		t.Error(err)
	}

	if !sub.IsEqual(subOut) {
		t.Error("Failed to insert subscription into db")
	}
}

func TestSubscriptionDB_Get(t *testing.T) {
	if err := setupSsdb(); err != nil {
		t.Fatal(err)
	}
	defer teardownSsdb()
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
		t.Error(err)
	}
	if sub != nil {
		if sub.ElectrumScripthash != "esh" {
			t.Error("invalid return")
		}
		if sub.Address != "add" {
			t.Error("invalid return")
		}
	}

	sub, err = ssdb.Get("spk2")
	if err != nil {
		t.Error(err)
	}
	if sub != nil {
		if sub.ElectrumScripthash != "esh2" {
			t.Error("invalid return")
		}
		if sub.Address != "add2" {
			t.Error("invalid return")
		}
	}
}

func TestSubscriptionDB_GetElectrumScripthash(t *testing.T) {
	if err := setupSsdb(); err != nil {
		t.Fatal(err)
	}
	defer teardownSsdb()
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
		t.Error(err)
	}
	if sub != nil {
		if sub.PkScript != "spk" {
			t.Error("invalid return")
		}
		if sub.Address != "add" {
			t.Error("invalid return")
		}
	}

	sub, err = ssdb.GetElectrumScripthash("esh2")
	if err != nil {
		t.Error(err)
	}
	if sub != nil {
		if sub.PkScript != "spk2" {
			t.Error("invalid return 2")
		}
		if sub.Address != "add2" {
			t.Error("invalid return 2")
		}
	}
}

func TestSubscriptionDB_GetAll(t *testing.T) {
	if err := setupSsdb(); err != nil {
		t.Fatal(err)
	}
	defer teardownSsdb()
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
		t.Fatal("incorrect number of subscribe scripts")
	}
	if !subs[0].IsEqual(sub) {
		t.Error("incorrect subscribe script")
	}
	if !subs[1].IsEqual(sub2) {
		t.Error("incorrect subscribe script")
	}
}

func TestSubscriptionsDB_Delete(t *testing.T) {
	if err := setupSsdb(); err != nil {
		t.Fatal(err)
	}
	defer teardownSsdb()
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
