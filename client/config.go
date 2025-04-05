package client

import (
	"net/url"
	"os"
	"path/filepath"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"

	"github.com/bisoncraft/go-electrum-client/electrumx"
	"github.com/bisoncraft/go-electrum-client/wallet"
)

const (
	appName      = "goele"
	DbTypeBolt   = "bbolt"
	DbTypeSqlite = "sqlite"
)

type ClientConfig struct {
	// Coin ticker to id the coin
	Coin string

	// bip44 mainnet
	CoinType wallet.CoinType

	// Net type - mainnet, testnet or regtest
	NetType string

	// Network parameters - make more general if it cannot adapt to other coins.
	Params *chaincfg.Params

	// Location of the data directory
	DataDir string

	// We use this electrumX server to bootstrap others so it should
	// be set.
	TrustedPeer *electrumx.NodeServerAddr

	// A localhost socks5 proxy port can be set here and will be used as to proxy
	// ElectrumX server onion connections.
	//
	// If not "" setting this enables goele to connect to a limited number of
	// onion servers. Default is "".
	//
	// You should have already set up a localhost socks5 proxy and tor service.
	//
	// Note: This is tested on Linux and is still considered *Experimental*
	ProxyPort string

	// Store the seed in encrypted storage - default false
	StoreEncSeed bool

	// Database implementation type (bbolt or sqlite)
	DbType string

	// An implementation of the Datastore interface
	DB wallet.Datastore

	// The default fee-per-byte for each level
	LowFee    int64
	MediumFee int64
	HighFee   int64

	// The highest allowable fee-per-byte
	MaxFee int64

	// External API to query to look up fees. If this field is nil then the
	// default fees will be used. If the API is unreachable then the default
	// fees will likewise be used. If the API returns a fee greater than MaxFee
	// then the MaxFee will be used instead.
	FeeAPI url.URL

	// If not testing do not overwrite existing wallet files
	Testing bool

	// Test RPC server
	RPCTestPort int
}

func NewDefaultConfig() *ClientConfig {
	return &ClientConfig{
		Coin:        "btc",
		CoinType:    wallet.Bitcoin,
		NetType:     "mainnet",
		Params:      &chaincfg.MainNetParams,
		ProxyPort:   "",
		DataDir:     btcutil.AppDataDir(appName, false),
		DbType:      DbTypeBolt,
		DB:          nil, // concrete impl
		RPCTestPort: 8887,
	}
}
func (cc *ClientConfig) MakeWalletConfig() *wallet.WalletConfig {
	wc := wallet.WalletConfig{
		Coin:         cc.Coin,
		CoinType:     cc.CoinType,
		NetType:      cc.NetType,
		Params:       cc.Params,
		StoreEncSeed: cc.StoreEncSeed,
		DataDir:      cc.DataDir,
		DbType:       cc.DbType,
		DB:           cc.DB,
		LowFee:       cc.LowFee,
		MediumFee:    cc.MediumFee,
		HighFee:      cc.HighFee,
		MaxFee:       cc.MaxFee,
		Testing:      cc.Testing,
	}
	return &wc
}

func (cc *ClientConfig) MakeElectrumXConfig() *electrumx.ElectrumXConfig {
	ex := electrumx.ElectrumXConfig{
		NetType:     cc.NetType,
		Params:      cc.Params, // only genesis .. TODO: remove
		DataDir:     cc.DataDir,
		TrustedPeer: cc.TrustedPeer,
		ProxyPort:   cc.ProxyPort,
		Testing:     cc.Testing,
	}
	return &ex
}

func GetConfigPath() (string, error) {
	userCfgDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appPath := filepath.Join(userCfgDir, appName)
	err = os.MkdirAll(appPath, os.ModeDir|0777)
	if err != nil {
		return "", err
	}
	return appPath, nil
}
