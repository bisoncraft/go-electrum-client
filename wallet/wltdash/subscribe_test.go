package wltdash

import (
	"testing"

	"github.com/bisoncraft/go-electrum-client/wallet"
	"github.com/btcsuite/btcd/chaincfg"
)

func createSubscriptionManager() *SubscriptionManager {
	subscriptions := make(map[string]*wallet.Subscription)
	return NewSubscriptionManager(&mockSubscriptionsStore{subcriptions: subscriptions}, &chaincfg.MainNetParams)
}

func TestStoreSubscription(t *testing.T) {
	sm := createSubscriptionManager()
	sub := &wallet.Subscription{
		PkScript:           "paymentscript",
		ElectrumScripthash: "electrumScripthash",
		Address:            "address",
	}
	err := sm.datastore.Put(sub)
	if err != nil {
		t.Fatal(err)
	}

	subs, err := sm.datastore.GetAll()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range subs {
		if s.IsEqual(sub) {
			return
		}
	}
	t.Fatal("stored subscription is not returned")
}

func TestDeleteSubscription(t *testing.T) {
	sm := createSubscriptionManager()
	sub := &wallet.Subscription{
		PkScript:           "paymentscript",
		ElectrumScripthash: "electrumScripthash",
		Address:            "address",
	}
	err := sm.datastore.Put(sub)
	if err != nil {
		t.Fatal(err)
	}

	err = sm.datastore.Delete("paymentscript")
	if err != nil {
		t.Fatal(err)
	}

	subs, err := sm.datastore.GetAll()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range subs {
		if s.IsEqual(sub) {
			t.Fatal("deleted subscription returned")
		}
	}
}
