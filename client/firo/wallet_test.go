package firo

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bisoncraft/go-electrum-client/client"
	"github.com/bisoncraft/go-electrum-client/wallet"
	"github.com/btcsuite/btcd/chaincfg"
)

func makeBitcoinRegtestTestConfig() (*client.ClientConfig, error) {
	cfg := client.NewDefaultConfig()
	cfg.CoinType = wallet.Bitcoin
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
	ec := NewFiroElectrumClient(cfg)
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
