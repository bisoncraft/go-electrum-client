package dash

import (
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
)

var (
	// bitcoin genesis mainnet
	bgen           = "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa"
	bgenScriptHash = "8b01df4e368ea28f8dc0423bcf7a4923e3a12d307c875e47a0cfbf90b5c39161"
	// goele wallet regtest
	a1           = "mvP2UeXooRghYvsX7H7XVj78FY49jJw6Sq"
	a1Scripthash = "6036b7e9dcb352f2d7bb4ad0efe0f06e03ba58fad4d16e943a25ae41082d1934"
	// electrum wallet regtest
	ab           = "bcrt1q3fx029uese6mrhvq68u4l6me49refj8maqxvfv"
	abScripthash = "02c21ac0ef859617cbb7ae68943b9af8fb99699d32ea35cb449384aac17b93d5"
	// electrumX regtest
	aWSH           = "bcrt1qdql55es0t6afs9gy9th2magjncahp0fxhs4jkn20mqjt4hjyjesqvp5ls8"
	aWSHScripthash = "c27062c2d7439f6b583704f576478141490a2197e6052bbde14851f24ac0e8b5"
)

func TestElectrumScripthash(t *testing.T) {
	_, err := addrToElectrumScripthash("", &chaincfg.MainNetParams)
	if err == nil {
		t.Fatal(err)
	}

	shGen, err := addrToElectrumScripthash(bgen, &chaincfg.MainNetParams)
	if err != nil {
		t.Fatal(err)
	}
	if shGen != bgenScriptHash {
		t.Fatal(err)
	}

	sh1, err := addrToElectrumScripthash(a1, &chaincfg.RegressionNetParams)
	if err != nil {
		t.Fatal(err)
	}
	if sh1 != a1Scripthash {
		t.Fatal(err)
	}

	shb, err := addrToElectrumScripthash(ab, &chaincfg.RegressionNetParams)
	if err != nil {
		t.Fatal(err)
	}
	if shb != abScripthash {
		t.Fatal(err)
	}

	shWSH, err := addrToElectrumScripthash(aWSH, &chaincfg.RegressionNetParams)
	if err != nil {
		t.Fatal(err)
	}
	if shWSH != aWSHScripthash {
		t.Fatal(err)
	}
}
