package btc

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/spf13/cast"
)

// RPC Server For testing only. Goele is golang code iintended to be used
// directly by other golang projects; for example a lite trading wallet.

// RPC Service methods
type Ec struct {
	EleClient *BtcElectrumClient
}

// Simple echo back to client method
func (e *Ec) RPCEcho(request map[string]string, response *map[string]string) error {
	r := *response
	for k, v := range request {
		r[k] = v
	}
	return nil
}

func (e *Ec) Tip() (int64, bool) {
	h := e.EleClient.clientHeaders
	return h.hdrsTip, h.synced
}
func (e *Ec) RPCTip(request map[string]string, response *map[string]string) error {
	r := *response
	t, s := e.Tip()
	tip := cast.ToString(t)
	synced := cast.ToString(s)
	r["tip"] = tip
	r["synced"] = synced
	return nil
}

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

// /////////////////////////////////////////////
// RPC Server
// ///////////
func (ec *BtcElectrumClient) RPCServe() error {
	bind_addr := "127.0.0.1:8888"
	addr, err := net.ResolveTCPAddr("tcp", bind_addr)
	if err != nil {
		return err
	}
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}
	// register Ec methods with correct signature
	// Method(request map[string]string, response *map[string]string) error
	rpcservice := &Ec{
		EleClient: ec,
	}
	err = rpc.Register(rpcservice)
	if err != nil {
		return err
	}
	// set up rpc handlers
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
		fmt.Println("rpc channel closed")
	}()

	fmt.Println("rpc http server Serve() start")
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

/////////////////////////////////////////
// Old
// ///
// s.Register("listunspent", func(ctx context.Context, params jsonrpc.Params) (jsonrpc.Result, error) {

// 	utxos, err := ec.ListUnspent()

// 	if err != nil {
// 		return jsonrpc.Result{
// 			"unspents": "",
// 		}, err
// 	}

// 	var sb strings.Builder
// 	var last = len(utxos) - 1
// 	for i, utxo := range utxos {
// 		sb.WriteString(utxo.Op.String())
// 		sb.WriteString(":")
// 		sb.WriteString(strconv.Itoa(int(utxo.Value)))
// 		sb.WriteString(":")
// 		sb.WriteString(strconv.Itoa(int(utxo.AtHeight)))
// 		sb.WriteString(":")
// 		sb.WriteString(hex.EncodeToString(utxo.ScriptPubkey))
// 		sb.WriteString(":")
// 		sb.WriteString(strconv.FormatBool(utxo.WatchOnly))
// 		sb.WriteString(":")
// 		sb.WriteString(strconv.FormatBool(utxo.Frozen))
// 		if i != last {
// 			sb.WriteString("\n")
// 		}
// 	}

// 	return jsonrpc.Result{
// 		"unspents": sb.String(),
// 	}, nil
// })

// s.Register("spend", func(ctx context.Context, params jsonrpc.Params) (jsonrpc.Result, error) {
// 	logger.Info("params: %v", params)

// 	amt := cast.ToInt64(params.Get("amount"))
// 	addr := cast.ToString(params.Get("address"))
// 	feeType := cast.ToString(params.Get("feeType"))
// 	var feeLvl wallet.FeeLevel
// 	switch feeType {
// 	case "PRIORITY":
// 		feeLvl = wallet.PRIORITY
// 	case "NORMAL":
// 		feeLvl = wallet.NORMAL
// 	case "ECONOMIC":
// 		feeLvl = wallet.ECONOMIC
// 	default:
// 		feeLvl = wallet.NORMAL
// 	}

// 	// tx, txid, err := ec.Spend(amt, addr, feeLvl, true)
// 	tx, txid, err := ec.Spend(amt, addr, feeLvl, false)

// 	if err != nil {
// 		return jsonrpc.Result{
// 			"tx":   "",
// 			"txid": "",
// 		}, err
// 	}

// 	return jsonrpc.Result{
// 		"tx":   tx,
// 		"txid": txid,
// 	}, nil
// })

// s.Register("broadcast", func(ctx context.Context, params jsonrpc.Params) (jsonrpc.Result, error) {
// 	logger.Info("params: %v", params)

// 	rawTx := cast.ToString(params.Get("rawTx"))

// 	txid, err := ec.Broadcast(rawTx)

// 	if err != nil {
// 		fmt.Printf("Broadcast error: %v\n", err.Error())
// 		return jsonrpc.Result{
// 			"txid": "",
// 		}, err
// 	}

// 	return jsonrpc.Result{
// 		"txid": txid,
// 	}, nil
// })
