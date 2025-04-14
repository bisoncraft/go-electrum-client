package elxfiro

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/bisoncraft/go-electrum-client/electrumx"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

const (
	FIRO_HEADER_SIZE         = 80
	FIRO_FIROPOW_EXTRA       = 40
	FIRO_FIROPOW_HEADER_SIZE = FIRO_HEADER_SIZE + FIRO_FIROPOW_EXTRA
)

// These configure ElectrumX network for: FIRO
const (
	FIRO_COIN                = "firo"
	FIRO_HEADER_SIZE_REGTEST = 80
	FIRO_HEADER_SIZE_FIROPOW = 120
	// FIRO_HEADER_SIZE              = 80 // check this for MTP legacy. Now FiroPoW (ProgPow clone) .. should be 80
	FIRO_STARTPOINT_REGTEST       = 0
	FIRO_STARTPOINT_TESTNET       = 170_000
	FIRO_STARTPOINT_MAINNET       = 987_000
	FIRO_GENESIS_REGTEST          = "a42b98f04cc2916e8adfb5d9db8a2227c4629bc205748ed2f33180b636ee885b"
	FIRO_GENESIS_TESTNET          = "aa22adcc12becaf436027ffe62a8fb21b234c58c23865291e5dc52cf53f64fca"
	FIRO_GENESIS_MAINNET          = "4381deb85b1b2c9843c222944b616d997516dcbd6a964e1eaf0def0830695233"
	FIRO_MAX_ONLINE_PEERS_REGTEST = 0
	FIRO_MAX_ONLINE_PEERS_TESTNET = 0 // only one testnet server 95.179.164.13:51002 - v0.14.14.0
	FIRO_MAX_ONLINE_PEERS_MAINNET = 3 // only 4 servers                              - v0.14.14.0
	FIRO_MAX_ONION                = 0
	FIRO_STRATEGY_FLAGS_REGTEST   = electrumx.NoDeleteKnownPeers // only one server
	FIRO_STRATEGY_FLAGS_TESTNET   = electrumx.NoDeleteKnownPeers // only one server
	FIRO_STRATEGY_FLAGS_MAINNET   = electrumx.NoDeleteKnownPeers // 4 servers
)

type headerDeserializer struct{}

func (d headerDeserializer) Deserialize(r io.Reader) (*electrumx.BlockHeader, error) {
	blockHeader := &electrumx.BlockHeader{}
	sz := int64(FIRO_FIROPOW_HEADER_SIZE)
	fullHeader := make([]byte, sz)
	_, err := io.ReadFull(r, fullHeader)
	if err != nil {
		return nil, err
	}

	// hash full header
	hash := chainhash.DoubleHashH(fullHeader)
	blockHeader.Hash = electrumx.WireHash(hash)

	// deserialize the block header without the extra progpow bytes
	blockHeaderRdr := bytes.NewReader(fullHeader[:FIRO_HEADER_SIZE])
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

type regtestHeaderDeserializer struct{}

func (d regtestHeaderDeserializer) Deserialize(r io.Reader) (*electrumx.BlockHeader, error) {
	wireHdr := &wire.BlockHeader{}
	err := wireHdr.Deserialize(r)
	if err != nil {
		return nil, err
	}
	blockHeader := &electrumx.BlockHeader{}
	blockHeader.Version = wireHdr.Version
	chainHash := wireHdr.BlockHash()
	blockHeader.Hash = electrumx.WireHash(chainHash)
	blockHeader.Prev = electrumx.WireHash(wireHdr.PrevBlock)
	blockHeader.Merkle = electrumx.WireHash(wireHdr.MerkleRoot)
	return blockHeader, nil
}

type ElectrumXInterface struct {
	config  *electrumx.ElectrumXConfig
	network *electrumx.Network
}

func NewElectrumXInterface(config *electrumx.ElectrumXConfig) (*ElectrumXInterface, error) {
	config.Coin = FIRO_COIN
	config.MaxOnion = FIRO_MAX_ONION

	switch config.NetType {
	case electrumx.Regtest:
		config.Flags = FIRO_STRATEGY_FLAGS_REGTEST
		config.HeaderDeserializer = regtestHeaderDeserializer{}
		config.BlockHeaderSize = FIRO_HEADER_SIZE_REGTEST
		config.Genesis = FIRO_GENESIS_REGTEST
		config.StartPoint = FIRO_STARTPOINT_REGTEST
		config.MaxOnlinePeers = FIRO_MAX_ONLINE_PEERS_REGTEST
	case electrumx.Testnet:
		config.Flags = FIRO_STRATEGY_FLAGS_TESTNET
		config.HeaderDeserializer = headerDeserializer{}
		config.BlockHeaderSize = FIRO_HEADER_SIZE_FIROPOW
		config.Genesis = FIRO_GENESIS_TESTNET
		config.StartPoint = FIRO_STARTPOINT_TESTNET
		config.MaxOnlinePeers = FIRO_MAX_ONLINE_PEERS_TESTNET
	case electrumx.Mainnet:
		config.Flags = FIRO_STRATEGY_FLAGS_TESTNET
		config.HeaderDeserializer = headerDeserializer{}
		config.BlockHeaderSize = FIRO_HEADER_SIZE_FIROPOW
		config.Genesis = FIRO_GENESIS_MAINNET
		config.StartPoint = FIRO_STARTPOINT_MAINNET
		config.MaxOnlinePeers = FIRO_MAX_ONLINE_PEERS_MAINNET
	default:
		return nil, fmt.Errorf("config error")
	}

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

var ErrNoNetwork error = errors.New("firo: network not running")

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
