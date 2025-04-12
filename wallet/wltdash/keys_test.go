package wltdash

import (
	"encoding/hex"
	"testing"

	"github.com/bisoncraft/go-electrum-client/client"
	"github.com/bisoncraft/go-electrum-client/wallet"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
)

func createKeyManager() (*KeyManager, error) {
	masterPrivKey, err := hdkeychain.NewKeyFromString("xprv9s21ZrQH143K25QhxbucbDDuQ4naNntJRi4KUfWT7xo4EKsHt2QJDu7KXp1A3u7Bi1j8ph3EGsZ9Xvz9dGuVrtHHs7pXeTzjuxBrCmmhgC6")
	if err != nil {
		return nil, err
	}
	return NewKeyManager(&mockKeyStore{make(map[string]*keyStoreEntry)}, &chaincfg.MainNetParams, masterPrivKey)
}

func TestNewKeyManager(t *testing.T) {
	km, err := createKeyManager()
	if err != nil {
		t.Error(err)
	}
	keys, err := km.datastore.GetAll()
	if err != nil {
		t.Error(err)
	}
	if len(keys) != client.GAP_LIMIT*2 {
		t.Error("Failed to generate lookahead windows when creating a new KeyManager")
	}
}

func TestBip44Derivation(t *testing.T) {
	masterPrivKey, err := hdkeychain.NewKeyFromString("xprv9s21ZrQH143K25QhxbucbDDuQ4naNntJRi4KUfWT7xo4EKsHt2QJDu7KXp1A3u7Bi1j8ph3EGsZ9Xvz9dGuVrtHHs7pXeTzjuxBrCmmhgC6")
	if err != nil {
		t.Error(err)
	}
	internal, external, err := Bip44Derivation(masterPrivKey)
	if err != nil {
		t.Error(err)
	}
	externalKey, err := external.Derive(0)
	if err != nil {
		t.Error(err)
	}
	externalAddr, err := externalKey.Address(&chaincfg.MainNetParams)
	if err != nil {
		t.Error(err)
	}
	if externalAddr.String() != "17rxURoF96VhmkcEGCj5LNQkmN9HVhWb7F" {
		t.Error("Incorrect Bip44 key derivation")
	}

	internalKey, err := internal.Derive(0)
	if err != nil {
		t.Error(err)
	}
	internalAddr, err := internalKey.Address(&chaincfg.MainNetParams)
	if err != nil {
		t.Error(err)
	}
	if internalAddr.String() != "16wbbYdecq9QzXdxa58q2dYXJRc8sfkE4J" {
		t.Error("Incorrect Bip44 key derivation")
	}
}

func TestKeys_generateChildKey(t *testing.T) {
	km, err := createKeyManager()
	if err != nil {
		t.Error(err)
	}
	internalKey, err := km.generateChildKey(wallet.INTERNAL, 0)
	if err != nil {
		t.Error(err)
	}
	internalAddr, err := internalKey.Address(&chaincfg.MainNetParams)
	if err != nil {
		t.Error(err)
	}
	if internalAddr.String() != "16wbbYdecq9QzXdxa58q2dYXJRc8sfkE4J" {
		t.Error("generateChildKey returned incorrect key")
	}
	externalKey, err := km.generateChildKey(wallet.EXTERNAL, 0)
	if err != nil {
		t.Error(err)
	}
	externalAddr, err := externalKey.Address(&chaincfg.MainNetParams)
	if err != nil {
		t.Error(err)
	}
	if externalAddr.String() != "17rxURoF96VhmkcEGCj5LNQkmN9HVhWb7F" {
		t.Error("generateChildKey returned incorrect key")
	}
}

func TestKeyManager_lookahead(t *testing.T) {
	masterPrivKey, err := hdkeychain.NewKeyFromString("xprv9s21ZrQH143K25QhxbucbDDuQ4naNntJRi4KUfWT7xo4EKsHt2QJDu7KXp1A3u7Bi1j8ph3EGsZ9Xvz9dGuVrtHHs7pXeTzjuxBrCmmhgC6")
	if err != nil {
		t.Error(err)
	}
	mock := &mockKeyStore{make(map[string]*keyStoreEntry)}
	km, err := NewKeyManager(mock, &chaincfg.MainNetParams, masterPrivKey)
	if err != nil {
		t.Error(err)
	}
	for _, key := range mock.keys {
		key.used = true
	}
	n := len(mock.keys)
	err = km.lookahead()
	if err != nil {
		t.Error(err)
	}
	if len(mock.keys) != n+(client.GAP_LIMIT*2) {
		t.Error("Failed to generated a correct lookahead window")
	}
	unused := 0
	for _, k := range mock.keys {
		if !k.used {
			unused++
		}
	}
	if unused != client.GAP_LIMIT*2 {
		t.Error("Failed to generated unused keys in lookahead window")
	}
}

func TestKeyManager_MarkKeyAsUsed(t *testing.T) {
	km, err := createKeyManager()
	if err != nil {
		t.Error(err)
	}
	i, err := km.datastore.GetUnused(wallet.EXTERNAL)
	if err != nil {
		t.Error(err)
	}
	if len(i) == 0 {
		t.Error("No unused keys in database")
	}
	key, err := km.generateChildKey(wallet.EXTERNAL, uint32(i[0]))
	if err != nil {
		t.Error(err)
	}
	addr, err := key.Address(&chaincfg.MainNetParams)
	if err != nil {
		t.Error(err)
	}
	err = km.MarkKeyAsUsed(addr.ScriptAddress())
	if err != nil {
		t.Error(err)
	}
	if len(km.GetKeys()) != (client.GAP_LIMIT*2)+1 {
		t.Error("Failed to extend lookahead window when marking as read")
	}
	unused, err := km.datastore.GetUnused(wallet.EXTERNAL)
	if err != nil {
		t.Error(err)
	}
	for _, i := range unused {
		if i == 0 {
			t.Error("Failed to mark key as used")
		}
	}
}

func TestKeyManager_GetUnusedKey(t *testing.T) {
	masterPrivKey, err := hdkeychain.NewKeyFromString("xprv9s21ZrQH143K25QhxbucbDDuQ4naNntJRi4KUfWT7xo4EKsHt2QJDu7KXp1A3u7Bi1j8ph3EGsZ9Xvz9dGuVrtHHs7pXeTzjuxBrCmmhgC6")
	if err != nil {
		t.Error(err)
	}
	mock := &mockKeyStore{make(map[string]*keyStoreEntry)}
	km, err := NewKeyManager(mock, &chaincfg.MainNetParams, masterPrivKey)
	if err != nil {
		t.Error(err)
	}
	var scriptAddress string
	for script, key := range mock.keys {
		if key.path.Change == wallet.EXTERNAL && key.path.Index == 0 {
			scriptAddress = script
			break
		}
	}
	key, err := km.GetUnusedKey(wallet.EXTERNAL)
	if err != nil {
		t.Error(err)
	}
	addr, err := key.Address(&chaincfg.Params{})
	if err != nil {
		t.Error(err)
	}
	if hex.EncodeToString(addr.ScriptAddress()) != scriptAddress {
		t.Error("CurrentKey returned wrong key")
	}
}

func TestKeyManager_GetFreshKey(t *testing.T) {
	km, err := createKeyManager()
	if err != nil {
		t.Error(err)
	}
	key, err := km.GetFreshKey(wallet.EXTERNAL)
	if err != nil {
		t.Error(err)
	}
	if len(km.GetKeys()) != client.GAP_LIMIT*2+1 {
		t.Error("Failed to create additional key")
	}
	edgeCaseKeyNumber := uint32(client.GAP_LIMIT)
	key2, err := km.generateChildKey(wallet.EXTERNAL, edgeCaseKeyNumber)
	if err != nil {
		t.Error(err)
	}
	if key.String() != key2.String() {
		t.Error("GetFreshKey returned incorrect key")
	}
}

func TestKeyManager_GetKeys(t *testing.T) {
	km, err := createKeyManager()
	if err != nil {
		t.Error(err)
	}
	keys := km.GetKeys()
	if len(keys) != client.GAP_LIMIT*2 {
		t.Error("Returned incorrect number of keys")
	}
	for _, key := range keys {
		if key == nil {
			t.Error("Incorrectly returned nil key")
		}
	}
}
