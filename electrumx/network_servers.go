package electrumx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path"
	"strings"
)

const ServerAddrFileName = "network_servers.json"

type serverAddr struct {
	Net     string `json:"net"`
	Address string `json:"addr"`
	Host    string `json:"host"`
	IsOnion bool   `json:"is_onion"`
	Version string `json:"version"`
	Caps    string `json:"caps"` // comma separated string eg. "cannot_lead,no_blks,..."
	Rep     int    `json:"rep"`  // reputation score - not fully implemented as yet
}

// Incoming list from server_connection.go - constructed using reflection
//
//	  type PeersResult struct {
//	  	  Addr  string // IP address or .onion name
//		  Host  string
//		  Feats []string
//	  }
//
// ..which we build into a list of serverAddr structs which can be used in both
// memory and persisted to 'network_servers.json' file in the data dir.

// getServers gets server peers (from the peerNode's server) which that server
// currently knows; which can be different than known persisted servers held by
// goele.
func (net *Network) getServers(ctx context.Context) error {
	if !net.started {
		return errNoNetwork
	}
	if net.getLeader() == nil {
		return errNoLeader
	}
	peerResults, err := net.getLeader().node.getServerPeers(ctx)
	if err != nil {
		return err
	}
	if len(peerResults) == 0 {
		return nil
	}
	return net.addIncomingServers(peerResults)
}

// addIncomingServers converts incoming peerResults and updates memory & stored
// server lists
func (net *Network) addIncomingServers(in []*peersResult) error {
	servers := makeIncomingServerAddrs(in)
	if len(servers) == 0 {
		// I have seen at least one peer on testnet return an empty list
		return errors.New("no incoming")
	}
	return net.updateNetworkServers(servers)
}

func makeIncomingServerAddrs(in []*peersResult) []*serverAddr {
	var servers = make([]*serverAddr, 0, 2*len(in))
	var goodAddresses = 0
	var badAddresses = 0

	for _, pres := range in {
		isOnion := false
		if strings.HasSuffix(pres.Addr, ".onion") {
			onionAddr := strings.Split(pres.Host, ".")
			isOnion = true
			if len(onionAddr[0]) != 56 { // no V2
				badAddresses++
				continue
			}
		} else {
			if net.ParseIP(pres.Addr) == nil {
				badAddresses++
				continue
			}
		}

		var isTcp bool
		var isSsl bool

		var version string
		var tcpPort string
		var sslPort string
		feats := pres.Feats
		for _, feat := range feats {
			switch []rune(feat)[0] {
			case 'v':
				version = feat[1:]
			case 't':
				tcpPort = feat[1:]
				isTcp = true
			case 's':
				sslPort = feat[1:]
				isSsl = true
			}
		}
		if isTcp {
			if len(tcpPort) == 0 {
				tcpPort = "51001" // default if no explicit port after 't'
			}
			saddr := &serverAddr{
				Net:     "tcp",
				Address: net.JoinHostPort(pres.Addr, tcpPort),
				Host:    pres.Host,
				IsOnion: isOnion,
				Version: version,
				Caps:    "",
			}
			servers = append(servers, saddr)
		}
		if isSsl {
			if len(sslPort) == 0 {
				sslPort = "51002" // default if no explicit port after 's'
			}
			saddr := &serverAddr{
				Net:     "ssl",
				Address: net.JoinHostPort(pres.Addr, sslPort),
				Host:    pres.Host,
				IsOnion: isOnion,
				Version: version,
				Caps:    "",
			}
			servers = append(servers, saddr)
		}
		goodAddresses++
	}
	return servers
}

func (net *Network) updateNetworkServers(servers []*serverAddr) error {
	net.knownServersMtx.Lock()
	defer net.knownServersMtx.Unlock()

	if len(net.knownServers) == 0 {
		net.knownServers = servers
		return nil
	}
	// add any new servers that we do not have already
	var tmpAdd = []*serverAddr{}
	for _, new := range servers { // for each incoming server
		matchedKnown := false
		for _, got := range net.knownServers { // loop through what we have already
			// already got?
			if got.Address == new.Address {
				matchedKnown = true
			}
		}
		if !matchedKnown {
			tmpAdd = append(tmpAdd, new)
		}
	}
	net.knownServers = append(net.knownServers, tmpAdd...)
	return nil
}

func (net *Network) updateStoredServers(servers []*serverAddr) error {
	net.knownServersMtx.Lock()
	defer net.knownServersMtx.Unlock()

	stored, numRead, err := net.readServerAddrFile()
	if err != nil {
		return err
	}
	if numRead == 0 {
		stored = servers
		return net.writeServerAddrFile(stored)
	}
	// add any new servers that we have not stored already
	var tmpAdd = []*serverAddr{}
	for _, new := range servers {
		matchedKnown := false
		for _, got := range stored {
			if got.Address == new.Address {
				matchedKnown = true
			}
		}
		if !matchedKnown {
			tmpAdd = append(tmpAdd, new)
		}
	}
	stored = append(stored, tmpAdd...)
	return net.writeServerAddrFile(stored)
}

func (net *Network) removeServer(server *serverAddr) error {
	if net.config.Flags&NoDeleteKnownPeers == NoDeleteKnownPeers {
		fmt.Printf("removeServer: not removing %s - Strategy: NoDeleteStoredPeers\n", server.Address)
		return nil
	}

	net.knownServersMtx.Lock()
	defer net.knownServersMtx.Unlock()

	// remove from memory first
	var lessKnown = make([]*serverAddr, 0)
	for _, known := range net.knownServers {
		if known.Address == server.Address {
			continue
		}
		lessKnown = append(lessKnown, known)
	}
	net.knownServers = lessKnown

	// remove from file
	stored, _, err := net.readServerAddrFile()
	if err != nil {
		return err
	}
	// find server in the list and remove
	var lessStored []*serverAddr = []*serverAddr{}
	for _, got := range stored {
		if got.Address == server.Address {
			continue
		}
		lessStored = append(lessStored, got)
	}

	return net.writeServerAddrFile(lessStored)
}

func (net *Network) readServerAddrFile() ([]*serverAddr, int, error) {
	serverAddrFile := path.Join(net.config.DataDir, ServerAddrFileName)
	f, err := os.OpenFile(serverAddrFile, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, 0, err
	}
	if len(b) == 0 {
		// 0 read is not an error
		return nil, 0, nil
	}
	var servers []*serverAddr
	buf := bytes.NewBuffer(b)
	d := json.NewDecoder(buf)
	err = d.Decode(&servers)
	if err != nil {
		return nil, 0, err
	}
	return servers, len(servers), nil
}

func (net *Network) writeServerAddrFile(servers []*serverAddr) error {
	jsonBytes, err := json.MarshalIndent(servers, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	if len(servers) == 0 {
		// json will marshal an empty slice to '[]' ..as it should ;-)
		// so write nil slice after truncation so we get an empty file
		// this is checked for just before the write syscall
		jsonBytes = []byte(nil)
	}
	// write all marshalled json after truncating file to 0 bytes
	serverAddrFile := path.Join(net.config.DataDir, ServerAddrFileName)
	f, err := os.OpenFile(serverAddrFile, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	err = f.Truncate(0)
	if err != nil {
		return err
	}
	newFileOffset, err := f.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	if newFileOffset != 0 {
		return fmt.Errorf("bad file offset %d after truncate then seek(0,0)", newFileOffset)
	}
	n, err := f.Write(jsonBytes)
	if err != nil {
		return err
	}
	if n != len(jsonBytes) {
		return fmt.Errorf("written %d bytes, wanted to write %d", n, len(jsonBytes))
	}
	return nil
}

// loadKnownServers loads any stored servers at network startup
func (net *Network) loadKnownServers() (int, error) {
	net.knownServersMtx.Lock() // not really needed when called during net.start
	defer net.knownServersMtx.Unlock()
	stored, numRead, err := net.readServerAddrFile()
	if err != nil {
		return 0, err
	}
	net.knownServers = stored
	return numRead, nil
}
