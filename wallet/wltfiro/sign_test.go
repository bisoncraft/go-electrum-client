package wltfiro

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/bisoncraft/go-electrum-client/wallet"
	"github.com/btcsuite/btcd/wire"
)

var (
	testUnsignedTx, _ = hex.DecodeString("0100000002a1f14ec5d026a95c9c28909d07480575f1e82587290ab58d238d893c6c4e5e020100000000ffffffffd1376afd51d837578e3f74f1716f95332752f84b9847bc6b7d56190111c49c210100000000ffffffff02463cc32300000000160014dd3c22b42d29ea8ab7ec454e8bce628a07200ccd0027b929000000001600148a94b43c8ab88812884b1f9aa5b9b8fdc0839fb300000000")
	testSignedTx, _   = hex.DecodeString("01000000000102a1f14ec5d026a95c9c28909d07480575f1e82587290ab58d238d893c6c4e5e020100000000ffffffffd1376afd51d837578e3f74f1716f95332752f84b9847bc6b7d56190111c49c210100000000ffffffff02463cc32300000000160014dd3c22b42d29ea8ab7ec454e8bce628a07200ccd0027b929000000001600148a94b43c8ab88812884b1f9aa5b9b8fdc0839fb30247304402205d8e2935940dbaa4b18b1f5ae1e5f6b6c82d468f6e248371a1014142495146d8022038afa4c58fb697ad8757f36f77fcf1ae4e39f17a10ed5f150a3788785f0b6e42012102cb969af83427bfb1d271a7eb16f7fa3d16794a93369d0da293f721e925af9135024730440220652bfde710afa0c64cd5dfd967525d37003c55b708cb34ea4a50a47b85eb1bac02200e518b663bfcfb4386e981c4872ac1a726c3e5fc0f3acc82303168563c7ce690012102cb969af83427bfb1d271a7eb16f7fa3d16794a93369d0da293f721e925af913500000000")
)

func getUtxos() []*wallet.Utxo {
	op1, _ := wire.NewOutPointFromString("01d36086d9851e7ebfdc4bb08be7870145cfb3e1272d89ad2cd0301990533af8:1")
	op2, _ := wire.NewOutPointFromString("025e4e6c3c898d238db50a298725e8f1750548079d90289c5ca926d0c54ef1a1:1")
	op3, _ := wire.NewOutPointFromString("094147051e28eb6e68f343f7deb13a9864a398f7a4d461ac8a3df33dca307ae7:1")
	op4, _ := wire.NewOutPointFromString("219cc4110119567d6bbc47984bf8522733956f71f1743f8e5737d851fd6a37d1:1")
	op5, _ := wire.NewOutPointFromString("5221b16c006e67f23dd2295b24ebd6e452ad30b424721792647c5df0e1b3be42:0")
	op6, _ := wire.NewOutPointFromString("58ab39331daa512aa0cdbc2b7adbfc0e6a7b96bb5e076402e8d10b9447c44c97:1")
	op7, _ := wire.NewOutPointFromString("5ea497f8d6edd2471303c2091e2e84770e40c83d1849cd08ac9edd2102b7f164:0")
	op8, _ := wire.NewOutPointFromString("72639920285ff6ed212b1a14b65a4175f5409ba25c8c735a725560eafa027dbe:0")
	script1, _ := hex.DecodeString("0014a30a0cf1da8c0c36ae8d637b674663ccf2b31e45")
	script2, _ := hex.DecodeString("0014a30a0cf1da8c0c36ae8d637b674663ccf2b31e45")
	script3, _ := hex.DecodeString("0014a30a0cf1da8c0c36ae8d637b674663ccf2b31e45")
	script4, _ := hex.DecodeString("0014a30a0cf1da8c0c36ae8d637b674663ccf2b31e45")
	script5, _ := hex.DecodeString("0014a30a0cf1da8c0c36ae8d637b674663ccf2b31e45")
	script6, _ := hex.DecodeString("0014a30a0cf1da8c0c36ae8d637b674663ccf2b31e45")
	script7, _ := hex.DecodeString("0014df0683535861d41af232009259b5d3811d4471a8")
	script8, _ := hex.DecodeString("0014a30a0cf1da8c0c36ae8d637b674663ccf2b31e45")

	var utxos = make([]*wallet.Utxo, 0, 10)
	utxos = []*wallet.Utxo{
		{
			Op:           *op1,
			Value:        300000000,
			AtHeight:     208,
			ScriptPubkey: script1,
			WatchOnly:    false,
			Frozen:       false,
		},
		{
			Op:           *op2,
			Value:        600000000,
			AtHeight:     208,
			ScriptPubkey: script2,
			WatchOnly:    false,
			Frozen:       false,
		},
		{
			Op:           *op3,
			Value:        200000000,
			AtHeight:     208,
			ScriptPubkey: script3,
			WatchOnly:    false,
			Frozen:       false,
		},
		{
			Op:           *op4,
			Value:        700000000,
			AtHeight:     208,
			ScriptPubkey: script4,
			WatchOnly:    false,
			Frozen:       false,
		},
		{
			Op:           *op5,
			Value:        100000000,
			AtHeight:     208,
			ScriptPubkey: script5,
			WatchOnly:    false,
			Frozen:       false,
		},
		{
			Op:           *op6,
			Value:        500000000,
			AtHeight:     208,
			ScriptPubkey: script6,
			WatchOnly:    false,
			Frozen:       false,
		},
		{
			Op:           *op7,
			Value:        110000000,
			AtHeight:     215,
			ScriptPubkey: script7,
			WatchOnly:    false,
			Frozen:       false,
		},
		{
			Op:           *op8,
			Value:        400000000,
			AtHeight:     208,
			ScriptPubkey: script8,
			WatchOnly:    false,
			Frozen:       false,
		},
	}
	return utxos
}

func TestSignTx(t *testing.T) {
	tx := wire.NewMsgTx(wire.TxVersion)
	r := bytes.NewBuffer(testUnsignedTx)
	_ = tx.Deserialize(r)
	info := &wallet.SigningInfo{
		UnsignedTx: tx,
		VerifyTx:   true,
	}
	w := MockWallet("abc")
	w.blockchainTip = 500
	utxos := getUtxos()
	for _, utxo := range utxos {
		err := w.txstore.Utxos().Put(*utxo)
		if err != nil {
			t.Error(err)
		}
	}
	signed, err := w.SignTx("abc", info)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(signed, testSignedTx) {
		t.Error(err)
	}
}
