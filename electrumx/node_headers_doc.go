package electrumx

// ExpBug0:
//
// Server can send blockHeader(s) calls correctly during headers sync but sends
// first notification response for an old block on headers.subscribe call and
// never sends any notification after that.

// Debug output:
// ---------------------------------------------------------------------------
// . . .
//
// ** Connected to testnet.aranguren.org:51002 over ssl on testnet3 ***
//    Using server software version Fulcrum 1.11.0 protocol version 1.4
//    Genesis 000000000933ea01ad0ee984209779baaec3ced90fa3f408719526f8d77f4943
// read: 24660640  bytes from header file
//  - incurred: 0.000000, total-cost: 0.000000
// read: 0 from server at height 2868258 max chunk size 2016
// starting verify at height 2868257
// header chain verified
// headers synced up to tip  2868257
// headrs queue started
// subscribe headers - height 2868044 our tip 2868257 diff -213
// . . .
// ---------------------------------------------------------------------------
// - The server just synced or we are already synced .. But then it sends back a
//   notification subscription reply with a height 213 blocks behind the blocks we
//   already loaded using blockHeader(s) calls.
// ----------------------------------------------------------------------------

// Analysis & Fix:

// - I think it is a bug in Fulcrum 1.11.0 code because I only saw this on servers
//   running that software and only on testnet as yet. It could be malicious on
//   testnet - probes maybe - but I also saw for `testnet.aranguren.org` which ran
//   for many days successfully as the trusted peer.

// - Suggested experimental fix in node_headers.go:headersNotify is to send back
//   an annotated error:
//   1. If we are starting trusted peer as the leader server then the program just
//      fails.
//   2. If promoting a running peer to leader it should failover to another peer
//      so that network can try to start another leader in network.go:checkLeader.

// THE ABOVE IS FOR BTC .. WE EXPECT SHENANNIGANS!
