package btc

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/dev-warrior777/go-electrum-client/client"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

func makeBitcoinRegtestTestConfig() (*client.ClientConfig, error) {
	cfg := client.NewDefaultConfig()
	cfg.Chain = wallet.Bitcoin
	cfg.Params = &chaincfg.RegressionNetParams
	cfg.StoreEncSeed = true
	appDir, err := client.GetConfigPath()
	if err != nil {
		return nil, err
	}
	regtestTestDir := filepath.Join(appDir, "btc", "regtest", "test")
	err = os.MkdirAll(regtestTestDir, os.ModeDir|0777)
	if err != nil {
		return nil, err
	}
	cfg.DataDir = regtestTestDir
	return cfg, nil
}

func rmTestDir() error {
	appDir, err := client.GetConfigPath()
	if err != nil {
		return err
	}
	regtestTestDir := filepath.Join(appDir, "btc", "regtest", "test")
	err = os.RemoveAll(regtestTestDir)
	if err != nil {
		return err
	}
	return nil
}

// Create a new standard wallet
func TestWalletCreation(t *testing.T) {
	cfg, err := makeBitcoinRegtestTestConfig()
	if err != nil {
		t.Fatal(err)
	}
	defer rmTestDir()
	cfg.Testing = true
	ec := NewBtcElectrumClient(cfg)
	pw := "abc"
	err = ec.CreateWallet(pw)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("made a btcWallet")

	adr, err := ec.GetWallet().GetUnusedAddress(wallet.EXTERNAL)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Current External address", adr)
	adrI, err := ec.GetWallet().GetUnusedAddress(wallet.INTERNAL)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Current Internal address", adrI)
}
