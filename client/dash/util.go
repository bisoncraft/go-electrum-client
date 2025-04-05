package dash

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"hash"

	"github.com/btcsuite/btcd/wire"
	"github.com/decred/dcrd/crypto/ripemd160"
)

// /////////////////////////////////////////////////////////////////////////////
// Helpers
// ///////

func newWireTx(b []byte, checkIo bool) (*wire.MsgTx, error) {
	tx := wire.NewMsgTx(wire.TxVersion)
	r := bytes.NewBuffer(b)
	err := tx.Deserialize(r)
	if checkIo {
		if len(tx.TxIn) == 0 {
			return nil, errors.New("tx: no inputs")
		}
		if len(tx.TxOut) == 0 {
			return nil, errors.New("tx: no outputs")
		}
	}
	return tx, err
}

func serializeWireTx(tx *wire.MsgTx) ([]byte, error) {
	b := make([]byte, 0, tx.SerializeSize())
	w := bytes.NewBuffer(b)
	err := tx.Serialize(w)
	if err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

// Calculate the hash of hasher over buf.
func calcHash(buf []byte, hasher hash.Hash) []byte {
	_, _ = hasher.Write(buf)
	return hasher.Sum(nil)
}

// hash160 calculates the hash ripemd160(sha256(b)).
func hash160(buf []byte) []byte {
	return calcHash(calcHash(buf, sha256.New()), ripemd160.New())
}
