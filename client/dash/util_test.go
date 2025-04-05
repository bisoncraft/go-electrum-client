package dash

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/decred/dcrd/crypto/ripemd160"
)

type test struct {
	in []byte

	sha256    []byte
	ripemd160 []byte
}

var in0 = []byte{} // 0x00
var sha0, _ = hex.DecodeString("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
var ripe0, _ = hex.DecodeString("b472a266d0bd89c13706a4132ccfb16f7c3b9fcb")

var in1, _ = hex.DecodeString("c7dc38180a7f84f5ca271fc9c63c0dcfaff2b895c528f6c9e604211cf1560bdd")
var sha1, _ = hex.DecodeString("4ae1d80a76b629a650c81fdf5fd7a76c0c1f769777d187b675b79d3ffa608cb3")
var ripe1, _ = hex.DecodeString("eeb99f1e8e284bcefcc19a63e6b6451488d7aee0")

var in2, _ = hex.DecodeString("24ce98b4306cb2c62f1817e35d925fed5a73344dab5fd99e4395ea42553069af")
var sha2, _ = hex.DecodeString("0663e05d985f43132be269b619d6a802f6557fef688dd3dea70e74eff8da4d5a")
var ripe2, _ = hex.DecodeString("99616c8c5006617b86f78731791398d6eca2b61c")

var in3, _ = hex.DecodeString("8277516a29e38542c1e87799fe29651e4e0f105d7b3c96c0ef9961ebce0dcc3a")
var sha3, _ = hex.DecodeString("b2f5809d41667d4df2bcd52fa42b64d9a4b6478c7907b0ad7cf092ab58bce814")
var ripe3, _ = hex.DecodeString("b87197617e4982a75327f609bdf50d8097c5db44")

var in4, _ = hex.DecodeString("2ebf053aa59f4c4b9148a9b45b21ff48a73b485153b03a44ad3b32f17486fad3")
var sha4, _ = hex.DecodeString("e1d89af552a9c05ead55559561d9cb0d759e40ba99f37095bbb588f9a089487c")
var ripe4, _ = hex.DecodeString("004361bf0e4ad797f59a38088da7cfe3e4353962")

var tests = [...]test{
	{nil, sha0, ripe0},
	{in0, sha0, ripe0},
	{in1, sha1, ripe1},
	{in2, sha2, ripe2},
	{in3, sha3, ripe3},
	{in4, sha4, ripe4},
}

func TestSha256(t *testing.T) {
	for i := 0; i < len(tests); i++ {
		sha256 := calcHash(tests[i].in, sha256.New())
		if !bytes.Equal(sha256, tests[i].sha256) {
			t.Fatalf("incorrect sha256 %x", sha256)
		}
		ripemd160 := calcHash(sha256, ripemd160.New())
		if !bytes.Equal(ripemd160, tests[i].ripemd160) {
			t.Fatalf("incorrect ripemd160 %x", ripemd160)
		}
	}
}

// ripemd160(sha256(n))
func TestHash160(t *testing.T) {
	for i := 0; i < len(tests); i++ {
		ripemd160 := hash160(tests[i].in)
		if !bytes.Equal(ripemd160, tests[i].ripemd160) {
			t.Fatalf("incorrect ripemd160 %x", ripemd160)
		}
	}
}
