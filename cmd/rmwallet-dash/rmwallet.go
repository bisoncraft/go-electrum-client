package main

// Delete a default test wallet - will not work for mainnet config

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/bisoncraft/go-electrum-client/client"
	"github.com/bisoncraft/go-electrum-client/electrumx"
	"github.com/bisoncraft/go-electrum-client/wallet"
	"github.com/btcsuite/btcd/chaincfg"
)

const blockchain_headers = "blockchain_headers"

var (
	coins = []string{"dash"} // add as implemented
	nets  = []string{"mainnet", "testnet", "testnet3", "testnet4", "regtest", "simnet"}
)

func makeBasicConfig(coin, net string) (*client.ClientConfig, error) {
	contains := func(s []string, str string) bool {
		for _, v := range s {
			if v == str {
				return true
			}
		}
		return false
	}

	cfg := client.NewDefaultConfig()
	if !contains(nets, net) {
		return nil, errors.New("invalid net")
	}
	if !contains(coins, coin) {
		return nil, errors.New("invalid coin")
	}

	switch coin {
	case "dash":
		cfg.CoinType = wallet.Bitcoin
		cfg.Coin = coin
		switch net {
		case "simnet", "regtest":
			cfg.NetType = electrumx.Regtest
			cfg.RPCTestPort = 28887
			cfg.Params = &chaincfg.RegressionNetParams
			cfg.TrustedPeer = &electrumx.NodeServerAddr{
				Net: "ssl", Addr: "127.0.0.1:50002",
			}
			cfg.StoreEncSeed = true
			cfg.Testing = true
			fmt.Println(net)
		case "testnet", "testnet3", "testnet4":
			cfg.NetType = electrumx.Testnet
			cfg.RPCTestPort = 18887
			cfg.Params = &chaincfg.TestNet3Params
			cfg.TrustedPeer = &electrumx.NodeServerAddr{
				Net: "ssl", Addr: "dash-electrum-testnet.pshenmic.dev:50002",
			}
			cfg.StoreEncSeed = true
			cfg.Testing = true
			fmt.Println(net)
		case "mainnet":
			cfg.Params = &chaincfg.MainNetParams
			cfg.NetType = electrumx.Mainnet
			cfg.RPCTestPort = 8887
			cfg.TrustedPeer = &electrumx.NodeServerAddr{
				// TODO: CHECK THIS AND ADD OTHERS
				Net: "ssl", Addr: "dash-electrum.pshenmic.dev:50002",
			}
			cfg.StoreEncSeed = false
			cfg.Testing = false
			fmt.Println(net)
		default:
			fmt.Printf("unknown net %s - exiting\n", net)
			flag.Usage()
			os.Exit(1)
		}
	default:
		return nil, errors.New("invalid coin")
	}

	appDir, err := client.GetConfigPath()
	if err != nil {
		return nil, err
	}
	coinNetDir := filepath.Join(appDir, coin, cfg.NetType)
	err = os.MkdirAll(coinNetDir, os.ModeDir|0777)
	if err != nil {
		return nil, err
	}
	cfg.DataDir = coinNetDir
	return cfg, nil
}

func configure() (*client.ClientConfig, error) {
	coin := flag.String("coin", "btc", "coin name")
	net := flag.String("net", "regtest", "network type; testnet, regtest. Mainnet not supported")
	flag.Parse()
	fmt.Println("coin:", *coin)
	fmt.Println("net:", *net)
	cfg, err := makeBasicConfig(*coin, *net)
	return cfg, err
}

func main() {
	fmt.Println("Goele rmwallet", client.GoeleVersion)
	cfg, err := configure()
	if err != nil {
		fmt.Println(err, " - exiting")
		flag.Usage()
		os.Exit(1)
	}
	if cfg.NetType == electrumx.Mainnet {
		fmt.Println("removing mainnet wallets is not supported")
		os.Exit(2)
	}
	type walletDb struct {
		db   string
		name string
	}
	walletFiles := []walletDb{
		{
			db:   "bbolt",
			name: "wallet.bdb",
		},
		{
			db:   "sqlite",
			name: "wallet.db",
		},
	}
	for _, w := range walletFiles {
		wallet := path.Join(cfg.DataDir, w.name)
		fmt.Println(wallet)
		if _, err := os.Stat(wallet); errors.Is(err, os.ErrNotExist) {
			fmt.Println(err)
			continue
		}
		if askForConfirmation("remove?") {
			os.Remove(wallet)
		}
	}

	if cfg.Params == &chaincfg.RegressionNetParams {
		headers := path.Join(cfg.DataDir, blockchain_headers)
		fmt.Println(headers)
		if _, err := os.Stat(headers); errors.Is(err, os.ErrNotExist) {
			fmt.Println(err)
			os.Exit(1)
		}
		if askForConfirmation("remove?") {
			os.Remove(headers)
		}
	}
}

func askForConfirmation(s string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [y/n]: ", s)

		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		}
	}
}
