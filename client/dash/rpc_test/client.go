package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cast"
)

// ////////////////////////////////////////////////
// Very simple rpc client. Everything is a string!
// ////////////////////////////////////////////////
func usage() {
	fmt.Println("rpctest v0.1.0")
	fmt.Println()
	fmt.Println("usage:")
	fmt.Println("  client <--network> cmd [positional args]")
	fmt.Println("")
	fmt.Println("  help", "\t\t\t\t\t This help")
	fmt.Println("  echo <any>", "\t\t\t\t Echo any input args - test only")
	fmt.Println("  tip", "\t\t\t\t\t Get blockchain tip")
	fmt.Println("  getbalance", "\t\t\t\t Get wallet confirmed & unconfirmed balance")
	fmt.Println("  listunspent", "\t\t\t\t List all wallet utxos")
	fmt.Println("  getunusedaddress", "\t\t\t Get a new unused wallet receive address")
	fmt.Println("  getchangeaddress", "\t\t\t Get a new unused wallet change address")
	fmt.Println("  spend pw amount address feeType", "\t Make signed transaction from wallet utxos")
	fmt.Println("  broadcast rawTx", "\t\t\t Broadcast rawTx to ElectrumX")
	fmt.Println()
	fmt.Println("Example: client --regtest tip")
	fmt.Println()
	fmt.Println("-------------------------------------------------------------")
	fmt.Println()
}

type cmd struct {
	cmd  string
	args []string
}

func (c *cmd) String() string {
	var a string = ""
	if len(c.args) > 0 {
		a = strings.Join(c.args, " ")

	}
	return fmt.Sprintf("%s    %s\n", c.cmd, a)
}

// echo
func (c *cmd) echo(client *rpc.Client) {
	var request = make(map[string]string)
	for _, a := range c.args {
		request[a] = a
	}
	var response = make(map[string]string)
	err := client.Call("Ec.RPCEcho", &request, &response)
	if err != nil {
		log.Fatal("Ec.RPCEcho:", err)
	}
	fmt.Printf("client response %v\n", response)
}

// tip
func (c *cmd) tip(client *rpc.Client) {
	var request = make(map[string]string)
	var response = make(map[string]string)
	err := client.Call("Ec.RPCTip", &request, &response)
	if err != nil {
		log.Fatal("Ec.RPCTip:", err)
	}
	// fmt.Printf("client response %v\n", response)
	tip := cast.ToString(response["tip"])
	fmt.Println("tip", tip)
}

// getbalance
func (c *cmd) getbalance(client *rpc.Client) {
	var request = make(map[string]string)
	var response = make(map[string]string)
	err := client.Call("Ec.RPCBalance", &request, &response)
	if err != nil {
		log.Fatal("Ec.RPCBalance:", err)
	}
	// fmt.Printf("client response %v\n", response)
	confirmed := cast.ToString(response["confirmed"])
	unconfirmed := cast.ToString(response["unconfirmed"])
	locked := cast.ToString(response["locked"])
	fmt.Println("confirmed", confirmed)
	fmt.Println("unconfirmed", unconfirmed)
	fmt.Println("locked", locked)
}

// listunspent
func (c *cmd) listunspent(client *rpc.Client) {
	var request = make(map[string]string)
	var response = make(map[string]string)
	err := client.Call("Ec.RPCListUnspent", &request, &response)
	if err != nil {
		log.Fatal("Ec.RPCListUnspent:", err)
	}
	// fmt.Printf("client response %v\n", response)
	allUnspents := cast.ToString(response["unspents"])
	if len(allUnspents) == 0 { // zero length string
		fmt.Println("[]")
		return
	}
	unspents := strings.Split(allUnspents, "\n")
	var us []string
	fmt.Println("[")
	for _, unspent := range unspents {
		us = strings.Split(unspent, ":")
		fmt.Println(" {")
		fmt.Println("   txid:", us[0])
		fmt.Println("   vout:", us[1])
		fmt.Println("   value:", us[2])
		fmt.Println("   spendheight:", us[3])
		fmt.Println("   script:", us[4])
		fmt.Println("   watch only:", us[5])
		fmt.Println("   frozen:", us[6])
		fmt.Println(" }")
	}
	fmt.Println("]")
}

// getunusedaddress
func (c *cmd) getunusedaddress(client *rpc.Client) {
	var request = make(map[string]string)
	var response = make(map[string]string)
	err := client.Call("Ec.RPCUnusedAddress", &request, &response)
	if err != nil {
		log.Fatal("Ec.RPCUnusedAddress:", err)
	}
	// fmt.Printf("client response %v\n", response)
	address := cast.ToString(response["address"])
	fmt.Println("address", address)
}

// getchangeaddress
func (c *cmd) getchangeaddress(client *rpc.Client) {
	var request = make(map[string]string)
	var response = make(map[string]string)
	err := client.Call("Ec.RPCChangeAddress", &request, &response)
	if err != nil {
		log.Fatal("Ec.RPCChangeAddress:", err)
	}
	// fmt.Printf("client response %v\n", response)
	address := cast.ToString(response["address"])
	fmt.Println("address", address)
}

// spend
func (c *cmd) spend(client *rpc.Client) {
	var request = make(map[string]string)
	request["pw"] = c.args[0]
	request["amount"] = c.args[1]
	request["address"] = c.args[2]
	request["feeType"] = c.args[3]
	var response = make(map[string]string)
	err := client.Call("Ec.RPCSpend", &request, &response)
	if err != nil {
		log.Fatal("Ec.RPCSpend:", err)
	}
	// fmt.Printf("\nclient response %v\n", response)
	changeIndex := cast.ToString(response["changeIndex"])
	fmt.Println("changeIndex", changeIndex)
	tx := cast.ToString(response["tx"])
	fmt.Println("tx", tx)
	txid := cast.ToString(response["txid"])
	fmt.Println("txid", txid)
}

// broadcast
func (c *cmd) broadcast(client *rpc.Client) {
	var request = make(map[string]string)
	request["rawTx"] = c.args[0]
	var response = make(map[string]string)
	err := client.Call("Ec.RPCBroadcast", &request, &response)
	if err != nil {
		log.Fatal("Ec.RPCBroadcast:", err)
	}
	// fmt.Printf("client response %v\n", response)
	txid := cast.ToString(response["txid"])
	fmt.Println("txid", txid)
}

func main() {
	var port = "" // no default
	args := os.Args
	if len(args) > 2 {
		switch args[1] {
		case "-regtest", "--regtest":
			port = "28887"
		case "-testnet", "--testnet":
			port = "18887"
		case "-mainnet", "--mainnet":
			port = "8887"
		default:
			fmt.Println("please set a network: -mainnet, -testnet, -regtest")
			os.Exit(1)
		}
	}

	args = args[1:]
	if len(args) < 2 {
		usage()
		log.Fatal("no command given")
	}

	var c = cmd{
		args: make([]string, 0),
	}

	for i, a := range args {
		if i == 0 {
			continue
		}
		if i == 1 {
			c.cmd = a
			continue
		}
		c.args = append(c.args, a)
	}
	fmt.Println(c.String())
	if c.cmd == "help" {
		usage()
		os.Exit(0)
	}

	switch c.cmd {
	case "tip", "listunspent", "getunusedaddress", "getchangeaddress", "getbalance":
	// no params
	case "echo":
	// any number of params
	case "spend":
		// 4 params, others ignored
		if len(c.args) < 4 {
			usage()
			log.Fatal(c.String(), "needs 4 arguments: pw amount address feeType")
		}
		if len(c.args[0]) == 0 {
			usage()
			log.Fatal(c.String(), "empty password")
		}
		i, err := strconv.Atoi(c.args[1])
		if err != nil {
			usage()
			log.Fatal(c.String(), "amount should be a number in satoshis")
		}
		if i < 10000 {
			usage()
			log.Fatal(c.String(), i, "amount is dust")
		}
		if len(c.args[2]) == 0 {
			usage()
			log.Fatal(c.String(), "address should be a bitcoin address")
		}
		switch c.args[3] {
		case "NORMAL", "PRIORITY", "ECONOMIC":
		default:
			usage()
			log.Fatal(c.String(), "feeType should be NORMAL, PRIORITY or ECONOMIC")
		}
	case "broadcast":
		// 1 param, others ignored
		if len(c.args) < 1 {
			usage()
			log.Fatal(c.String(), "needs 1 argument: the raw tx")
		}
	default:
		usage()
		log.Fatal(c.String(), "unknown command")
	}

	client, err := rpc.DialHTTP("tcp", net.JoinHostPort("127.0.0.1", port))
	if err != nil {
		log.Fatal("dialing:", err)
	}

	switch c.cmd {
	case "echo":
		c.echo(client)
	case "tip":
		c.tip(client)
	case "getbalance":
		c.getbalance(client)
	case "listunspent":
		c.listunspent(client)
	case "getunusedaddress":
		c.getunusedaddress(client)
	case "getchangeaddress":
		c.getchangeaddress(client)
	case "broadcast":
		c.broadcast(client)
	case "spend":
		c.spend(client)
	}
}
