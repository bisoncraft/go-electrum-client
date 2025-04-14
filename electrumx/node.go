package electrumx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
)

var ErrNotConnected = errors.New("node not connected")

type Node struct {
	serverAddr             string
	netProto               string
	connectOpts            *connectOpts
	server                 *Server
	leader                 bool
	networkHeaders         *headers
	clientTipChangeNotify  chan int64
	clientScriptHashNotify chan *ScripthashStatusResult
	session                *session
}

func newNode(
	netAddr *NodeServerAddr,
	proxyAddr string,
	isLeader bool,
	networkHeaders *headers,
	clientTipChangeNotify chan int64,
	clientScriptHashNotify chan *ScripthashStatusResult) (*Node, error) {

	netProto := netAddr.Network()
	addr := netAddr.String()
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	var tlsConfig *tls.Config
	switch netProto {
	case "ssl":
		rootCAs, _ := x509.SystemCertPool()
		tlsConfig = &tls.Config{
			InsecureSkipVerify: true,
			RootCAs:            rootCAs,
			MinVersion:         tls.VersionTLS12, // works ok
			ServerName:         host,
		}
	case "tcp":
		tlsConfig = nil
	default:
		return nil, fmt.Errorf("unknown protocol: %s", netProto)
	}

	connectOpts := &connectOpts{
		TLSConfig: tlsConfig,
		TorProxy:  proxyAddr,
	}

	n := &Node{
		serverAddr:             addr,
		netProto:               netProto,
		connectOpts:            connectOpts,
		server:                 &Server{},
		leader:                 isLeader,
		networkHeaders:         networkHeaders,
		clientTipChangeNotify:  clientTipChangeNotify,
		clientScriptHashNotify: clientScriptHashNotify,
		session:                nil,
	}
	return n, nil
}

func (n *Node) start(nodeCtx context.Context, nodeCancel context.CancelCauseFunc, network, nettype, genesis string) error {
	// connect to electrumX
	sc, err := connectServer(nodeCtx, nodeCancel, n.serverAddr, n.connectOpts)
	if err != nil {
		return err
	}

	version, err := sc.serverVersion(nodeCtx, "Electrum", "1.4")
	if err != nil {
		return err
	}

	feats, err := sc.features(nodeCtx)
	if err != nil {
		return err
	}
	if feats.Genesis != genesis {
		return fmt.Errorf("wrong genesis hash for %s %s", network, nettype)
	}

	fmt.Printf(
		"** Connected to %s over %s on %s ***\n   Using server software version %s protocol version %s\n   Genesis %s\n",
		n.serverAddr, n.netProto, nettype, version[0], version[1], genesis)

	n.server.conn = sc
	n.server.connected = true
	n.server.nodeCancel = nodeCancel
	n.server.softwareVersion = version[0]
	n.server.protocolVersion = version[1]

	// start a new session for this node to monitor resource use
	n.session = newSession()
	n.session.start(nodeCtx)

	// Node is up and ready - if not leader then we exit here
	if !n.leader {
		return nil
	}

	// leader sync headers
	err = n.syncHeaders(nodeCtx)
	if err != nil {
		return err
	}
	// start header notifications
	err = n.headersNotify(nodeCtx)
	if err != nil {
		return err
	}
	// start scripthash notifications
	return n.scriptHashNotify(nodeCtx)
}

// promoteToLeader makes a non-leader responsible for continuing sync if not
// synced & receiving incoming notifications
func (n *Node) promoteToLeader(nodeCtx context.Context) error {
	h := n.networkHeaders
	// start sync if not synced
	if !h.synced {
		err := n.syncHeaders(nodeCtx)
		if err != nil {
			return err
		}
	}
	// start header notifications
	err := n.headersNotify(nodeCtx)
	if err != nil {
		return err
	}
	// start scripthash notifications
	err = n.scriptHashNotify(nodeCtx)
	if err != nil {
		return err
	}
	n.leader = true
	return nil
}

//-----------------------------------------------------------------------------
// Server API
//-----------------------------------------------------------------------------

// getServerPeers gets this node's electrumx server's peers!
func (n *Node) getServerPeers(nodeCtx context.Context) ([]*peersResult, error) {
	if !n.server.connected {
		return nil, ErrNotConnected
	}
	return n.server.conn.serverPeers(nodeCtx)
}

func (n *Node) getHeadersNotify() chan *headersNotifyResult {
	if !n.server.connected {
		return nil
	}
	return n.server.conn.getHeadersNotify()
}

func (n *Node) subscribeHeaders(nodeCtx context.Context) (*headersNotifyResult, error) {
	if !n.server.connected {
		return nil, ErrNotConnected
	}
	return n.server.conn.subscribeHeaders(nodeCtx)
}

func (n *Node) getScripthashNotify() chan *ScripthashStatusResult {
	if !n.server.connected {
		return nil
	}
	return n.server.conn.GetScripthashNotify()
}

func (n *Node) subscribeScripthashNotify(nodeCtx context.Context, scripthash string) (*ScripthashStatusResult, error) {
	if !n.server.connected {
		return nil, ErrNotConnected
	}
	return n.server.conn.SubscribeScripthash(nodeCtx, scripthash)
}

func (n *Node) unsubscribeScripthashNotify(nodeCtx context.Context, scripthash string) {
	if !n.server.connected {
		return
	}
	n.server.conn.UnsubscribeScripthash(nodeCtx, scripthash)
}

func (n *Node) blockHeader(nodeCtx context.Context, height int64) (string, error) {
	if !n.server.connected {
		return "", ErrNotConnected
	}
	blkHdr, err := n.server.conn.blockHeader(nodeCtx, uint32(height))
	if err == nil {
		n.session.bumpCostString(blkHdr)
	} else {
		n.session.bumpCostError()
	}
	return blkHdr, err
}

func (n *Node) blockHeaders(nodeCtx context.Context, startHeight int64, blockCount int) (*getBlockHeadersResult, error) {
	if !n.server.connected {
		return nil, ErrNotConnected
	}
	gbh_res, err := n.server.conn.blockHeaders(nodeCtx, startHeight, blockCount)
	if err == nil {
		n.session.bumpCostString(gbh_res.HexConcat)
	} else {
		n.session.bumpCostError()
	}
	return gbh_res, err
}

func (n *Node) getHistory(nodeCtx context.Context, scripthash string) (HistoryResult, error) {
	if !n.server.connected {
		return nil, ErrNotConnected
	}
	gh_res, err := n.server.conn.GetHistory(nodeCtx, scripthash)
	if err == nil {
		n.session.bumpCostStruct(gh_res)
	} else {
		n.session.bumpCostError()
	}
	return gh_res, err
}

func (n *Node) getListUnspent(nodeCtx context.Context, scripthash string) (ListUnspentResult, error) {
	if !n.server.connected {
		return nil, ErrNotConnected
	}
	lu_res, err := n.server.conn.GetListUnspent(nodeCtx, scripthash)
	if err == nil {
		n.session.bumpCostStruct(lu_res)

	} else {
		n.session.bumpCostError()
	}
	return lu_res, err
}

func (n *Node) getTransaction(nodeCtx context.Context, txid string) (*GetTransactionResult, error) {
	if !n.server.connected {
		return nil, ErrNotConnected
	}
	gtx_res, err := n.server.conn.getTransaction(nodeCtx, txid)
	if err == nil {
		n.session.bumpCostString(txid)
		n.session.bumpCostStruct(gtx_res)
	} else {
		n.session.bumpCostError()
	}
	return gtx_res, err
}

func (n *Node) getRawTransaction(nodeCtx context.Context, txid string) (string, error) {
	if !n.server.connected {
		return "", ErrNotConnected
	}
	grt_res, err := n.server.conn.getRawTransaction(nodeCtx, txid)
	if err == nil {
		n.session.bumpCostString(txid)
		n.session.bumpCostString(grt_res)
	} else {
		n.session.bumpCostError()
	}
	return grt_res, err
}

func (n *Node) broadcast(nodeCtx context.Context, rawTx string) (string, error) {
	if !n.server.connected {
		return "", ErrNotConnected
	}
	txid, err := n.server.conn.Broadcast(nodeCtx, rawTx)
	if err == nil {
		n.session.bumpCostString(rawTx)
		n.session.bumpCostString(txid)
	} else {
		n.session.bumpCostError()
	}
	return txid, err
}

func (n *Node) estimateFeeRate(nodeCtx context.Context, confTarget int64) (int64, error) {
	if !n.server.connected {
		return 0, ErrNotConnected
	}
	ef_res, err := n.server.conn.EstimateFee(nodeCtx, confTarget)
	if err == nil {
		n.session.bumpCostBytes(8) // 2* int64
	} else {
		n.session.bumpCostError()
	}
	return ef_res, err
}
