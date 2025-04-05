package wltfiro

import (
	"encoding/hex"
	"errors"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/bisoncraft/go-electrum-client/wallet"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/tyler-smith/go-bip39"
)

var test_mnemonic = "jungle pair grass super coral bubble tomato sheriff pulp cancel luggage wagon"

func makeRegtestSeed() []byte {
	return bip39.NewSeed(test_mnemonic, "")
}

func createTxStore() (*TxStore, *StorageManager) {
	mockDb := MockDatastore{
		&mockConfig{creationDate: time.Now()},
		&mockStorage{blob: make([]byte, 10)},
		&mockKeyStore{make(map[string]*keyStoreEntry)},
		&mockUtxoStore{make(map[string]*wallet.Utxo)},
		&mockStxoStore{make(map[string]*wallet.Stxo)},
		&mockTxnStore{make(map[string]*wallet.Txn)},
		&mockSubscriptionsStore{make(map[string]*wallet.Subscription)},
	}

	seed := makeRegtestSeed()
	key, _ := hdkeychain.NewMaster(seed, &chaincfg.RegressionNetParams)
	km, _ := NewKeyManager(mockDb.Keys(), &chaincfg.RegressionNetParams, key)
	sm := NewStorageManager(mockDb.Enc(), &chaincfg.RegressionNetParams)
	txStore, _ := NewTxStore(&chaincfg.RegressionNetParams, &mockDb, km)
	return txStore, sm
}

// A 'regtest' wallet
func MockWallet(pw string) *FiroElectrumWallet {
	txstore, storageMgr := createTxStore()

	storageMgr.store.Xprv = "tprv8ZgxMBicQKsPfJU6JyiVdmFAtAzmWmTeEv85nTAHjLQyL35tdP2fAPWDSBBnFqGhhfTHVQMcnZhZDFkzFmCjm1bgf5UDwMAeFUWhJ9Dr8c4"
	storageMgr.store.Xpub = "tpubD6NzVbkrYhZ4YmVtCdP63AuHTCWhg6eYpDis4yCb9cDNAXLfFmrFLt85cLFTwHiDJ9855NiE7cgQdiTGt5mb2RS9RfaxgVDkwBybJWm54Gh"
	storageMgr.store.ShaPw = chainhash.HashB([]byte(pw))
	storageMgr.store.Seed = []byte{0x01, 0x02, 0x03}

	wallet := &FiroElectrumWallet{
		txstore:        txstore,
		keyManager:     txstore.keyManager,
		storageManager: storageMgr,
		params:         &chaincfg.RegressionNetParams,
		feeProvider:    wallet.DefaultFeeProvider(),
	}

	// fundWallet(wallet)

	return wallet
}

type MockDatastore struct {
	cfg              wallet.Cfg
	enc              wallet.Enc
	keys             wallet.Keys
	utxos            wallet.Utxos
	stxos            wallet.Stxos
	txns             wallet.Txns
	subscribeScripts wallet.Subscriptions
}

func (m *MockDatastore) Cfg() wallet.Cfg {
	return m.cfg
}

func (m *MockDatastore) Enc() wallet.Enc {
	return m.enc
}

func (m *MockDatastore) Keys() wallet.Keys {
	return m.keys
}

func (m *MockDatastore) Utxos() wallet.Utxos {
	return m.utxos
}

func (m *MockDatastore) Stxos() wallet.Stxos {
	return m.stxos
}

func (m *MockDatastore) Txns() wallet.Txns {
	return m.txns
}

func (m *MockDatastore) Subscriptions() wallet.Subscriptions {
	return m.subscribeScripts
}

type mockConfig struct {
	creationDate time.Time
}

func (mc *mockConfig) PutCreationDate(date time.Time) error {
	mc.creationDate = date
	return nil
}

func (mc *mockConfig) GetCreationDate() (time.Time, error) {
	return mc.creationDate, nil
}

// encrypted blob
type mockStorage struct {
	blob []byte
}

// reverse simulates encryption/decryption between bytes and a database blob
func reverse(s []byte) []byte {
	var newB = make([]byte, len(s))
	copy(newB, s)
	left := 0
	right := len(s) - 1
	for left < right {
		newB[left], newB[right] = newB[right], newB[left]
		left++
		right--
	}
	return newB
}

func (ms *mockStorage) PutEncrypted(b []byte, pw string) error {
	if pw != "abc" {
		return errors.New("invalid password")
	}
	ms.blob = reverse(b)
	return nil
}

func (ms *mockStorage) GetDecrypted(pw string) ([]byte, error) {
	if pw != "abc" {
		return nil, errors.New("invalid password")
	}
	return reverse(ms.blob), nil
}

type keyStoreEntry struct {
	scriptAddress []byte
	path          wallet.KeyPath
	used          bool
	key           *btcec.PrivateKey
}

type mockKeyStore struct {
	keys map[string]*keyStoreEntry
}

func (m *mockKeyStore) Put(scriptAddress []byte, keyPath wallet.KeyPath) error {
	m.keys[hex.EncodeToString(scriptAddress)] = &keyStoreEntry{scriptAddress, keyPath, false, nil}
	return nil
}

func (m *mockKeyStore) ImportKey(scriptAddress []byte, key *btcec.PrivateKey) error {
	kp := wallet.KeyPath{Purpose: wallet.EXTERNAL, Index: -1}
	m.keys[hex.EncodeToString(scriptAddress)] = &keyStoreEntry{scriptAddress, kp, false, key}
	return nil
}

func (m *mockKeyStore) MarkKeyAsUsed(scriptAddress []byte) error {
	key, ok := m.keys[hex.EncodeToString(scriptAddress)]
	if !ok {
		return errors.New("key does not exist")
	}
	key.used = true
	return nil
}

func (m *mockKeyStore) GetLastKeyIndex(purpose wallet.KeyPurpose) (int, bool, error) {
	i := -1
	used := false
	for _, key := range m.keys {
		if key.path.Purpose == purpose && key.path.Index > i {
			i = key.path.Index
			used = key.used
		}
	}
	if i == -1 {
		return i, used, errors.New("no saved keys")
	}
	return i, used, nil
}

func (m *mockKeyStore) GetPathForKey(scriptAddress []byte) (wallet.KeyPath, error) {
	key, ok := m.keys[hex.EncodeToString(scriptAddress)]
	if !ok || key.path.Index == -1 {
		return wallet.KeyPath{}, errors.New("key does not exist")
	}
	return key.path, nil
}

func (m *mockKeyStore) GetUnused(purpose wallet.KeyPurpose) ([]int, error) {
	var i []int
	for _, key := range m.keys {
		if !key.used && key.path.Purpose == purpose {
			i = append(i, key.path.Index)
		}
	}
	sort.Ints(i)
	return i, nil
}

func (m *mockKeyStore) GetAll() ([]wallet.KeyPath, error) {
	var kp []wallet.KeyPath
	for _, key := range m.keys {
		kp = append(kp, key.path)
	}
	return kp, nil
}

func (m *mockKeyStore) GetDbg() string {
	var ret string = "TODO{}"
	//
	return ret
}

func (m *mockKeyStore) GetLookaheadWindows() map[wallet.KeyPurpose]int {
	internalLastUsed := -1
	externalLastUsed := -1
	for _, key := range m.keys {
		if key.path.Purpose == wallet.INTERNAL && key.used && key.path.Index > internalLastUsed {
			internalLastUsed = key.path.Index
		}
		if key.path.Purpose == wallet.EXTERNAL && key.used && key.path.Index > externalLastUsed {
			externalLastUsed = key.path.Index
		}
	}
	internalUnused := 0
	externalUnused := 0
	for _, key := range m.keys {
		if key.path.Purpose == wallet.INTERNAL && !key.used && key.path.Index > internalLastUsed {
			internalUnused++
		}
		if key.path.Purpose == wallet.EXTERNAL && !key.used && key.path.Index > externalLastUsed {
			externalUnused++
		}
	}
	mp := make(map[wallet.KeyPurpose]int)
	mp[wallet.INTERNAL] = internalUnused
	mp[wallet.EXTERNAL] = externalUnused
	return mp
}

type mockUtxoStore struct {
	utxos map[string]*wallet.Utxo
}

func (m *mockUtxoStore) Put(utxo wallet.Utxo) error {
	key := utxo.Op.Hash.String() + ":" + strconv.Itoa(int(utxo.Op.Index))
	m.utxos[key] = &utxo
	return nil
}

func (m *mockUtxoStore) GetAll() ([]wallet.Utxo, error) {
	var utxos []wallet.Utxo
	for _, v := range m.utxos {
		utxos = append(utxos, *v)
	}
	return utxos, nil
}

func (m *mockUtxoStore) SetWatchOnly(utxo wallet.Utxo) error {
	key := utxo.Op.Hash.String() + ":" + strconv.Itoa(int(utxo.Op.Index))
	u, ok := m.utxos[key]
	if !ok {
		return errors.New("not found")
	}
	u.WatchOnly = true
	return nil
}

func (m *mockUtxoStore) Freeze(utxo wallet.Utxo) error {
	key := utxo.Op.Hash.String() + ":" + strconv.Itoa(int(utxo.Op.Index))
	u, ok := m.utxos[key]
	if !ok {
		return errors.New("not found")
	}
	u.Frozen = true
	return nil
}

func (m *mockUtxoStore) UnFreeze(utxo wallet.Utxo) error {
	key := utxo.Op.Hash.String() + ":" + strconv.Itoa(int(utxo.Op.Index))
	u, ok := m.utxos[key]
	if !ok {
		return errors.New("not found")
	}
	u.WatchOnly = false
	return nil
}

func (m *mockUtxoStore) Delete(utxo wallet.Utxo) error {
	key := utxo.Op.Hash.String() + ":" + strconv.Itoa(int(utxo.Op.Index))
	_, ok := m.utxos[key]
	if !ok {
		return errors.New("not found")
	}
	delete(m.utxos, key)
	return nil
}

type mockStxoStore struct {
	stxos map[string]*wallet.Stxo
}

func (m *mockStxoStore) Put(stxo wallet.Stxo) error {
	m.stxos[stxo.SpendTxid.String()] = &stxo
	return nil
}

func (m *mockStxoStore) GetAll() ([]wallet.Stxo, error) {
	var stxos []wallet.Stxo
	for _, v := range m.stxos {
		stxos = append(stxos, *v)
	}
	return stxos, nil
}

func (m *mockStxoStore) Delete(stxo wallet.Stxo) error {
	_, ok := m.stxos[stxo.SpendTxid.String()]
	if !ok {
		return errors.New("not found")
	}
	delete(m.stxos, stxo.SpendTxid.String())
	return nil
}

type mockTxnStore struct {
	txns map[string]*wallet.Txn
}

func (m *mockTxnStore) Put(raw []byte, txid string, value int64, height int64, timestamp time.Time, watchOnly bool) error {
	m.txns[txid] = &wallet.Txn{
		Txid:      txid,
		Value:     value,
		Height:    height,
		Timestamp: timestamp,
		WatchOnly: watchOnly,
		Bytes:     raw,
	}
	return nil
}

func (m *mockTxnStore) Get(txid string) (wallet.Txn, error) {
	t, ok := m.txns[txid]
	if !ok {
		return wallet.Txn{}, errors.New("not found")
	}
	return *t, nil
}

func (m *mockTxnStore) GetAll(includeWatchOnly bool) ([]wallet.Txn, error) {
	var txns []wallet.Txn
	for _, t := range m.txns {
		if !includeWatchOnly && t.WatchOnly {
			continue
		}
		txns = append(txns, *t)
	}
	return txns, nil
}

func (m *mockTxnStore) UpdateHeight(txid string, height int, timestamp time.Time) error {
	txn, ok := m.txns[txid]
	if !ok {
		return errors.New("not found")
	}
	txn.Height = int64(height)
	txn.Timestamp = timestamp
	return nil
}

func (m *mockTxnStore) Delete(txid string) error {
	_, ok := m.txns[txid]
	if !ok {
		return errors.New("not found")
	}
	delete(m.txns, txid)
	return nil
}

type mockSubscriptionsStore struct {
	subcriptions map[string]*wallet.Subscription
}

func (m *mockSubscriptionsStore) Put(sub *wallet.Subscription) error {
	if sub == nil {
		return errors.New("nil subscription")
	}
	m.subcriptions[sub.PkScript] = sub
	return nil
}

func (m *mockSubscriptionsStore) Get(scriptPubKey string) (*wallet.Subscription, error) {
	_, ok := m.subcriptions[scriptPubKey]
	if !ok {
		return nil, errors.New("not found")
	}
	return m.subcriptions[scriptPubKey], nil
}

func (m *mockSubscriptionsStore) GetElectrumScripthash(electrumScripthash string) (*wallet.Subscription, error) {
	for _, sub := range m.subcriptions {
		if sub.ElectrumScripthash == electrumScripthash {
			return sub, nil
		}
	}
	return nil, nil
}

func (m *mockSubscriptionsStore) GetAll() ([]*wallet.Subscription, error) {
	var ret []*wallet.Subscription
	for _, sub := range m.subcriptions {
		ret = append(ret, sub)
	}
	return ret, nil
}

func (m *mockSubscriptionsStore) Delete(scriptPubKey string) error {
	_, ok := m.subcriptions[scriptPubKey]
	if !ok {
		return errors.New("not found")
	}
	delete(m.subcriptions, scriptPubKey)
	return nil
}

func TestUtxo_IsEqual(t *testing.T) {
	h, err := chainhash.NewHashFromStr("16bed6368b8b1542cd6eb87f5bc20dc830b41a2258dde40438a75fa701d24e9a")
	if err != nil {
		t.Error(err)
	}
	u := &wallet.Utxo{
		Op:           *wire.NewOutPoint(h, 0),
		ScriptPubkey: make([]byte, 32),
		AtHeight:     400000,
		Value:        1000000,
	}
	if !u.IsEqual(u) {
		t.Error("Failed to return utxos as equal")
	}
	testUtxo := *u
	testUtxo.Op.Index = 3
	if u.IsEqual(&testUtxo) {
		t.Error("Failed to return utxos as not equal")
	}
	testUtxo = *u
	testUtxo.AtHeight = 1
	if u.IsEqual(&testUtxo) {
		t.Error("Failed to return utxos as not equal")
	}
	testUtxo = *u
	testUtxo.Value = 4
	if u.IsEqual(&testUtxo) {
		t.Error("Failed to return utxos as not equal")
	}
	testUtxo = *u
	ch2, err := chainhash.NewHashFromStr("1f64249abbf2fcc83fc060a64f69a91391e9f5d98c5d3135fe9716838283aa4c")
	if err != nil {
		t.Error(err)
	}
	testUtxo.Op.Hash = *ch2
	if u.IsEqual(&testUtxo) {
		t.Error("Failed to return utxos as not equal")
	}
	testUtxo = *u
	testUtxo.ScriptPubkey = make([]byte, 4)
	if u.IsEqual(&testUtxo) {
		t.Error("Failed to return utxos as not equal")
	}
	if u.IsEqual(nil) {
		t.Error("Failed to return utxos as not equal")
	}
}

func TestStxo_IsEqual(t *testing.T) {
	h, err := chainhash.NewHashFromStr("16bed6368b8b1542cd6eb87f5bc20dc830b41a2258dde40438a75fa701d24e9a")
	if err != nil {
		t.Error(err)
	}
	u := &wallet.Utxo{
		Op:           *wire.NewOutPoint(h, 0),
		ScriptPubkey: make([]byte, 32),
		AtHeight:     400000,
		Value:        1000000,
	}
	h2, _ := chainhash.NewHashFromStr("1f64249abbf2fcc83fc060a64f69a91391e9f5d98c5d3135fe9716838283aa4c")
	s := &wallet.Stxo{
		Utxo:        *u,
		SpendHeight: 400001,
		SpendTxid:   *h2,
	}
	if !s.IsEqual(s) {
		t.Error("Failed to return stxos as equal")
	}

	testStxo := *s
	testStxo.SpendHeight = 5
	if s.IsEqual(&testStxo) {
		t.Error("Failed to return stxos as not equal")
	}
	h3, _ := chainhash.NewHashFromStr("3c5cea030a432ba9c8cf138a93f7b2e5b28263ea416894ee0bdf91bc31bb04f2")
	testStxo = *s
	testStxo.SpendTxid = *h3
	if s.IsEqual(&testStxo) {
		t.Error("Failed to return stxos as not equal")
	}
	if s.IsEqual(nil) {
		t.Error("Failed to return stxos as not equal")
	}
	testStxo = *s
	testStxo.Utxo.AtHeight = 7
	if s.IsEqual(&testStxo) {
		t.Error("Failed to return stxos as not equal")
	}
}
