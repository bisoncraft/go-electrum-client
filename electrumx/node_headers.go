package electrumx

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"
)

const REWIND = 8 // reorg

func (n *Node) syncHeaders(nodeCtx context.Context) error {
	return n.syncNetworkHeaders(nodeCtx)
}

// syncNetworkHeaders reads blockchain_headers file, then gets any missing block
// from end of file to current tip from server. The current set of headers is
// also stored in headers map and the chain verified by checking previous block
// hashes backwards from local Tip.
func (n *Node) syncNetworkHeaders(nodeCtx context.Context) error {
	h := n.networkHeaders

	// we start from a recent height for testnet/mainnet
	startPointHeight := h.startPoint

	// 1. Read last stored blockchain_headers file for this network

	b, err := h.readAllBytesFromFile()
	if err != nil {
		return err
	}
	lenb := int64(len(b))
	fmt.Println("read:", lenb, " bytes from header file")
	numHeaders, err := h.bytesToNumHdrs(lenb)
	if err != nil {
		return err
	}

	var maybeTip int64 = startPointHeight + numHeaders - 1

	// 2. Gather new block headers we did not have in file up to current tip

	// Do not make requested block count too big or electrumX may throttle response
	// as an anti ddos measure. ElectrumX Doc.: "Recommended to be at least one
	// Bitcoin difficulty retarget period, i.e. 2016."
	var blockCount = 2016
	var doneGathering = false
	var startHeight = startPointHeight + numHeaders

	hdrsRes, err := n.blockHeaders(nodeCtx, startHeight, blockCount)
	if err != nil {
		return err
	}
	count := hdrsRes.Count

	fmt.Printf("read: %d from server at height %d max chunk size %d\n", count, startHeight, hdrsRes.Max)

	if count > 0 {
		b, err := hex.DecodeString(hdrsRes.HexConcat)
		if err != nil {
			log.Fatal(err)
		}
		nh, err := h.appendHeadersFile(b)
		if err != nil {
			log.Fatal(err)
		}
		maybeTip += int64(count)

		fmt.Println(" appended: ", nh, " headers at ", startHeight, " maybeTip ", maybeTip)
	}

	if count < blockCount {
		doneGathering = true
	}

	for !doneGathering {

		startHeight += int64(blockCount)

		select {

		case <-nodeCtx.Done():
			<-n.server.conn.Done()
			return nil

		case <-time.After(time.Second):
			hdrsRes, err := n.blockHeaders(nodeCtx, startHeight, blockCount)
			if err != nil {
				return err
			}
			count = hdrsRes.Count

			if count > 0 {
				b, err := hex.DecodeString(hdrsRes.HexConcat)
				if err != nil {
					return err
				}
				nh, err := h.appendHeadersFile(b)
				if err != nil {
					return err
				}
				maybeTip += int64(count)

				fmt.Println(" Appended: ", nh, " headers at ", startHeight, " maybeTip ", maybeTip)
			}

			if count < blockCount {
				doneGathering = true
			}
		}
	}

	// 3. Read up to date blockchain_headers file
	b2, err := h.readAllBytesFromFile()
	if err != nil {
		return err
	}

	// 4. Store all headers in the hdrs map
	err = h.store(b2, startPointHeight)
	if err != nil {
		return err
	}
	h.setTip(maybeTip)

	// 5. Verify headers in headers map
	fmt.Printf("starting verify at height %d\n", h.getTip())
	err = h.verifyAll()
	if err != nil {
		return err
	}
	fmt.Println("header chain verified")

	h.synced = true
	fmt.Println("headers synced up to tip ", h.getTip())
	return nil
}

// headersNotify subscribes to new block tip notifications from the
// electrumx server and queues them as they arrive.
//
// ElectrumX Documentation Note:
//   - should a new block arrive quickly, perhaps while the server is still processing
//     prior blocks, the server may only notify of the most recent chain tip. The
//     protocol does not guarantee notification of all intermediate block headers.
func (n *Node) headersNotify(nodeCtx context.Context) error {
	h := n.networkHeaders
	// get a channel to receive headers notifications from this node's <- server connection
	hdrsNotifyChan := n.getHeadersNotify()
	if hdrsNotifyChan == nil {
		return errors.New("server headers notify channel is nil")
	}
	// start headers queue with depth 8
	qchan := make(chan *headersNotifyResult, 8)
	go n.headerQueue(nodeCtx, qchan)
	// subscribe headers call also returns latest header the server knows
	hdrRes, err := n.subscribeHeaders(nodeCtx)
	if err != nil {
		return err
	}
	ourTip := h.getTip()
	diff := hdrRes.Height - ourTip

	// ------------------------------------------------------------------------
	// See notes in node_headers_doc.go
	// ------------------------------------------------------------------------
	if diff < 0 {
		return fmt.Errorf("ExpBug0: diff %d between our tip and server tip"+
			" reported in subscribe.headers is negative after sync", diff)
	}
	// ------------------------------------------------------------------------

	qchan <- hdrRes

	go func() {
		fmt.Println("=== Waiting for Header Notifications")
		defer close(qchan)
		for {
			if nodeCtx.Err() != nil {
				<-n.server.conn.done
				return
			}
			// from server into queue
			hdrs := <-hdrsNotifyChan
			qchan <- hdrs
		}
	}()

	return nil
}

// headerQueue receives incoming headers notify results from qchan
// - run as a goroutine.
// The client local 'blockhain_headers' file is appended and the headers map updated and verified.
func (n *Node) headerQueue(nodeCtx context.Context, qchan <-chan *headersNotifyResult) {
	h := n.networkHeaders
	for {
		if nodeCtx.Err() != nil {
			<-n.server.conn.Done()
			return
		}

		for hdrRes := range qchan {

			if hdrRes == nil {
				return
			}

			ourTip := h.getTip()

			fmt.Printf("incoming header notification height: %d\n", hdrRes.Height)

			if hdrRes.Height < h.startPoint {
				// earlier than our starting checkpoint
				// This condition triggers node/server change
				n.server.nodeCancel(errNodeMisbehavingCanceled)
				return
			}

			if hdrRes.Height <= ourTip {
				// we already have it
				fmt.Printf(" - we already have a header for height %d\n\n", hdrRes.Height)
				continue
			}

			if hdrRes.Height == (ourTip + 1) {
				// simple case one header incoming .. try connect on top of our chain
				if !n.connectTip(hdrRes.Hex) {
					continue
				}
				n.session.bumpCostString(hdrRes.Hex)
				// connected the block & updated our headers tip
				fmt.Printf(" - updated 1 header - our new tip is %d\n\n", h.getTip())
				// notify client
				n.clientTipChangeNotify <- h.getTip()
				continue
			}
			// two or more headers that we do not have yet
			numHdrs := n.syncHeadersOntoOurTip(nodeCtx, hdrRes.Height)
			// updating less hdrs than requested is not an error - we hope to get them next time
			fmt.Printf(" - updated %d headers - our new tip is %d\n\n", numHdrs, h.getTip())
			if numHdrs > 0 {
				n.clientTipChangeNotify <- h.getTip()
			}
		}
	}
}

func (n *Node) syncHeadersOntoOurTip(nodeCtx context.Context, serverHeight int64) int64 {
	h := n.networkHeaders
	ourTip := h.getTip()
	missing := serverHeight - ourTip
	from := ourTip + 1
	to := serverHeight
	// fmt.Printf("syncHeadersFromTip: ourTip %d server height %d num missing %d\n", ourTip, serverHeight, missing)
	// per electrum, but I don't think it matters and we could always use BlockHeaders once
	if missing > REWIND {
		return n.updateFromChunk(nodeCtx, from, to)
	}
	return n.updateFromBlocks(nodeCtx, from, to)
}

func (n *Node) updateFromBlocks(nodeCtx context.Context, from, to int64) int64 {
	var headersConnected int64 = 0
	for i := from; i <= to; i++ {
		header, err := n.blockHeader(nodeCtx, i)
		if err != nil {
			break
		}
		if !n.connectTip(header) {
			break
		}
		headersConnected++
		// reduce electrumx server session cost .. per electrum
		time.Sleep(100 * time.Millisecond)
	}
	return headersConnected
}

func (n *Node) updateFromChunk(nodeCtx context.Context, from, to int64) int64 {
	h := n.networkHeaders
	var headersConnected int64 = 0
	reqCount := int(to - from + 1)
	hdrsRes, err := n.blockHeaders(nodeCtx, from, reqCount)
	if err != nil {
		return 0
	}
	// check max send parameter for validity per electrum
	if hdrsRes.Max < 2016 {
		// corrupted or electrumx server code different from that expected
		return 0
	}
	oneHdrLen := int(h.headerSize) * 2
	allHdrs := hdrsRes.HexConcat
	strLenAll := len(allHdrs)
	// check size of returned concatenated blocks
	if strLenAll != oneHdrLen*hdrsRes.Count {
		// corrupted
		return 0
	}
	// check the number of headers we asked for has been returned
	if reqCount != hdrsRes.Count {
		// maybe corrupt - maybe deliberate - I never saw this happen!
		return 0
	}
	// connect headers
	for i := 0; i < hdrsRes.Count; i++ {
		header := hdrsRes.HexConcat[i*oneHdrLen : (i+1)*oneHdrLen]
		if !n.connectTip(header) {
			break
		}
		headersConnected++
	}
	return headersConnected
}

func (n *Node) connectTip(serverHeader string) bool {
	h := n.networkHeaders
	incomingHdr, incomingHdrBytes, err := n.convertStringHdrToBlkHdr(serverHeader)
	if err != nil {
		return false
	}
	// check connect block
	if !h.checkCanConnect(incomingHdr) {
		fmt.Printf("connectTip - cannot connect\n"+
			" -- incoming hash:           %s\n"+
			" -- incoming prev hash:      %s\n"+
			" -- our current tip hash:    %s\n",
			incomingHdr.Hash.StringRev(), incomingHdr.Prev.StringRev(), h.getTipHash().StringRev())
		h.dbgDumpTipHashes(3)
		// fork maybe?
		n.reorgRecovery()
		fmt.Printf("*** removed %d stored headers from tip -  new tip is %d ***\n",
			REWIND, n.networkHeaders.getTip())
		n.session.bumpCostError()
		return false
	}
	// connect
	_, err = h.appendHeadersFile(incomingHdrBytes)
	if err != nil {
		return false
	}
	h.storeOneHdr(incomingHdr) // (sets tip++)
	h.recovery = false
	return true
}

// ElectrumX *does* fixup reorgs when it sees them but like us it cannot know
// until the next block so may send us an unconnectable block in the notification
// or block headers we ask for on that notification.
//
// So this is not a question of looking on other peers' chains for chains with more
// proof of work .. ElectrumX does that!
//
// When we cannot connect a block header we wind back our tip, hdrs map + truncate
// blockchain_headers file by REWIND block headers so that next time a notification
// comes in we ask for the last REWIND blocks. If still unconnectable on the next
// headers notification we wind back again until startPoint where we panic.
func (n *Node) reorgRecovery() {
	h := n.networkHeaders
	tip := h.getTip()
	if tip < h.startPoint+REWIND {
		// consider failover after n tries
		panic("reorgRecovery: tip <= startPoint+REWIND - cannot recover further")
	}
	// truncate and remove from map atomically
	newNumHeaders, err := h.truncateHeadersFile(REWIND)
	if err != nil {
		errMsg := fmt.Sprintf("reorgRecovery: truncateHeadersFile returned: %v", err)
		panic(errMsg)
	}
	fmt.Printf("truncateHeadersFile: new num headers is %d\n", newNumHeaders)
	for i := 0; i < REWIND; i++ {
		h.removeOneHdrFromTip() // (sets tip--)
	}

	h.recoveryTip = tip // what we  send back to users in getTip() during recovery
	h.recovery = true
}

// ----------------------------------------------------------------------------
// Util
// ----------------------------------------------------------------------------

func (n *Node) convertStringHdrToBlkHdr(svrHdr string) (*BlockHeader, []byte, error) {
	h := n.networkHeaders
	rawBytes, err := hex.DecodeString(svrHdr)
	if err != nil {
		return nil, nil, err
	}
	if len(rawBytes) != h.headerSize {
		return nil, nil, fmt.Errorf("corrupted header - length %d, expected %d",
			len(rawBytes), h.headerSize)
	}
	rdr := bytes.NewBuffer(rawBytes)
	blkHdr, err := h.headerDeserialzer.Deserialize(rdr)
	if err != nil {
		return nil, nil, errors.New("corrupted header - cannot deserialize BlockHash")
	}
	return blkHdr, rawBytes, nil
}
