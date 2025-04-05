package wltbtc

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/bisoncraft/go-electrum-client/wallet"
)

type ClosingBuffer struct {
	*bytes.Buffer
}

func (cb *ClosingBuffer) Close() (err error) {
	return
}

type mockHttpClient struct{}

func (m *mockHttpClient) Get(url string) (*http.Response, error) {
	data := `{"priority":450,"normal":420,"economic":390}`
	cb := &ClosingBuffer{bytes.NewBufferString(data)}
	resp := &http.Response{
		Body: cb,
	}
	return resp, nil
}

func TestFeeProvider_GetFeePerByte(t *testing.T) {
	fp := wallet.NewFeeProvider(2000, 360, 320, 280, "https://mempool.space/api/v1/fees/recommended", nil)
	// fp := wallet.NewFeeProvider(2000, 360, 320, 280, "https://mempool.space/testnet/api/v1/fees/recommended", nil)
	fp.HttpClient = new(mockHttpClient)

	// Test fetch from API
	if fp.GetFeePerByte(wallet.PRIORITY) != 450 {
		t.Error("Returned incorrect fee per byte")
	}
	if fp.GetFeePerByte(wallet.NORMAL) != 420 {
		t.Error("Returned incorrect fee per byte")
	}
	if fp.GetFeePerByte(wallet.ECONOMIC) != 390 {
		t.Error("Returned incorrect fee per byte")
	}
	if fp.GetFeePerByte(wallet.FEE_BUMP) != 450 {
		t.Error("Returned incorrect fee per byte")
	}

	// Test return over max
	fp.MaxFee = 100
	if fp.GetFeePerByte(wallet.PRIORITY) != 100 {
		t.Error("Returned incorrect fee per byte")
	}
	if fp.GetFeePerByte(wallet.NORMAL) != 100 {
		t.Error("Returned incorrect fee per byte")
	}
	if fp.GetFeePerByte(wallet.ECONOMIC) != 100 {
		t.Error("Returned incorrect fee per byte")
	}
	if fp.GetFeePerByte(wallet.FEE_BUMP) != 100 {
		t.Error("Returned incorrect fee per byte")
	}

	// Test no API provided
	fp.FeeAPI = ""
	if fp.GetFeePerByte(wallet.PRIORITY) != 360 {
		t.Error("Returned incorrect fee per byte")
	}
	if fp.GetFeePerByte(wallet.NORMAL) != 320 {
		t.Error("Returned incorrect fee per byte")
	}
	if fp.GetFeePerByte(wallet.ECONOMIC) != 280 {
		t.Error("Returned incorrect fee per byte")
	}
	if fp.GetFeePerByte(wallet.FEE_BUMP) != 360 {
		t.Error("Returned incorrect fee per byte")
	}
}
