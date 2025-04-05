package wltdash

import (
	"encoding/hex"
	"errors"
	"testing"
	"time"
)

type rawTx struct {
	txid string
	tx   string
}
type rawTransactions []rawTx

func makeTxList() rawTransactions {
	var txList = make(rawTransactions, 0)

	txList = rawTransactions{
		{
			txid: "86218441bf448a2f406cfa881b455f314225a2bb01c75bc1726223074745f1af",
			tx:   "0200000000010105df9f46364604db2a66bc3012f3d6021e66fbeefa9907157702b4b3696dc5860000000000feffffff024c078d2500000000160014d7533dc3d62e597935117b26ec5d222e7fcd9958801d2c0400000000160014a30a0cf1da8c0c36ae8d637b674663ccf2b31e4502473044022018f8bfe1513e60517f5a50076f13417461d6a9c10fe4a78389ff9829e97ac683022070e5a0dc6224cf15b70863314a7651c5b80fd1c6d9309000c1b511cbf35ed7230121022e9b5f62989de06630e30db76335c6bdcf07ed5dcf81b7174f3ee22022e06c8700000000",
		},
		{
			txid: "3e082b08c681a7a910336a25a720f73d7e6ecf5172a4d8b53bf96519c929e42a",
			tx:   "020000000001013dbb19d942ecb5ad9d471b3da5fcd7f481a0499c7b53f02a5128817ae547a27e0000000000feffffff02f28569310000000016001468525477d468dbd72b2c66abe54b9121f1eb6444c05f3b0400000000160014a30a0cf1da8c0c36ae8d637b674663ccf2b31e4502473044022074efe4ab61234d8f51cc3eae7a7faf8f2d99525f9858c73a0201ec8830bd0493022023f519bbd8ebc21e8049938c4e3b7df8716c4fcd9cbc6abc6f11e4209e7b73ce0121020c1d23e64345fe12c9282c4b5bea6d900d7c7c2ffef94393fea643a11b07f08000000000",
		},
		{
			txid: "97a6c1d3db91e0fede0efc4e0b43d81103c651780f2f5427333d1994364f6443",
			tx:   "020000000001013dbb19d942ecb5ad9d471b3da5fcd7f481a0499c7b53f02a5128817ae547a27e0100000000feffffff0200a24a0400000000160014a30a0cf1da8c0c36ae8d637b674663ccf2b31e45733eab010000000016001406f35d1e2b599f727117e6d4914210f494f8648002473044022004cfdee600a873075f2cce1ebbaf8e6830dfc6c95f43791e79d873e24b78b2a102201dba314f33f7ac29bd8990a2caffa8271c5795725d9e32b44d909f4ffdd431ea0121030958f6fb97080b0e3fed9320316e49d3bdb113345e82caf8e5525d095c70cbb100000000",
		},
		{
			txid: "a52ae555c0fe69a79b48cdebe7f04911b8c648939dec70ead09cdc449c2c977d",
			tx:   "02000000000101c8aefbdd6403accc71af7e9f185d08b73f80fe61c5a87a661e9149ca361331b40100000000feffffff0240e4590400000000160014a30a0cf1da8c0c36ae8d637b674663ccf2b31e4533be870d00000000160014b547133f30188a6109dfcbb6332ddf7a2a70bc690247304402200ff99c369f2725b0e5221eff86af72d683a3a5095fb0ac7f0fe697591d17422b022016fc9ad8f389b72633b75d34e197080b4377df0b1b3f6d5c16867e26b63d77d80121030958f6fb97080b0e3fed9320316e49d3bdb113345e82caf8e5525d095c70cbb100000000",
		},
		{
			txid: "fe24aa503afd6bd4eebbd3dae550322d1fa758f00c2673607db4d6ed4340edff",
			tx:   "020000000001012e38e4f3b64f31274980f02d4cccc215b2838c90bf51f2faf19852e797f1840d0000000000feffffff02f33d6419000000001600146ac1724535a735f60a6c45ea3c3a79a4fb4c3ac08026690400000000160014a30a0cf1da8c0c36ae8d637b674663ccf2b31e45024730440220348fd3b3bc6afd4b3475f372b69d5ca4ec524f05c293944d6fa969eaab7499bb02201c7f6794780aa84960ae2d7bf84879797b0858f41b14bad9ae64df9e44d9b64d0121030958f6fb97080b0e3fed9320316e49d3bdb113345e82caf8e5525d095c70cbb100000000",
		},
		{
			txid: "d525be56e63fd38729e0e75429982781b69470fac310865ea940452ad582d498",
			tx:   "020000000001017b8465411882bc2eba55bbf0ed0edc2ccfc956ac9afd355e6e6e78529f6b46c40100000000feffffff02c068780400000000160014a30a0cf1da8c0c36ae8d637b674663ccf2b31e45b3bd40250000000016001497919f863ec61f31f494c46adab1750da8d7fb200247304402203aeb2d1300ee640c2484fb7b7d51432dffcd6ea43bcfd1f98afd27d685a3578d02204d712cb43a45524a5cae5e50533f64b0471ed15696d71b5aa5ca98e01d1417560121030958f6fb97080b0e3fed9320316e49d3bdb113345e82caf8e5525d095c70cbb100000000",
		},
		{
			txid: "759e0c2fa47a6fe8eb45dbb2042bb2f45a97886ad7131d7c468819f06c3be29c",
			tx:   "02000000000101a2a65004d6f71400909a82a5d561e32f5b435056dc326539ba286ce0f96a04d50000000000feffffff0200ab870400000000160014a30a0cf1da8c0c36ae8d637b674663ccf2b31e45731e133700000000160014ec2340ae3c66dd1b47f10d8f09eec6f49d13c7a90247304402206755619dac197f3f85bec16119fa99e8f5802f3f4fc1444c99052e5de207b1ad02206b2c08f17a7e47ac31b7836ceb644a2285560aba456b602c9dbf9ec437fee90a0121030958f6fb97080b0e3fed9320316e49d3bdb113345e82caf8e5525d095c70cbb100000000",
		},
		{
			txid: "799cb818766b88db4ae74e2b748b5792262e9fdc7065ccae4eff06ad2b9911a1",
			tx:   "020000000001019626eb4be6da1dd39b8264c262e7673b2b1dd30fa551eccd3004391419afaf770100000000feffffff0240ed960400000000160014a30a0cf1da8c0c36ae8d637b674663ccf2b31e453341d15400000000160014add8f22e9742fc34d2a0576772d4b3df94ee51020247304402202e1d7cbbc767a528403812185d6cd1c46b44c5eeb0c78c9ea40b65f363f1eebe0220027b2eebf11420bad036ce4ff11e6f99de04767b8897c3aaf46d7b1b3cedd42a0121030958f6fb97080b0e3fed9320316e49d3bdb113345e82caf8e5525d095c70cbb100000000",
		},
	}
	return txList
}

func fundWallet(w *DashElectrumWallet, height int64, timeStamp time.Time) error {
	txList := makeTxList()
	for _, rawTxStr := range txList {
		rawTx, err := hex.DecodeString(rawTxStr.tx)
		if err != nil {
			return err
		}
		msgTx, err := newWireTx(rawTx, true)
		if err != nil {
			return err
		}
		hits, err := w.txstore.AddTransaction(msgTx, height, timeStamp)
		if err != nil {
			return err
		}
		if hits == 0 {
			return err
		}
		txn, err := w.GetTransaction(rawTxStr.txid)
		if err != nil {
			return errors.New("no hits")
		}
		if txn.Height != height {
			return errors.New("bad tx height")
		}
	}
	return nil
}

func TestFundWallet(t *testing.T) {
	w := MockWallet("abc")
	err := fundWallet(w, 0, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	err = fundWallet(w, 0, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	err = fundWallet(w, 1, time.Now())
	if err != nil {
		t.Fatal(err)
	}
}

func TestBalance(t *testing.T) {
	w := MockWallet("abc")
	err := fundWallet(w, 0, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	c, u, l, err := w.Balance()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("confirmed: %d, unconfirmed: %d, locked: %d, ", c, u, l)
	if u != 588000000 {
		t.Fatal(err)
	}

	txns, err := w.txstore.Txns().GetAll(false)
	if err != nil {
		t.Fatal(err)
	}
	for _, txn := range txns {
		msgTx, err := newWireTx(txn.Bytes, true)
		if err != nil {
			t.Fatal(err)
		}
		err = w.AddTransaction(msgTx, 1, time.Now())
		if err != nil {
			t.Fatal(err)
		}
	}
	c, u, l, err = w.Balance()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("confirmed: %d, unconfirmed: %d, locked: %d, ", c, u, l)
	if c != 588000000 {
		t.Fatal(err)
	}

	unspents, err := w.txstore.Utxos().GetAll()
	if err != nil {
		t.Fatal(err)
	}
	for _, unspent := range unspents {
		w.FreezeUTXO(&unspent.Op)
	}
	c, u, l, err = w.Balance()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("confirmed: %d, unconfirmed: %d, locked: %d, ", c, u, l)
	if l != 588000000 {
		t.Fatal(err)
	}
}
