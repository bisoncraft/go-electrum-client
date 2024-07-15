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

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/dev-warrior777/go-electrum-client/client"
	"github.com/dev-warrior777/go-electrum-client/client/btc"
	"github.com/dev-warrior777/go-electrum-client/electrumx"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

var (
	coins = []string{"btc"} // add as implemented
	nets  = []string{"mainnet", "testnet", "regtest", "simnet"}
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
	if !contains(coins, coin) {
		return nil, errors.New("invalid coin")
	}
	if !contains(nets, net) {
		return nil, errors.New("invalid net")
	}
	switch coin {
	case "btc":
	default:
		return nil, errors.New("invalid coin")
	}
	cfg := client.NewDefaultConfig()
	cfg.CoinType = wallet.Bitcoin
	cfg.Coin = coin
	cfg.NetType = net
	cfg.StoreEncSeed = true
	appDir, err := client.GetConfigPath()
	if err != nil {
		return nil, err
	}
	coinNetDir := filepath.Join(appDir, coin, net)
	err = os.MkdirAll(coinNetDir, os.ModeDir|0777)
	if err != nil {
		return nil, err
	}
	cfg.DataDir = coinNetDir
	switch net {
	case "regtest", "simnet":
		cfg.Params = &chaincfg.RegressionNetParams
		cfg.TrustedPeer = &electrumx.NodeServerAddr{
			// Net: "ssl", Addr: "127.0.0.1:57002", // debug server
			Net: "ssl", Addr: "127.0.0.1:53002", // harness server
		}
		cfg.StoreEncSeed = true
		cfg.RPCTestPort = 28887
		cfg.Testing = true
	case "testnet":
		cfg.Params = &chaincfg.TestNet3Params
		cfg.ProxyPort = "9050"
		cfg.TrustedPeer = &electrumx.NodeServerAddr{
			// Net: "ssl", Addr: "testnet.aranguren.org:51002", // Fulcrum - ExpBug0

			// Net:   "ssl", // Fulcrum - ExpBug0
			// Addr:  "gsw6sn27quwf6u3swgra6o7lrp5qau6kt3ymuyoxgkth6wntzm2bjwyd.onion:51002", // Fulcrum - ExpBug0
			// Onion: true,

			Net:   "tcp", // electrs-esplora 0.4.1
			Addr:  "explorerzydxu5ecjrkwceayqybizmpjjznk5izmitf2modhcusuqlid.onion:143",
			Onion: true,

			// Net: "ssl", Addr: "electrum.blockstream.info:60002", // no verbose gtx
			// Net: "ssl", Addr: "testnet.qtornado.com:51002", // doesn't sends same 3 peers or none .. suspect hanky panky
			// Net: "tcp", Addr: "testnet.qtornado.com:51001", // suspect hanky panky
		}
		cfg.StoreEncSeed = true
		cfg.RPCTestPort = 18887
		cfg.Testing = true
	case "mainnet":
		cfg.Params = &chaincfg.MainNetParams
		cfg.ProxyPort = "9050"
		cfg.TrustedPeer = &electrumx.NodeServerAddr{
			Net: "ssl", Addr: "[2a01:4f9:c010:e9d3::1]:50002",
			// Net: "ssl", Addr: "elx.bitske.com:50002",
		}
		cfg.RPCTestPort = 8887
		cfg.StoreEncSeed = false
		cfg.Testing = false
	}
	return cfg, nil
}

func configure() (string, *client.ClientConfig, error) {
	coin := flag.String("coin", "btc", "coin name")
	net := flag.String("net", "regtest", "network type; testnet, mainnet, regtest")
	pass := flag.String("pass", "", "wallet password")
	flag.Parse()
	fmt.Println("coin:", *coin)
	fmt.Println("net:", *net)
	cfg, err := makeBasicConfig(*coin, *net)
	fmt.Printf("electrumX server address: %s\n", cfg.TrustedPeer)
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
	// f, e := os.Create("goele.prof")
	// if e != nil {
	// 	fmt.Println(e)
	// 	return
	// }
	// pprof.StartCPUProfile(f)
	// defer pprof.StopCPUProfile()
	// //--------------------------

	// Run your program here
	fmt.Println("Goele", client.GoeleVersion)
	pass, cfg, err := configure()
	if err != nil {
		fmt.Println(err, " - exiting")
		os.Exit(1)
	}
	net := cfg.Params.Name
	fmt.Println(net)

	cfg.DbType = "sqlite"
	fmt.Println(cfg.DbType)

	// make basic client
	ec := btc.NewBtcElectrumClient(cfg)

	// start client, create node & sync headers
	clientCtx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	err = ec.Start(clientCtx)
	if err != nil {
		fmt.Printf("%v - exiting.\n%s\n", err, checkSimnetHelp(cfg))
		os.Exit(1)
	}

	feeRate, _ := ec.FeeRate(clientCtx, 6)
	fmt.Println("Fee rate: ", feeRate)

	// to make the client's wallet:
	// - for regtest/testnet testing recreate a wallet with a known set of keys.
	// - use the mkwallet and rmwallet tools to create, recreate a wallet at the
	//   configured location
	// - the rmwallet tool removes a wallet from the configured location.
	//   regtest & testnet *only*

	switch net {
	case "regtest":
		// mnemonic := "jungle pair grass super coral bubble tomato sheriff pulp cancel luggage wagon"
		// err := ec.RecreateWallet("abc", mnemonic)
		err := ec.LoadWallet("abc")
		if err != nil {
			fmt.Println(err, " - exiting")
			os.Exit(1)
		}
	case "testnet3":
		// mnemonic := "canyon trip truly ritual lonely quiz romance rose alone journey like bronze"
		// err := ec.RecreateWallet("abc", mnemonic)
		ec.LoadWallet("abc")
		if err != nil {
			fmt.Println(err, " - exiting")
			os.Exit(1)
		}
	case "mainnet":
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
	err = btc.RPCServe(ec)
	if err != nil {
		fmt.Println(err, " - exiting")
		os.Exit(1)
	}

	// SIGINT kills the node server(s) & test rpc server
	time.Sleep(time.Second)
}
