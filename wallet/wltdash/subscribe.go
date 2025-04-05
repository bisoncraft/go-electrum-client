package wltdash

import (
	"github.com/bisoncraft/go-electrum-client/wallet"
	"github.com/btcsuite/btcd/chaincfg"
)

type SubscriptionManager struct {
	datastore wallet.Subscriptions
	params    *chaincfg.Params
}

func NewSubscriptionManager(db wallet.Subscriptions, params *chaincfg.Params) *SubscriptionManager {
	sm := &SubscriptionManager{
		datastore: db,
		params:    params,
	}
	return sm
}

func (sm *SubscriptionManager) Put(subscription *wallet.Subscription) error {
	return sm.datastore.Put(subscription)
}

func (sm *SubscriptionManager) Get(scriptPubKey string) (*wallet.Subscription, error) {
	return sm.datastore.Get(scriptPubKey)
}

func (sm *SubscriptionManager) GetElectrumScripthash(electrumScripthash string) (*wallet.Subscription, error) {
	return sm.datastore.GetElectrumScripthash(electrumScripthash)
}

func (sm *SubscriptionManager) GetAll() ([]*wallet.Subscription, error) {
	return sm.datastore.GetAll()
}

func (sm *SubscriptionManager) Delete(scriptPubKey string) error {
	return sm.datastore.Delete(scriptPubKey)
}
