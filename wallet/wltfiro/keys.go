package wltfiro

import (
	"errors"

	"github.com/bisoncraft/go-electrum-client/client"
	"github.com/bisoncraft/go-electrum-client/wallet"
	hd "github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
)

// Lookahead window size from client constants
const GAP_LIMIT = client.GAP_LIMIT

type KeyManager struct {
	datastore wallet.Keys
	params    *chaincfg.Params

	internalKey *hd.ExtendedKey
	externalKey *hd.ExtendedKey
}

func NewKeyManager(db wallet.Keys, params *chaincfg.Params, masterPrivKey *hd.ExtendedKey) (*KeyManager, error) {
	internal, external, err := Bip44Derivation(masterPrivKey)
	masterPrivKey.Zero()
	if err != nil {
		return nil, err
	}
	km := &KeyManager{
		datastore:   db,
		params:      params,
		internalKey: internal,
		externalKey: external,
	}
	if err := km.lookahead(); err != nil {
		return nil, err
	}
	return km, nil
}

// m / purpose' / coin_type' / account' / change / address_index
func Bip44Derivation(masterPrivKey *hd.ExtendedKey) (internal, external *hd.ExtendedKey, err error) {
	// Purpose = bip44
	fourtyFour, err := masterPrivKey.Derive(hd.HardenedKeyStart + 44)
	if err != nil {
		return nil, nil, err
	}
	// Cointype = bitcoin
	bitcoin, err := fourtyFour.Derive(hd.HardenedKeyStart + 0)
	if err != nil {
		return nil, nil, err
	}
	// Account = 0
	account, err := bitcoin.Derive(hd.HardenedKeyStart + 0)
	if err != nil {
		return nil, nil, err
	}
	// Change(0) = external
	external, err = account.Derive(0)
	if err != nil {
		return nil, nil, err
	}
	// Change(1) = internal
	internal, err = account.Derive(1)
	if err != nil {
		return nil, nil, err
	}
	return internal, external, nil
}

// GetUnusedKey gets the first unused key for 'purpose'. CAUTION: There may not
// be any keys within the gap limit. In this case a used key can be utilized or
// user can wait until the gap is updated with new key(s). This happens when a
// transaction newly gets client.AGEDTX confirmations.
func (km *KeyManager) GetUnusedKey(purpose wallet.KeyChange) (*hd.ExtendedKey, error) {
	i, err := km.datastore.GetUnused(purpose)
	if err != nil {
		return nil, err
	}
	if len(i) == 0 {
		return nil, errors.New("no unused keys in database")
	}
	return km.generateChildKey(purpose, uint32(i[0]))
}

func (km *KeyManager) GetFreshKey(purpose wallet.KeyChange) (*hd.ExtendedKey, error) {
	index, _, err := km.datastore.GetLastKeyIndex(purpose)
	var childKey *hd.ExtendedKey
	if err != nil {
		index = 0
	} else {
		index += 1
	}
	for {
		// There is a small possibility bip32 keys can be invalid. The procedure in such cases
		// is to discard the key and derive the next one. This loop will continue until a valid key
		// is derived.
		childKey, err = km.generateChildKey(purpose, uint32(index))
		if err == nil {
			break
		}
		index += 1
	}
	addr, err := childKey.Address(km.params)
	if err != nil {
		return nil, err
	}
	p := wallet.KeyPath{
		Change: wallet.KeyChange(purpose),
		Index:  index,
	}
	err = km.datastore.Put(addr.ScriptAddress(), p)
	if err != nil {
		return nil, err
	}
	return childKey, nil
}

func (km *KeyManager) GetKeys() []*hd.ExtendedKey {
	var keys []*hd.ExtendedKey
	keyPaths, err := km.datastore.GetAll()
	if err != nil {
		return keys
	}
	for _, path := range keyPaths {
		k, err := km.generateChildKey(path.Change, uint32(path.Index))
		if err != nil {
			continue
		}
		keys = append(keys, k)
	}
	return keys
}

func (km *KeyManager) GetKeyForScript(scriptAddress []byte) (*hd.ExtendedKey, error) {
	keyPath, err := km.datastore.GetPathForKey(scriptAddress)
	if err != nil {
		return nil, err
	}
	return km.generateChildKey(keyPath.Change, uint32(keyPath.Index))
}

// Mark the given key as used and extend the lookahead window
func (km *KeyManager) MarkKeyAsUsed(scriptAddress []byte) error {
	if err := km.datastore.MarkKeyAsUsed(scriptAddress); err != nil {
		return err
	}
	return km.lookahead()
}

func (km *KeyManager) generateChildKey(purpose wallet.KeyChange, index uint32) (*hd.ExtendedKey, error) {
	if purpose == wallet.EXTERNAL {
		return km.externalKey.Derive(index)
	} else if purpose == wallet.INTERNAL {
		return km.internalKey.Derive(index)
	}
	return nil, errors.New("unknown key purpose")
}

func (km *KeyManager) lookahead() error {
	lookaheadWindows := km.datastore.GetLookaheadWindows()
	for purpose, size := range lookaheadWindows {
		if size < GAP_LIMIT {
			for i := 0; i < (GAP_LIMIT - size); i++ {
				_, err := km.GetFreshKey(purpose)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
