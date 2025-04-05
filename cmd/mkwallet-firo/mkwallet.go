package main

// Run create or recreate a wallet for testing

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bisoncraft/go-electrum-client/client"
	"github.com/bisoncraft/go-electrum-client/client/firo"
	"github.com/bisoncraft/go-electrum-client/electrumx"
	"github.com/bisoncraft/go-electrum-client/wallet"
	"github.com/btcsuite/btcd/chaincfg"
)

var (
	coins = []string{"firo"} // add as implemented
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
	case "firo":
		cfg.CoinType = wallet.Firo
		cfg.Coin = coin
		switch net {
		case "simnet", "regtest":
			cfg.CoinType = 136
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
			cfg.CoinType = 136
			cfg.NetType = electrumx.Testnet
			cfg.RPCTestPort = 18887
			cfg.Params = &chaincfg.TestNet3Params
			cfg.TrustedPeer = &electrumx.NodeServerAddr{
				Net: "ssl", Addr: "95.179.164.13:51002",
			}
			cfg.StoreEncSeed = true
			cfg.Testing = true
			fmt.Println(net)
		case "mainnet":
			cfg.CoinType = 136
			cfg.Params = &chaincfg.MainNetParams
			cfg.NetType = electrumx.Mainnet
			cfg.RPCTestPort = 8887
			cfg.TrustedPeer = &electrumx.NodeServerAddr{
				// Net: "ssl", Addr: "electrumx.firo.org:50002",
				// Net: "ssl", Addr: "electrumx01.firo.org:50002",
				Net: "ssl", Addr: "electrumx02.firo.org:50002",
				// Net: "ssl", Addr: "electrumx03.firo.org:50002",
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

func configure() (string, string, string, *client.ClientConfig, error) {
	help := flag.Bool("help", false, "usage help")
	coin := flag.String("coin", "firo", "coin name")
	net := flag.String("net", "regtest", "network type; testnet, mainnet, regtest")
	pass := flag.String("pass", "", "wallet password")
	action := flag.String("action", "create", "action: 'create'a new wallet or 'recreate' from seed")
	seed := flag.String("seed", "", "'seed words for recreate' inside ''; example: 'word1 word2 ... word12'")
	test_wallet := flag.Bool("tw", false, "known test wallets override for regtest/testnet")
	dbType := flag.String("dbtype", "bbolt", "set database type: 'bbolt' default, 'sqlite'")

	flag.Parse()
	if *help {
		flag.Usage()
		os.Exit(0)
	}
	fmt.Println("coin:", *coin)
	fmt.Println("net:", *net)
	fmt.Println("action:", *action)
	fmt.Println("pass:", *pass)
	fmt.Println("seed:", *seed)
	fmt.Println("test_wallet:", *test_wallet)
	fmt.Println("dbtype:", *dbType)
	if *test_wallet {
		switch *net {
		case "regtest", "simnet":
			*seed = "jungle pair grass super coral bubble tomato sheriff pulp cancel luggage wagon"
		case "testnet", "testnet3":
			*seed = "canyon trip truly ritual lonely quiz romance rose alone journey like bronze"
		default:
			return "", "", "", nil, errors.New("no test_wallet for mainnet")
		}
	}
	if *action == "create" && *pass == "" {
		return "", "", "", nil, errors.New("wallet create needs a password")
	} else if *action == "recreate" {
		if *pass == "" {
			return "", "", "", nil, errors.New("wallet recreate needs a new password - " +
				"can be different to the previous password")
		}
		if *seed == "" {
			return "", "", "", nil, errors.New("wallet recreate needs the old wallet seed")
		}
		words := strings.SplitN(*seed, " ", 12)
		fmt.Printf("%q (len %d)\n", words, len(words))
		if len(words) != 12 {
			return "", "", "", nil, errors.New("a seed must have 12 words each separated by a space")
		}
		var bad bool
		for _, word := range words {
			if len(word) < 3 {
				fmt.Printf("bad word: '%s'\n", word)
				bad = true
			}
		}
		if bad {
			return "", "", "", nil, errors.New("malformed seed -- did you put extra spaces?")
		}
	}
	cfg, err := makeBasicConfig(*coin, *net)
	if *dbType == "sqlite" {
		cfg.DbType = "sqlite"
	}
	return *action, *pass, *seed, cfg, err
}

func checkSimnetHelp(cfg *client.ClientConfig) string {
	var help string
	switch cfg.Params {
	case &chaincfg.RegressionNetParams:
		help = "check out simnet harness scripts at client/btc/test_harness\n" +
			"README.md, src_harness.sh & ex.sh\n" +
			"Then when goele starts navigate to client/btc/rpctest and use the\n" +
			"minimalist rpc test client"
	default:
		help = "is ElectrumX server up and running?"
	}
	return help
}

func main() {
	fmt.Println("Goele mkwallet", client.GoeleVersion)
	action, pass, seed, cfg, err := configure()
	fmt.Println(action, pass, seed)
	if err != nil {
		fmt.Println(err, " - exiting")
		flag.Usage()
		os.Exit(1)

	}
	// make basic client
	ec := firo.NewFiroElectrumClient(cfg)

	if action == "create" {
		err := ec.CreateWallet(pass)
		if err != nil {
			fmt.Println(err)
		}
		os.Exit(0)
	}

	// start client, create node & sync headers
	err = ec.Start(context.Background())
	if err != nil {
		fmt.Printf("%v - exiting.\n%s\n", err, checkSimnetHelp(cfg))
		os.Exit(1)
	}

	// recreate the client's wallet
	// for non-mainnet testing recreate a wallet with a known set of keys if -tw ..
	err = ec.RecreateWallet(context.TODO(), pass, seed)
	if err != nil {
		fmt.Println(err, " - exiting")
	}
}
