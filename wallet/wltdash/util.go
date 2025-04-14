package wltdash

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
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

// NewOutPointFromString returns a new bitcoin transaction outpoint parsed from
// the provided string, which should be in the format "hash:index".
func NewOutPointFromString(outpoint string) (*wire.OutPoint, error) {
	parts := strings.Split(outpoint, ":")
	if len(parts) != 2 {
		return nil, errors.New("outpoint should be of the form txid:index")
	}
	hash, err := chainhash.NewHashFromStr(parts[0])
	if err != nil {
		return nil, err
	}

	outputIndex, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid output index: %v", err)
	}

	return &wire.OutPoint{
		Hash:  *hash,
		Index: uint32(outputIndex),
	}, nil
}

func outPointsEqual(a, b wire.OutPoint) bool {
	if !a.Hash.IsEqual(&b.Hash) {
		return false
	}
	return a.Index == b.Index
}
