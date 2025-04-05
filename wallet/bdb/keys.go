package bdb

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/bisoncraft/go-electrum-client/wallet"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	bolt "go.etcd.io/bbolt"
)

type KeysDB struct {
	db   *bolt.DB
	lock *sync.RWMutex
}

func (k *KeysDB) Put(scriptAddress []byte, keyPath wallet.KeyPath) error {
	krec := &keyRec{
		ScriptAddress: scriptAddress,
		Purpose:       int(keyPath.Purpose),
		KeyIndex:      keyPath.Index,
		Used:          false,
	}
	return k.put(krec)
}

func (k *KeysDB) MarkKeyAsUsed(scriptAddress []byte) error {
	krec, err := k.get(scriptAddress)
	if err != nil {
		return err
	}
	krec.Used = true
	return k.put(krec)
}

// GetLastKeyIndex gets the last (highest) key index stored and whether it has been used.
// If error or no records it will return -1 and error.
func (k *KeysDB) GetLastKeyIndex(purpose wallet.KeyPurpose) (int, bool, error) {
	krecList, err := k.getAllSorted()
	if err != nil {
		return -1, false, err
	}
	if len(krecList) == 0 {
		return -1, false, errors.New("no key records in db")
	}
	var krecListPurpose = make([]keyRec, 0)
	for _, krec := range krecList {
		if krec.Purpose == int(purpose) {
			krecListPurpose = append(krecListPurpose, krec)
		}
	}
	if len(krecListPurpose) == 0 {
		return -1, false, errors.New("no key records in db for purpose")
	}
	lastRec := len(krecListPurpose) - 1
	return krecListPurpose[lastRec].KeyIndex, krecListPurpose[lastRec].Used, nil
}

func (k *KeysDB) GetPathForKey(scriptAddress []byte) (wallet.KeyPath, error) {
	keyPath := wallet.KeyPath{}
	krec, err := k.get(scriptAddress)
	if err != nil {
		return keyPath, err
	}
	keyPath.Purpose = wallet.KeyPurpose(krec.Purpose)
	keyPath.Index = krec.KeyIndex
	return keyPath, nil
}

func (k *KeysDB) GetUnused(purpose wallet.KeyPurpose) ([]int, error) {
	var ret []int
	krecList, err := k.getAllSorted()
	if err != nil {
		return nil, err
	}
	for _, krec := range krecList {
		if purpose == wallet.KeyPurpose(krec.Purpose) && !krec.Used {
			ret = append(ret, krec.KeyIndex)
		}
	}
	return ret, nil
}

func (k *KeysDB) GetAll() ([]wallet.KeyPath, error) {
	var ret []wallet.KeyPath
	krecList, err := k.getAllSorted()
	if err != nil {
		return nil, err
	}
	for _, krec := range krecList {
		keyPath := wallet.KeyPath{
			Purpose: wallet.KeyPurpose(krec.Purpose),
			Index:   krec.KeyIndex,
		}
		ret = append(ret, keyPath)
	}
	return ret, nil
}

func (k *KeysDB) GetDbg() string {
	var ret string
	krecList, err := k.getAllSorted()
	if err != nil {
		return ""
	}
	for _, krec := range krecList {
		scriptAddress := hex.EncodeToString(krec.ScriptAddress)
		var segwitAddrStr string
		segwitAddress, swerr := btcutil.NewAddressWitnessPubKeyHash(
			krec.ScriptAddress, &chaincfg.RegressionNetParams)
		if swerr != nil {
			segwitAddrStr = ""
			fmt.Println(swerr)
		} else {
			segwitAddrStr = segwitAddress.String()
		}
		var purpose string
		if krec.Purpose == int(wallet.EXTERNAL) {
			purpose = "EXTERNAL"
		} else {
			purpose = "INTERNAL"
		}
		keyIndex := strconv.Itoa(krec.KeyIndex)
		var used string
		if krec.Used {
			used = " ** USED **"
		}
		var sb strings.Builder
		sb.WriteString(" Script Address: ")
		sb.WriteString(scriptAddress)
		sb.WriteString("  ")
		sb.WriteString(segwitAddrStr)
		sb.WriteString("\n")
		sb.WriteString(" Key Purpose:    ")
		sb.WriteString(purpose)
		sb.WriteString("\n")
		sb.WriteString(" Key Index:      ")
		sb.WriteString(keyIndex)
		sb.WriteString("\n")
		sb.WriteString(used)
		sb.WriteString("\n\n")
		ret += sb.String()
	}
	return ret
}

func (k *KeysDB) GetLookaheadWindows() map[wallet.KeyPurpose]int {
	windows := make(map[wallet.KeyPurpose]int)
	krecList, err := k.getAllSorted()
	if err != nil || len(krecList) == 0 {
		windows[wallet.EXTERNAL] = 0
		windows[wallet.INTERNAL] = 0
		return windows
	}
	var unusedCountExternal int = 0
	var unusedCountInternal int = 0
	for _, krec := range krecList {
		if krec.Used {
			continue
		}
		if krec.Purpose == int(wallet.EXTERNAL) {
			unusedCountExternal++

		} else { // INTERNAL
			unusedCountInternal++
		}
	}
	windows[wallet.EXTERNAL] = unusedCountExternal
	windows[wallet.INTERNAL] = unusedCountInternal
	return windows
}

// DB access record
type keyRec struct {
	// Unique key - Used as K & V[ScriptAddress]
	ScriptAddress []byte `json:"script_address"`
	Purpose       int    `json:"purpose"`
	KeyIndex      int    `json:"key_index"`
	Used          bool   `json:"used"`
}

func (k *KeysDB) put(krec *keyRec) error {
	k.lock.Lock()
	defer k.lock.Unlock()

	key := krec.ScriptAddress
	if len(key) != 20 {
		return errors.New("bad key length")
	}
	value, err := json.Marshal(krec)
	if err != nil {
		return err
	}

	e := k.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(keysBkt)
		if b == nil {
			return ErrBucketNotFound
		}
		err := b.Put(key, value)
		return err
	})

	return e
}

func (k *KeysDB) get(scriptAddress []byte) (*keyRec, error) {
	k.lock.RLock()
	defer k.lock.RUnlock()

	key := []byte(scriptAddress)
	if len(key) != 20 {
		return nil, errors.New("bad key length")
	}

	var krec keyRec
	e := k.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(keysBkt)
		if b == nil {
			return ErrBucketNotFound
		}
		value := b.Get(key)
		err := json.Unmarshal(value, &krec)
		return err
	})

	return &krec, e
}

// getAllSorted returns all the key records sorted on KeyIndex
func (k *KeysDB) getAllSorted() ([]keyRec, error) {
	k.lock.RLock()
	defer k.lock.RUnlock()

	var krecList []keyRec
	e := k.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(keysBkt)
		if b == nil {
			return ErrBucketNotFound
		}
		b.ForEach(func(k, v []byte) error {
			var krec keyRec
			err := json.Unmarshal(v, &krec)
			if err != nil {
				return err
			}
			krecList = append(krecList, krec)
			return nil
		})
		return nil
	})

	// Sort on KeyIndex
	sort.Slice(krecList, func(i, j int) bool {
		return krecList[i].KeyIndex < krecList[j].KeyIndex
	})

	return krecList, e
}

// delete deletes the record from the db - currently unused except for test
func (k *KeysDB) delete(scriptAddress []byte) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	key := scriptAddress
	if len(key) != 20 {
		return errors.New("bad key length")
	}

	e := k.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(keysBkt)
		if b == nil {
			return ErrBucketNotFound
		}
		err := b.Delete(key)
		return err
	})

	return e
}

func (k *KeysDB) isKeyUsed(scriptAddress []byte) (bool, error) {
	k.lock.RLock()
	defer k.lock.RUnlock()
	krec, err := k.get(scriptAddress)
	if err != nil {
		return false, err
	}
	return krec.Used, nil
}
