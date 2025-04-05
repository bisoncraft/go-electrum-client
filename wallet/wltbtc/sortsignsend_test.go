package wltbtc

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/bisoncraft/go-electrum-client/wallet"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

// var test_mnemonic = "jungle pair grass super coral bubble tomato sheriff pulp cancel luggage wagon"

// func makeRegtestSeed() []byte {
// 	return bip39.NewSeed(test_mnemonic, "")
// }

// func createTxStore() (*TxStore, *StorageManager) {
// 	mockDb := MockDatastore{
// 		&mockConfig{creationDate: time.Now()},
// 		&mockStorage{blob: make([]byte, 10)},
// 		&mockKeyStore{make(map[string]*keyStoreEntry)},
// 		&mockUtxoStore{make(map[string]*wallet.Utxo)},
// 		&mockStxoStore{make(map[string]*wallet.Stxo)},
// 		&mockTxnStore{make(map[string]*wallet.Txn)},
// 		&mockSubscriptionsStore{make(map[string]*wallet.Subscription)},
// 	}

// 	seed = makeRegtestSeed()
// 	// fmt.Println("Made test seed")
// 	key, _ := hdkeychain.NewMaster(seed, &chaincfg.RegressionNetParams)
// 	km, _ := NewKeyManager(mockDb.Keys(), &chaincfg.RegressionNetParams, key)
// 	sm := NewStorageManager(mockDb.Enc(), &chaincfg.RegressionNetParams)
// 	txStore, _ := NewTxStore(&chaincfg.RegressionNetParams, &mockDb, km)
// 	return txStore, sm
// }

// // A sample regtest wallet
// func MockWallet() *BtcElectrumWallet {
// 	txstore, storageMgr := createTxStore()

// 	storageMgr.store.Xprv = "tprv8ZgxMBicQKsPfJU6JyiVdmFAtAzmWmTeEv85nTAHjLQyL35tdP2fAPWDSBBnFqGhhfTHVQMcnZhZDFkzFmCjm1bgf5UDwMAeFUWhJ9Dr8c4"
// 	storageMgr.store.Xpub = "tpubD6NzVbkrYhZ4YmVtCdP63AuHTCWhg6eYpDis4yCb9cDNAXLfFmrFLt85cLFTwHiDJ9855NiE7cgQdiTGt5mb2RS9RfaxgVDkwBybJWm54Gh"
// 	storageMgr.store.ShaPw = chainhash.HashB([]byte(pw))
// 	storageMgr.store.Seed = []byte{0x01, 0x02, 0x03}

// 	return &BtcElectrumWallet{
// 		txstore:        txstore,
// 		keyManager:     txstore.keyManager,
// 		storageManager: storageMgr,
// 		params:         &chaincfg.RegressionNetParams,
// 		feeProvider:    wallet.DefaultFeeProvider(),
// 	}
// }

// test gather coins
func Test_gatherCoins(t *testing.T) {
	w := MockWallet("abc")
	w.blockchainTip = 500
	// utxo 1
	txid1 := "edfab2f9b2a013a36c524bf63e9778a5d13ca8bf1fce279647fbcee30bf7dd62"
	h1, err := chainhash.NewHashFromStr(txid1)
	if err != nil {
		t.Error(err)
	}
	script1, err := hex.DecodeString("0014b8da433782cd9d142f32f726926bcc8161beeaef")
	if err != nil {
		t.Error(err)
	}
	op1 := wire.OutPoint{
		Hash:  *h1,
		Index: 1,
	}
	utxo1 := wallet.Utxo{Op: op1, ScriptPubkey: script1, AtHeight: 426, Value: 17556690040, WatchOnly: false, Frozen: false}
	err = w.txstore.Utxos().Put(utxo1)
	if err != nil {
		t.Error(err)
	}
	// utxo 2
	txid2 := "5ebbbeafb0d23805c09c87a1442d58c3900b4ab643c23871893cad1f4f421c60"
	h2, err := chainhash.NewHashFromStr(txid2)
	if err != nil {
		t.Error(err)
	}
	script2, err := hex.DecodeString("0014df0683535861d41af232009259b5d3811d4471a8")
	if err != nil {
		t.Error(err)
	}
	op2 := wire.OutPoint{
		Hash:  *h2,
		Index: 0,
	}
	utxo2 := wallet.Utxo{Op: op2, ScriptPubkey: script2, AtHeight: 426, Value: 53333300000, WatchOnly: false, Frozen: false}
	err = w.txstore.Utxos().Put(utxo2)
	if err != nil {
		t.Error(err)
	}
	coins := w.gatherCoins(false)
	for _, coin := range coins {
		fmt.Println(coin.Hash().String(), coin.Index(), coin.NumConfs(), coin.Value(), coin.PkScript())
	}
	// test freeze
	err = w.txstore.Utxos().Freeze(utxo1)
	if err != nil {
		t.Error(err)
	}
	// test freeze
	err = w.txstore.Utxos().Freeze(utxo2)
	if err != nil {
		t.Error(err)
	}
	coins = w.gatherCoins(false)
	if len(coins) > 0 {
		t.Fatal("should be no unfrozen coin in map")
	}
}

// The wallet is segwit by default. Here we test making a transaction from 2 inputs
func Test_newSegwitMultiInputTransaction(t *testing.T) {
	w := MockWallet(pw)
	w.blockchainTip = 500
	// utxo 1
	txid1 := "edfab2f9b2a013a36c524bf63e9778a5d13ca8bf1fce279647fbcee30bf7dd62"
	h1, err := chainhash.NewHashFromStr(txid1)
	if err != nil {
		t.Error(err)
	}
	script1, err := hex.DecodeString("0014b8da433782cd9d142f32f726926bcc8161beeaef")
	if err != nil {
		t.Error(err)
	}
	op1 := wire.OutPoint{
		Hash:  *h1,
		Index: 1,
	}
	utxo1 := wallet.Utxo{Op: op1, ScriptPubkey: script1, AtHeight: 426, Value: 17556690040, WatchOnly: false, Frozen: false}
	err = w.txstore.Utxos().Put(utxo1)
	if err != nil {
		t.Error(err)
	}
	// utxo 2
	txid2 := "5ebbbeafb0d23805c09c87a1442d58c3900b4ab643c23871893cad1f4f421c60"
	h2, err := chainhash.NewHashFromStr(txid2)
	if err != nil {
		t.Error(err)
	}
	script2, err := hex.DecodeString("0014df0683535861d41af232009259b5d3811d4471a8")
	if err != nil {
		t.Error(err)
	}
	op2 := wire.OutPoint{
		Hash:  *h2,
		Index: 0,
	}
	utxo2 := wallet.Utxo{Op: op2, ScriptPubkey: script2, AtHeight: 426, Value: 53333300000, WatchOnly: false, Frozen: false}
	err = w.txstore.Utxos().Put(utxo2)
	if err != nil {
		t.Error(err)
	}

	// to harness ->
	address, err := btcutil.DecodeAddress("bcrt1qqfepzsehqytlfvm3gmmx3zrz3yhjw2nm3yuccd", &chaincfg.RegressionNetParams)
	if err != nil {
		t.Error(err)
	}

	_, _, err = w.Spend(
		"abc",
		int64(55500000000), // this amount needs both inputs
		address,            // send to harness ->
		wallet.NORMAL,
	)
	if err != nil {
		t.Error(err)
	}
}

// default segwit transaction - 1 utxo consumed
func Test_newSegwitTransaction(t *testing.T) {
	w := MockWallet("abc")
	w.blockchainTip = 500

	// A real Tx from harness->goele

	// make one utxo
	txid := "50b636d971e7d4d918d92876d6d53a22ccc960e051f540108056ca4ad6ec080c"
	vout := 0
	witnessProgram := "a30a0cf1da8c0c36ae8d637b674663ccf2b31e45"
	h1Txid, err := chainhash.NewHashFromStr(txid)
	if err != nil {
		t.Error(err)
	}
	h1OutIndex := uint32(vout)
	h1WitnessProgram, err := hex.DecodeString(witnessProgram)
	if err != nil {
		t.Error(err)
	}

	segwitAddress, swerr := btcutil.NewAddressWitnessPubKeyHash(h1WitnessProgram, w.params)
	if swerr != nil {
		t.Error(err)
	}

	fmt.Println("Segwit address", segwitAddress.EncodeAddress())

	script, err := txscript.PayToAddrScript(segwitAddress)
	if err != nil {
		t.Error(err)
	}
	op := wire.OutPoint{
		Hash:  *h1Txid,
		Index: h1OutIndex,
	}
	err = w.txstore.Utxos().Put(wallet.Utxo{
		Op:           op,
		ScriptPubkey: script,
		AtHeight:     421,
		Value:        10030000})
	if err != nil {
		t.Error(err)
	}

	// to harness ->
	address, err := btcutil.DecodeAddress("bcrt1q322tg0y2hzyp9zztr7d2twdclhqg88anvzxwwr", &chaincfg.RegressionNetParams)
	if err != nil {
		t.Error(err)
	}

	_, _, err = w.Spend(
		"abc",
		int64(3000000),
		address,
		wallet.NORMAL,
	)
	if err != nil {
		t.Error(err)
	}
}

// The wallet is segwit by default. But we may want to spend from a legacy address
// we previously passed to an entity (CEX?) that does not yet send to segwit bech32
// addresses. Hopefully never though!
func Test_newLegacyTransaction(t *testing.T) {
	w := MockWallet("abc")
	w.blockchainTip = 500

	// make one utxo
	txid := "50b636d971e7d4d918d92876d6d53a22ccc960e051f540108056ca4ad6ec080c"
	vout := 0
	scriptAddress := "a30a0cf1da8c0c36ae8d637b674663ccf2b31e45"
	h1Txid, err := chainhash.NewHashFromStr(txid)
	if err != nil {
		t.Error(err)
	}
	h1OutIndex := uint32(vout)
	h1ScriptAddress, err := hex.DecodeString(scriptAddress)
	if err != nil {
		t.Error(err)
	}

	legacyAddress, lgerr := btcutil.NewAddressPubKeyHash(h1ScriptAddress, w.params)
	if lgerr != nil {
		t.Error(err)
	}

	fmt.Println("legacy address", legacyAddress.EncodeAddress())

	script, err := txscript.PayToAddrScript(legacyAddress)
	if err != nil {
		t.Error(err)
	}
	op := wire.OutPoint{
		Hash:  *h1Txid,
		Index: h1OutIndex,
	}
	err = w.txstore.Utxos().Put(wallet.Utxo{
		Op:           op,
		ScriptPubkey: script,
		AtHeight:     421,
		Value:        10030000})
	if err != nil {
		t.Error(err)
	}

	// to harness ->
	address, err := btcutil.DecodeAddress("bcrt1q322tg0y2hzyp9zztr7d2twdclhqg88anvzxwwr", &chaincfg.RegressionNetParams)
	if err != nil {
		t.Error(err)
	}

	_, _, err = w.Spend(
		"abc",
		int64(100000),
		address,
		wallet.NORMAL,
	)
	if err != nil {
		t.Error(err)
	}
}
