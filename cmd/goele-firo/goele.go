package main

// Run goele as an app for testing

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

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
		cfg.CoinType = wallet.Bitcoin
		cfg.Coin = coin
		switch net {
		case "simnet", "regtest":
			cfg.CoinType = 136
			cfg.NetType = electrumx.Regtest
			cfg.RPCTestPort = 28886
			cfg.Params = &chaincfg.RegressionNetParams
			cfg.TrustedPeer = &electrumx.NodeServerAddr{
				Net: "ssl", Addr: "127.0.0.1:50002",
			}
			cfg.StoreEncSeed = true
			cfg.Testing = true
		case "testnet", "testnet3", "testnet4":
			cfg.CoinType = 136
			cfg.NetType = electrumx.Testnet
			cfg.RPCTestPort = 18886
			cfg.Params = &chaincfg.TestNet3Params
			cfg.TrustedPeer = &electrumx.NodeServerAddr{
				Net: "ssl", Addr: "95.179.164.13:51002",
			}
			cfg.StoreEncSeed = true
			cfg.Testing = true
		case "mainnet":
			cfg.CoinType = 136
			cfg.Params = &chaincfg.MainNetParams
			cfg.NetType = electrumx.Mainnet
			cfg.RPCTestPort = 8886
			cfg.TrustedPeer = &electrumx.NodeServerAddr{
				Net: "ssl", Addr: "electrumx.firo.org:50002",
				// Net: "ssl", Addr: "electrumx01.firo.org:50002",
				// Net: "ssl", Addr: "electrumx02.firo.org:50002",
				// Net: "ssl", Addr: "electrumx03.firo.org:50002",
			}
			cfg.StoreEncSeed = false
			cfg.Testing = false
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

func configure() (string, *client.ClientConfig, error) {
	coin := flag.String("coin", "firo", "coin name")
	net := flag.String("net", "regtest", "network type; testnet, mainnet, regtest")
	pass := flag.String("pass", "", "wallet password")
	flag.Parse()
	cfg, err := makeBasicConfig(*coin, *net)
	return *pass, cfg, err
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
	// // Start profiling
	// f, e := os.Create("goele-firo.prof")
	// if e != nil {
	// 	fmt.Println(e)
	// 	return
	// }
	// pprof.StartCPUProfile(f)
	// defer pprof.StopCPUProfile()
	// //--------------------------

	// Run your program here
	fmt.Println("Goele-Firo", client.GoeleVersion)
	pass, cfg, err := configure()
	if err != nil {
		fmt.Println(err, " - exiting")
		os.Exit(1)
	}
	fmt.Println(cfg.Coin)

	net := cfg.NetType
	fmt.Println(net)

	if net == electrumx.Regtest {
		// Use SqLite3 for regtest as you can debug the wallet while running.
		// BoltDb for production but only one instance of the bdb can be run
		// concurrently on the same machine.
		cfg.DbType = "sqlite"
	}
	fmt.Println(cfg.DbType)

	fmt.Printf("electrumX server address: %s\n", cfg.TrustedPeer)

	// make basic client
	ec := firo.NewFiroElectrumClient(cfg)

	// start client, create ElectrumXInterface & sync headers
	clientCtx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	err = ec.Start(clientCtx)
	if err != nil {
		fmt.Printf("%v - exiting.\n%s\n", err, checkSimnetHelp(cfg))
		os.Exit(1)
	}
	fmt.Println("synced", ec.Synced())

	// to make the client's wallet:
	// - for regtest/testnet testing recreate a wallet with a known set of keys.
	// - use the mkwallet and rmwallet tools to create, recreate a wallet at the
	//   configured location
	// - the rmwallet tool removes a wallet from the configured location.
	//   regtest & testnet *only*

	switch cfg.NetType {
	case electrumx.Regtest:
		// mnemonic := "jungle pair grass super coral bubble tomato sheriff pulp cancel luggage wagon"
		// err := ec.RecreateWallet("abc", mnemonic)
		err := ec.LoadWallet("abc")
		if err != nil {
			fmt.Println(err, " - exiting")
			os.Exit(1)
		}
	case electrumx.Testnet:
		// mnemonic := "canyon trip truly ritual lonely quiz romance rose alone journey like bronze"
		// err := ec.RecreateWallet("abc", mnemonic)
		ec.LoadWallet("abc")
		if err != nil {
			fmt.Println(err, " - exiting")
			os.Exit(1)
		}
	case electrumx.Mainnet:
		// production usage: load the client's wallet - needs -pass param
		err := ec.LoadWallet(pass)
		if err != nil {
			fmt.Println(err, " - exiting")
			os.Exit(1)
		}
	default:
		fmt.Printf("unknown net %s - exiting\n", net)
		os.Exit(1)
	}

	// Set up Notify for all our already given out receive addresses (getunusedaddress)
	// and broadcasted change addresses in order to receive any changes to the state of
	// the address history back from electrumx
	err = ec.SyncWallet(clientCtx)
	if err != nil {
		fmt.Println(err, " - exiting")
		os.Exit(1)
	}

	fmt.Println("")
	fmt.Println("       -------------")
	fmt.Println("       Ctl-c to exit")
	fmt.Println("       -------------")
	fmt.Println("")

	// for testing only
	err = firo.RPCServe(ec)
	if err != nil {
		fmt.Println(err, " - exiting")
		os.Exit(1)
	}

	// SIGINT kills the node server(s) & test rpc server
	time.Sleep(time.Second)
}
