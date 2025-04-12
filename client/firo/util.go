package firo

import (
	"bytes"
	"crypto/sha256"
	"errors"

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
	if err != nil {
		return nil, err
	}
	if checkIo {
		if len(tx.TxIn) == 0 {
			return nil, errors.New("tx: no inputs")
		}
		if len(tx.TxOut) == 0 {
			return nil, errors.New("tx: no outputs")
		}
	}
	return tx, nil
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

func hash160(buf []byte) ([]byte, error) {
	h := sha256.New()
	_, err := h.Write(buf)
	if err != nil {
		return nil, err
	}
	sha := h.Sum(nil)
	r := ripemd160.New()
	_, err = r.Write(sha)
	if err != nil {
		return nil, err
	}
	ripeMd := r.Sum(nil)
	return ripeMd, nil
}
