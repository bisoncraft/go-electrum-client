package elxbtc

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/bisoncraft/go-electrum-client/electrumx"
	"github.com/btcsuite/btcd/wire"
)

// These configure ElectrumX network for: BTC
const (
	BTC_COIN                     = "btc"
	BTC_HEADER_SIZE              = 80
	BTC_STARTPOINT_REGTEST       = 0
	BTC_STARTPOINT_TESTNET       = 2560000
	BTC_STARTPOINT_MAINNET       = 823000
	BTC_GENESIS_REGTEST          = "0f9188f13cb7b2c71f2a335e3a4fc328bf5beb436012afca590b1a11466e2206"
	BTC_GENESIS_TESTNET          = "000000000933ea01ad0ee984209779baaec3ced90fa3f408719526f8d77f4943"
	BTC_GENESIS_MAINNET          = "000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f"
	BTC_MAX_ONLINE_PEERS_REGTEST = 0
	BTC_MAX_ONLINE_PEERS_TESTNET = 3
	BTC_MAX_ONLINE_PEERS_MAINNET = 10
	BTC_MAX_ONION                = 2
	BTC_STRATEGY_FLAGS_REGTEST   = electrumx.NoDeleteKnownPeers
	BTC_STRATEGY_FLAGS_TESTNET   = electrumx.Default
	BTC_STRATEGY_FLAGS_MAINNET   = electrumx.Default
)

type headerDeserialzer struct{}

func (d headerDeserialzer) Deserialize(r io.Reader) (*electrumx.BlockHeader, error) {
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
	config.Coin = BTC_COIN
	config.BlockHeaderSize = BTC_HEADER_SIZE
	config.MaxOnion = BTC_MAX_ONION

	switch config.NetType {
	case electrumx.Regtest:
		config.Flags = BTC_STRATEGY_FLAGS_REGTEST
		config.Genesis = BTC_GENESIS_REGTEST
		config.StartPoint = BTC_STARTPOINT_REGTEST
		config.MaxOnlinePeers = BTC_MAX_ONLINE_PEERS_REGTEST
	case electrumx.Testnet:
		config.Flags = BTC_STRATEGY_FLAGS_TESTNET
		config.Genesis = BTC_GENESIS_TESTNET
		config.StartPoint = BTC_STARTPOINT_TESTNET
		config.MaxOnlinePeers = BTC_MAX_ONLINE_PEERS_TESTNET
	case electrumx.Mainnet:
		config.Flags = BTC_STRATEGY_FLAGS_MAINNET
		config.Genesis = BTC_GENESIS_MAINNET
		config.StartPoint = BTC_STARTPOINT_MAINNET
		config.MaxOnlinePeers = BTC_MAX_ONLINE_PEERS_MAINNET
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
