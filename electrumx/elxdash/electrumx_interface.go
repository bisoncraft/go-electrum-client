package elxdash

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/bisoncraft/go-electrum-client/electrumx"
	"github.com/btcsuite/btcd/wire"
	"github.com/phoreproject/go-x11"
)

// These configure ElectrumX network for: DASH
const (
	DASH_COIN                     = "dash"
	DASH_HEADER_SIZE              = 80 // https://docs.dash.org/en/stable/docs/core/reference/block-chain-block-headers.html
	DASH_STARTPOINT_REGTEST       = 0
	DASH_STARTPOINT_TESTNET       = 1225000 // March/April 2025
	DASH_STARTPOINT_MAINNET       = 2248000 // March/April 2025
	DASH_GENESIS_REGTEST          = "000008ca1832a4baf228eb1553c03d3a2c8e02399550dd6ea8d65cec3ef23d2e"
	DASH_GENESIS_TESTNET          = "00000bafbc94add76cb75e2ec92894837288a481e5c005f6563d91623bf8bc2c"
	DASH_GENESIS_MAINNET          = "00000ffd590b1485b3caadc19b22e6379c733355108f107a430458cdf3407ab6"
	DASH_MAX_ONLINE_PEERS_REGTEST = 0
	DASH_MAX_ONLINE_PEERS_TESTNET = 1
	DASH_MAX_ONLINE_PEERS_MAINNET = 3
	DASH_MAX_ONION                = 1
	DASH_STRATEGY_FLAGS_REGTEST   = electrumx.NoDeleteKnownPeers
	DASH_STRATEGY_FLAGS_TESTNET   = electrumx.NoDeleteKnownPeers
	DASH_STRATEGY_FLAGS_MAINNET   = electrumx.NoDeleteKnownPeers // 1..5 servers per kool_guy
)

type headerDeserialzer struct{}

// Deserialize deserializes a Dash block header and keeps an in memory copy of
// fields of interest. It also hashes the header using X11 hash algorithm.
// See also: https://github.com/phoreproject/go-x11/blob/master/readme.md
func (d headerDeserialzer) Deserialize(r io.Reader) (*electrumx.BlockHeader, error) {
	blockHeader := &electrumx.BlockHeader{}
	sz := int64(DASH_HEADER_SIZE)
	header := make([]byte, sz)
	_, err := io.ReadFull(r, header)
	if err != nil {
		return nil, err
	}

	// hash the header
	hs, hash := x11.New(), [32]byte{}
	hs.Hash(header, hash[:])
	// fmt.Printf("hash: %x\n", hash[:])
	blockHeader.Hash = electrumx.WireHash(hash)

	// deserialize the block header
	blockHeaderRdr := bytes.NewReader(header)
	wireHdr := &wire.BlockHeader{}
	err = wireHdr.Deserialize(blockHeaderRdr)
	if err != nil {
		return nil, err
	}
	blockHeader.Version = wireHdr.Version
	blockHeader.Prev = electrumx.WireHash(wireHdr.PrevBlock)
	blockHeader.Merkle = electrumx.WireHash(wireHdr.MerkleRoot)
	return blockHeader, nil
}

type ElectrumXInterface struct {
	config  *electrumx.ElectrumXConfig
	network *electrumx.Network
}

func NewElectrumXInterface(config *electrumx.ElectrumXConfig) (*ElectrumXInterface, error) {
	config.Coin = DASH_COIN
	config.BlockHeaderSize = DASH_HEADER_SIZE
	config.MaxOnion = DASH_MAX_ONION

	switch config.NetType {
	case electrumx.Regtest:
		config.Flags = DASH_STRATEGY_FLAGS_REGTEST
		config.Genesis = DASH_GENESIS_REGTEST
		config.StartPoint = DASH_STARTPOINT_REGTEST
		config.MaxOnlinePeers = DASH_MAX_ONLINE_PEERS_REGTEST
	case electrumx.Testnet:
		config.Flags = DASH_STRATEGY_FLAGS_TESTNET
		config.Genesis = DASH_GENESIS_TESTNET
		config.StartPoint = DASH_STARTPOINT_TESTNET
		config.MaxOnlinePeers = DASH_MAX_ONLINE_PEERS_TESTNET
	case electrumx.Mainnet:
		config.Flags = DASH_STRATEGY_FLAGS_MAINNET
		config.Genesis = DASH_GENESIS_MAINNET
		config.StartPoint = DASH_STARTPOINT_MAINNET
		config.MaxOnlinePeers = DASH_MAX_ONLINE_PEERS_MAINNET
	default:
		return nil, fmt.Errorf("config error")
	}

	config.HeaderDeserializer = &headerDeserialzer{}
	x := ElectrumXInterface{
		config:  config,
		network: nil,
	}
	return &x, nil
}

func (x *ElectrumXInterface) Start(ctx context.Context) error {
	network := electrumx.NewNetwork(x.config)
	err := network.Start(ctx)
	if err != nil {
		return err
	}
	x.network = network
	return nil
}

var ErrNoNetwork error = errors.New("btc: network not running")

func (x *ElectrumXInterface) GetTip() int64 {
	if x.network == nil {
		return 0
	}
	tip, err := x.network.Tip()
	if err != nil {
		return 0
	}
	return tip
}

func (x *ElectrumXInterface) GetSyncStatus() bool {
	if x.network == nil {
		return false
	}
	return x.network.Synced()
}

func (x *ElectrumXInterface) GetBlockHeader(height int64) (*electrumx.ClientBlockHeader, error) {
	if x.network == nil {
		return nil, ErrNoNetwork
	}
	return x.network.BlockHeader(height)
}

func (x *ElectrumXInterface) GetBlockHeaders(startHeight int64, blockCount int64) ([]*electrumx.ClientBlockHeader, error) {
	if x.network == nil {
		return nil, ErrNoNetwork
	}
	return x.network.BlockHeaders(startHeight, blockCount)
}

func (x *ElectrumXInterface) GetTipChangeNotify() (<-chan int64, error) {
	if x.network == nil {
		return nil, ErrNoNetwork
	}
	return x.network.GetTipChangeNotify(), nil
}

func (x *ElectrumXInterface) GetScripthashNotify() (<-chan *electrumx.ScripthashStatusResult, error) {
	if x.network == nil {
		return nil, ErrNoNetwork
	}
	return x.network.GetScripthashNotify(), nil
}

func (x *ElectrumXInterface) SubscribeScripthashNotify(ctx context.Context, scripthash string) (*electrumx.ScripthashStatusResult, error) {
	if x.network == nil {
		return nil, ErrNoNetwork
	}
	return x.network.SubscribeScripthashNotify(ctx, scripthash)
}

func (x *ElectrumXInterface) UnsubscribeScripthashNotify(ctx context.Context, scripthash string) {
	if x.network == nil {
		return
	}
	x.network.UnsubscribeScripthashNotify(ctx, scripthash)
}

func (x *ElectrumXInterface) GetHistory(ctx context.Context, scripthash string) (electrumx.HistoryResult, error) {
	if x.network == nil {
		return nil, ErrNoNetwork
	}
	return x.network.GetHistory(ctx, scripthash)
}

func (x *ElectrumXInterface) GetListUnspent(ctx context.Context, scripthash string) (electrumx.ListUnspentResult, error) {
	if x.network == nil {
		return nil, ErrNoNetwork
	}
	return x.network.GetListUnspent(ctx, scripthash)
}

func (x *ElectrumXInterface) GetTransaction(ctx context.Context, txid string) (*electrumx.GetTransactionResult, error) {
	if x.network == nil {
		return nil, ErrNoNetwork
	}
	return x.network.GetTransaction(ctx, txid)
}

func (x *ElectrumXInterface) GetRawTransaction(ctx context.Context, txid string) (string, error) {
	if x.network == nil {
		return "", ErrNoNetwork
	}
	return x.network.GetRawTransaction(ctx, txid)
}

func (x *ElectrumXInterface) Broadcast(ctx context.Context, rawTx string) (string, error) {
	if x.network == nil {
		return "", ErrNoNetwork
	}
	return x.network.Broadcast(ctx, rawTx)
}

func (x *ElectrumXInterface) EstimateFeeRate(ctx context.Context, confTarget int64) (int64, error) {
	if x.network == nil {
		return 0, ErrNoNetwork
	}
	return x.network.EstimateFeeRate(ctx, confTarget)
}
