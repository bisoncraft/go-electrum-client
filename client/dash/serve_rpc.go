package dash

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/bisoncraft/go-electrum-client/client"
	"github.com/bisoncraft/go-electrum-client/wallet"
	"github.com/spf13/cast"
)

// RPC Server For testing only. Not production.

// RPC Service methods
type Ec struct {
	EleClient *DashElectrumClient
}

// Simple echo back to client method
func (e *Ec) RPCEcho(request map[string]string, response *map[string]string) error {
	r := *response
	for k, v := range request {
		r[k] = v
	}
	return nil
}

// Get the blockchain tip
func (e *Ec) Tip() int64 {
	return e.EleClient.Tip()
}

func (e *Ec) RPCTip(request map[string]string, response *map[string]string) error {
	r := *response
	t := e.Tip()
	tip := cast.ToString(t)
	r["tip"] = tip
	return nil
}

func (e *Ec) RPCBalance(request map[string]string, response *map[string]string) error {
	r := *response
	c, u, l, err := e.EleClient.Balance()
	if err != nil {
		return err
	}
	confirmed := cast.ToString(c)
	unconfirmed := cast.ToString(u)
	locked := cast.ToString(l)
	r["confirmed"] = confirmed
	r["unconfirmed"] = unconfirmed
	r["locked"] = locked
	return nil
}

// List unspent outputs in the wallet including frozen utxos
func (e *Ec) ListUnspent() (string, error) {
	utxos, err := e.EleClient.ListUnspent()
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	var last = len(utxos) - 1
	for i, utxo := range utxos {
		sb.WriteString(utxo.Op.String())
		sb.WriteString(":")
		sb.WriteString(strconv.Itoa(int(utxo.Value)))
		sb.WriteString(":")
		sb.WriteString(strconv.Itoa(int(utxo.AtHeight)))
		sb.WriteString(":")
		sb.WriteString(hex.EncodeToString(utxo.ScriptPubkey))
		sb.WriteString(":")
		sb.WriteString(strconv.FormatBool(utxo.WatchOnly))
		sb.WriteString(":")
		sb.WriteString(strconv.FormatBool(utxo.Frozen))
		if i != last {
			sb.WriteString("\n")
		}
	}
	return sb.String(), nil
}
func (e *Ec) RPCListUnspent(request map[string]string, response *map[string]string) error {
	r := *response
	unspents, err := e.ListUnspent()
	if err != nil {
		return err
	}
	r["unspents"] = unspents
	return nil
}

// Get a new unused wallet receive address
func (e *Ec) RPCUnusedAddress(request map[string]string, response *map[string]string) error {
	r := *response
	address, err := e.EleClient.UnusedAddress(context.TODO())
	if err != nil {
		return err
	}
	r["address"] = address
	return nil
}

// Get a new unused wallet change address
func (e *Ec) RPCChangeAddress(request map[string]string, response *map[string]string) error {
	r := *response
	address, err := e.EleClient.ChangeAddress(context.TODO())
	if err != nil {
		return err
	}
	r["address"] = address
	return nil
}

func (e *Ec) RPCSpend(request map[string]string, response *map[string]string) error {
	r := *response
	pw := cast.ToString(request["pw"])
	amt := cast.ToInt64(request["amount"])
	addr := cast.ToString(request["address"])
	feeType := cast.ToString(request["feeType"])
	var feeLvl wallet.FeeLevel
	switch feeType {
	case "PRIORITY":
		feeLvl = wallet.PRIORITY
	case "NORMAL":
		feeLvl = wallet.NORMAL
	case "ECONOMIC":
		feeLvl = wallet.ECONOMIC
	default:
		feeLvl = wallet.NORMAL
	}

	changeIndex, tx, txid, err := e.EleClient.Spend(pw, amt, addr, feeLvl)
	if err != nil {
		return err
	}
	r["tx"] = tx
	r["txid"] = txid
	r["changeIndex"] = cast.ToString(changeIndex)
	return nil
}

func (e *Ec) RPCBroadcast(request map[string]string, response *map[string]string) error {
	r := *response
	rawTx := cast.ToString(request["rawTx"])
	if len(rawTx) > 27 {
		fmt.Println("rpc:", rawTx[:27])
	}
	txid, err := e.EleClient.RpcBroadcast(context.TODO(), rawTx)
	fmt.Println("rpc err:", err)
	if err != nil {
		return err
	}
	r["txid"] = txid
	return nil
}

// /////////////////////////////////////////////
// RPC Server
// ///////////
const (
	RpcDefaultIP   = "127.0.0.1"
	RpcDefaultPort = 28887 // regtest
)

func RPCServe(electrumClient client.ElectrumClient) error {
	btcElectrumClient, ok := electrumClient.(*DashElectrumClient)
	if !ok {
		return errors.New("electrumClient is not an instance of DashElectrumClient")
	}
	rpc_ip := RpcDefaultIP
	rpc_port := electrumClient.GetConfig().RPCTestPort
	if rpc_port == 0 {
		rpc_port = RpcDefaultPort
	}
	bind_addr := fmt.Sprintf("%s:%d", rpc_ip, rpc_port)
	addr, err := net.ResolveTCPAddr("tcp", bind_addr)
	if err != nil {
		return err
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}

	// register Ec methods with correct signature:
	//
	// 	func (e *Ec) RPCMethod(request map[string]string, response *map[string]string) error
	//
	rpcservice := &Ec{
		EleClient: btcElectrumClient,
	}
	err = rpc.Register(rpcservice)
	if err != nil {
		return err
	}
	// set up http handlers for rpc
	rpc.HandleHTTP()

	// Http Server
	var srv http.Server
	rpcConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		// "^C"
		if err := srv.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			fmt.Printf("rpc http server Shutdown: %v\n", err)
		}
		close(rpcConnsClosed)
	}()

	fmt.Println("=== rpc http server Serve() start - Ctl-c to stop ===")
	fmt.Println()
	if err := srv.Serve(listener); err != http.ErrServerClosed {
		// error closing listener
		fmt.Printf("rpc http server Serve: %v\n", err)
		fmt.Println("rpc error exit")
		os.Exit(1)
	}

	<-rpcConnsClosed
	fmt.Println("rpc clean exit")
	return nil
}
