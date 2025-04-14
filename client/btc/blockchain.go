package btc

import (
	"context"
	"errors"

	"github.com/bisoncraft/go-electrum-client/electrumx"
)

// Tip returns the headers current tip
func (ec *BtcElectrumClient) Tip() (tip int64) {
	return ec.GetX().GetTip()
}

// tipChange receives notifications from network leader nodes. If an api user
// has registered to receive tip change notifications - forward the notification
// on the client's registered channel. rcvTipChangeNotify is a single unbuffered
// channel kept open by network.go. Run as a goroutine from client startup.
func (ec *BtcElectrumClient) tipChange(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case tip, ok := <-ec.rcvTipChangeNotify:
			if !ok {
				return
			}
			// update wallet's notion of the tip for confirmations
			ec.updateWalletTip(tip)
			// user
			ec.sendTipChangeNotifyMtx.RLock()
			if ec.sendTipChangeNotify != nil {
				ec.sendTipChangeNotify <- tip
			}
			ec.sendTipChangeNotifyMtx.RUnlock()
		}
	}
}

// RegisterTipChangeNotify sends a new tip change channel back to an api user
func (ec *BtcElectrumClient) RegisterTipChangeNotify() (<-chan int64, error) {
	ec.sendTipChangeNotifyMtx.Lock()
	defer ec.sendTipChangeNotifyMtx.Unlock()

	if ec.sendTipChangeNotify != nil {
		return nil, errors.New("notify already registered - unregister to close the channel")
	}
	ec.sendTipChangeNotify = make(chan int64, 1)
	return ec.sendTipChangeNotify, nil
}

// UnregisterTipChangeNotify closes the current tip change channel
func (ec *BtcElectrumClient) UnregisterTipChangeNotify() {
	ec.sendTipChangeNotifyMtx.Lock()
	defer ec.sendTipChangeNotifyMtx.Unlock()

	if ec.sendTipChangeNotify != nil {
		close(ec.sendTipChangeNotify)
		ec.sendTipChangeNotify = nil
	}
}

// Synced returns the headers sync status
func (ec *BtcElectrumClient) Synced() bool {
	return ec.GetX().GetSyncStatus()
}

// GetBlockHeader returns a block header from ElectrumXInterface current stored headers
func (ec *BtcElectrumClient) GetBlockHeader(height int64) (*electrumx.ClientBlockHeader, error) {
	return ec.GetX().GetBlockHeader(height)
}

// GetBlockHeaders returns a range of block headers from ElectrumXInterface current stored headers
func (ec *BtcElectrumClient) GetBlockHeaders(startHeight, count int64) ([]*electrumx.ClientBlockHeader, error) {
	return ec.GetX().GetBlockHeaders(startHeight, count)
}
