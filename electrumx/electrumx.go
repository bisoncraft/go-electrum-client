package electrumx

// Package electrumx provides a client for an ElectrumX server. Not all methods
// are implemented. For the methods and their request and response types, see
// https://electrumx.readthedocs.io/en/latest/protocol-methods.html.

import (
	"context"
	"encoding/hex"
	"io"
	"net"

	"github.com/btcsuite/btcd/chaincfg"
)

const LOCALHOST = "127.0.0.1"

type Server struct {
	conn            *serverConn
	connected       bool
	softwareVersion string
	protocolVersion string
	nodeCancel      context.CancelCauseFunc
}

type NodeServerAddr struct {
	Net   string
	Addr  string
	Onion bool
}

func (nsa NodeServerAddr) Network() string {
	return nsa.Net
}

func (nsa NodeServerAddr) String() string {
	return nsa.Addr
}

func (nsa NodeServerAddr) IsOnion() bool {
	return nsa.Onion
}

func (nsa *NodeServerAddr) IsEqual(other *NodeServerAddr) bool {
	return nsa.Addr == other.Addr && nsa.Net == other.Net
}

// Ensure NodeServerAddr implements the net.Addr interface.
var _ net.Addr = NodeServerAddr{}

// Strategy flags
const (
	// The Default strategy is for a more dynamic electrumx network such as BTC
	// where we remove servers from the knownServers slice and also from the
	// 'network_servers.json' file when they disconnect or misbehave. If a new
	// server is found (server.peers.subscribe RPC), which only works iff the
	// leader's server has PEER_DISCOVERY environment variable configured, then
	// it is added to both network.go's knownServers slice & 'network_servers.json'.
	Default = uint8(0x00)

	// For static sized electrumx networks like Firo we know and maybe trust all
	// or most of the servers and we pre-populate 'network_servers.json' with the
	// servers in the network. These are never deleted or removed from the knownServers
	// slice or 'network_servers.json'.
	NoDeleteKnownPeers = uint8(0x01)
)

const (
	MAINNET = "mainnet"
	TESTNET = "testnet"
	REGTEST = "regtest"
)

// TODO: remove
// const (
// 	COIN_BTC = "btc"
// )

// Assumption: all hashes are 32 bytes long
const HashSize = 32

type WireHash [HashSize]byte

func (wh *WireHash) String() string {
	return hex.EncodeToString(wh[:])
}

type BlockHeader struct {
	Version int32
	Hash    WireHash
	Prev    WireHash
	Merkle  WireHash
}

type HeaderDeserializer interface {
	Deserialize(r io.Reader) (*BlockHeader, error)
}

// For client use
type ClientBlockHeader struct {
	Hash   string
	Prev   string
	Merkle string
}

type ElectrumXConfig struct {
	// Coin ticker to id the coin
	// Filled in by each coin in ElectrumXInterface
	Coin string

	// Size of a block header for this chain, normally 80 bytes.
	// Filled in by each coin in ElectrumXInterface
	BlockHeaderSize int

	// How the coin serializes the block header into a version, block hash,
	// previous block hash and merkle root.
	// Filled in by each coin in ElectrumXInterface
	HeaderDeserializer HeaderDeserializer

	// Checkpoints for each network: mainnet, testnet, regtest
	// Filled in by each coin in ElectrumXInterface
	StartPoint int64

	// Genesis for each network: mainnet, testnet, regtest
	// Filled in by each coin in ElectrumXInterface
	Genesis string

	// Maximum online peers for each network
	// Filled in by each coin in ElectrumXInterface
	MaxOnlinePeers int

	// Strategy flags for each network
	// Filled in by each coin in ElectrumXInterface
	Flags uint8

	// mainnet, testnet, regtest
	NetType string

	// NetType parameters.. can chaincfg adapt for all coins? for now we use the NetType
	// for everything except for genesis block hash.
	Params *chaincfg.Params

	// A localhost socks5 proxy port. E.g.  9050
	ProxyPort string

	// MaxOnion is the max onion peers we want to start for a coin
	MaxOnion int

	// Location of the data directory
	DataDir string

	// If you wish to connect to a single trusted electrumX peer set this. It is
	// recommended to set this for security.
	//
	// For now it *must be set*
	TrustedPeer *NodeServerAddr

	// If not testing do not overwrite existing wallet files
	Testing bool
}

var Regtest string = "regtest"
var Testnet string = "testnet"
var Mainnet string = "mainnet"

var DebugMode bool

type ElectrumX interface {
	Start(ctx context.Context) error

	GetTip() int64
	GetSyncStatus() bool
	GetBlockHeader(height int64) (*ClientBlockHeader, error)
	GetBlockHeaders(startHeight int64, blockCount int64) ([]*ClientBlockHeader, error)
	GetTipChangeNotify() (<-chan int64, error)

	SubscribeScripthashNotify(ctx context.Context, scripthash string) (*ScripthashStatusResult, error)
	UnsubscribeScripthashNotify(ctx context.Context, scripthash string)
	GetScripthashNotify() (<-chan *ScripthashStatusResult, error)

	GetHistory(ctx context.Context, scripthash string) (HistoryResult, error)
	GetListUnspent(ctx context.Context, scripthash string) (ListUnspentResult, error)
	GetTransaction(ctx context.Context, txid string) (*GetTransactionResult, error)
	GetRawTransaction(ctx context.Context, txid string) (string, error)
	//
	EstimateFeeRate(ctx context.Context, confTarget int64) (int64, error)
	Broadcast(ctx context.Context, rawTx string) (string, error)
}
