// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package network

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"time"

	"github.com/fulcrumchain/indigo/common"
	"github.com/fulcrumchain/indigo/log"
	"github.com/fulcrumchain/indigo/p2p/discover"
	"github.com/fulcrumchain/indigo/p2p/netutil"
	"github.com/fulcrumchain/indigo/swarm/network/kademlia"
	"github.com/fulcrumchain/indigo/swarm/storage"
)

// Hive is the logistic manager of the swarm
// it uses a generic kademlia nodetable to find best peer list
// for any target
// this is used by the netstore to search for content in the swarm
// the bzz protocol peersMsgData exchange is relayed to Kademlia
// for db storage and filtering
// connections and disconnections are reported and relayed
// to keep the nodetable uptodate

type Hive struct {
	listenAddr   func() string
	callInterval uint64
	id           discover.NodeID
	addr         kademlia.Address
	kad          *kademlia.Kademlia
	path         string
	quit         chan bool
	toggle       chan bool
	more         chan bool

	// for testing only
	swapEnabled bool
	syncEnabled bool
	blockRead   bool
	blockWrite  bool
}

const (
	callInterval = 3000000000
	// bucketSize   = 3
	// maxProx      = 8
	// proxBinSize  = 4
)

type HiveParams struct {
	CallInterval uint64
	KadDbPath    string
	*kademlia.KadParams
}

//create default params
func NewDefaultHiveParams() *HiveParams {
	kad := kademlia.NewDefaultKadParams()
	// kad.BucketSize = bucketSize
	// kad.MaxProx = maxProx
	// kad.ProxBinSize = proxBinSize

	return &HiveParams{
		CallInterval: callInterval,
		KadParams:    kad,
	}
}

//this can only finally be set after all config options (file, cmd line, env vars)
//have been evaluated
func (h *HiveParams) Init(path string) {
	h.KadDbPath = filepath.Join(path, "bzz-peers.json")
}

func NewHive(addr common.Hash, params *HiveParams, swapEnabled, syncEnabled bool) *Hive {
	kad := kademlia.New(kademlia.Address(addr), params.KadParams)
	return &Hive{
		callInterval: params.CallInterval,
		kad:          kad,
		addr:         kad.Addr(),
		path:         params.KadDbPath,
		swapEnabled:  swapEnabled,
		syncEnabled:  syncEnabled,
	}
}

func (h *Hive) SyncEnabled(on bool) {
	h.syncEnabled = on
}

func (h *Hive) SwapEnabled(on bool) {
	h.swapEnabled = on
}

func (h *Hive) BlockNetworkRead(on bool) {
	h.blockRead = on
}

func (h *Hive) BlockNetworkWrite(on bool) {
	h.blockWrite = on
}

// public accessor to the hive base address
func (h *Hive) Addr() kademlia.Address {
	return h.addr
}

// Start receives network info only at startup
// listedAddr is a function to retrieve listening address to advertise to peers
// connectPeer is a function to connect to a peer based on its NodeID or enode URL
// there are called on the p2p.Server which runs on the node
func (h *Hive) Start(id discover.NodeID, listenAddr func() string, connectPeer func(string) error) (err error) {
	h.toggle = make(chan bool)
	h.more = make(chan bool)
	h.quit = make(chan bool)
	h.id = id
	h.listenAddr = listenAddr
	err = h.kad.Load(h.path, nil)
	if err != nil {
		log.Warn(fmt.Sprintf("Warning: error reading kaddb '%s' (skipping): %v", h.path, err))
		err = nil
	}
	// this loop is doing bootstrapping and maintains a healthy table
	go h.keepAlive()
	go func() {
		// whenever toggled ask kademlia about most preferred peer
		for alive := range h.more {
			if !alive {
				// receiving false closes the loop while allowing parallel routines
				// to attempt to write to more (remove Peer when shutting down)
				return
			}
			node, need, proxLimit := h.kad.Suggest()

			if node != nil && len(node.Url) > 0 {
				log.Trace(fmt.Sprintf("call known bee %v", node.Url))
				// enode or any lower level connection address is unnecessary in future
				// discovery table is used to look it up.
				connectPeer(node.Url)
			}
			if need {
				// a random peer is taken from the table
				peers := h.kad.FindClosest(kademlia.RandomAddressAt(h.addr, rand.Intn(h.kad.MaxProx)), 1)
				if len(peers) > 0 {
					// a random address at prox bin 0 is sent for lookup
					randAddr := kademlia.RandomAddressAt(h.addr, proxLimit)
					req := &retrieveRequestMsgData{
						Key: storage.Key(randAddr[:]),
					}
					log.Trace(fmt.Sprintf("call any bee near %v (PO%03d) - messenger bee: %v", randAddr, proxLimit, peers[0]))
					peers[0].(*peer).retrieve(req)
				} else {
					log.Warn(fmt.Sprintf("no peer"))
				}
				log.Trace(fmt.Sprintf("buzz kept alive"))
			} else {
				log.Info(fmt.Sprintf("no need for more bees"))
			}
			select {
			case h.toggle <- need:
			case <-h.quit:
				return
			}
			log.Debug(fmt.Sprintf("queen's address: %v, population: %d (%d)", h.addr, h.kad.Count(), h.kad.DBCount()))
		}
	}()
	return
}

// keepAlive is a forever loop
// in its awake state it periodically triggers connection attempts
// by writing to self.more until Kademlia Table is saturated
// wake state is toggled by writing to self.toggle
// it restarts if the table becomes non-full again due to disconnections
func (h *Hive) keepAlive() {
	alarm := time.NewTicker(time.Duration(h.callInterval)).C
	for {
		select {
		case <-alarm:
			if h.kad.DBCount() > 0 {
				select {
				case h.more <- true:
					log.Debug(fmt.Sprintf("buzz wakeup"))
				default:
				}
			}
		case need := <-h.toggle:
			if alarm == nil && need {
				alarm = time.NewTicker(time.Duration(h.callInterval)).C
			}
			if alarm != nil && !need {
				alarm = nil

			}
		case <-h.quit:
			return
		}
	}
}

func (h *Hive) Stop() error {
	// closing toggle channel quits the updateloop
	close(h.quit)
	return h.kad.Save(h.path, saveSync)
}

// called at the end of a successful protocol handshake
func (h *Hive) addPeer(p *peer) error {
	defer func() {
		select {
		case h.more <- true:
		default:
		}
	}()
	log.Trace(fmt.Sprintf("hi new bee %v", p))
	err := h.kad.On(p, loadSync)
	if err != nil {
		return err
	}
	// h lookup (can be encoded as nil/zero key since peers addr known) + no id ()
	// the most common way of saying hi in bzz is initiation of gossip
	// let me know about anyone new from my hood , here is the storageradius
	// to send the 6 byte h lookup
	// we do not record as request or forward it, just reply with peers
	p.retrieve(&retrieveRequestMsgData{})
	log.Trace(fmt.Sprintf("'whatsup wheresdaparty' sent to %v", p))

	return nil
}

// called after peer disconnected
func (h *Hive) removePeer(p *peer) {
	log.Debug(fmt.Sprintf("bee %v removed", p))
	h.kad.Off(p, saveSync)
	select {
	case h.more <- true:
	default:
	}
	if h.kad.Count() == 0 {
		log.Debug(fmt.Sprintf("empty, all bees gone"))
	}
}

// Retrieve a list of live peers that are closer to target than us
func (h *Hive) getPeers(target storage.Key, max int) (peers []*peer) {
	var addr kademlia.Address
	copy(addr[:], target[:])
	for _, node := range h.kad.FindClosest(addr, max) {
		peers = append(peers, node.(*peer))
	}
	return
}

// disconnects all the peers
func (h *Hive) DropAll() {
	log.Info(fmt.Sprintf("dropping all bees"))
	for _, node := range h.kad.FindClosest(kademlia.Address{}, 0) {
		node.Drop()
	}
}

// contructor for kademlia.NodeRecord based on peer address alone
// TODO: should go away and only addr passed to kademlia
func newNodeRecord(addr *peerAddr) *kademlia.NodeRecord {
	now := time.Now()
	return &kademlia.NodeRecord{
		Addr:  addr.Addr,
		Url:   addr.String(),
		Seen:  now,
		After: now,
	}
}

// called by the protocol when receiving peerset (for target address)
// peersMsgData is converted to a slice of NodeRecords for Kademlia
// this is to store all thats needed
func (h *Hive) HandlePeersMsg(req *peersMsgData, from *peer) {
	var nrs []*kademlia.NodeRecord
	for _, p := range req.Peers {
		if err := netutil.CheckRelayIP(from.remoteAddr.IP, p.IP); err != nil {
			log.Trace(fmt.Sprintf("invalid peer IP %v from %v: %v", from.remoteAddr.IP, p.IP, err))
			continue
		}
		nrs = append(nrs, newNodeRecord(p))
	}
	h.kad.Add(nrs)
}

// peer wraps the protocol instance to represent a connected peer
// it implements kademlia.Node interface
type peer struct {
	*bzz // protocol instance running on peer connection
}

// protocol instance implements kademlia.Node interface (embedded peer)
func (p *peer) Addr() kademlia.Address {
	return p.remoteAddr.Addr
}

func (p *peer) Url() string {
	return p.remoteAddr.String()
}

// TODO take into account traffic
func (p *peer) LastActive() time.Time {
	return p.lastActive
}

// reads the serialised form of sync state persisted as the 'Meta' attribute
// and sets the decoded syncState on the online node
func loadSync(record *kademlia.NodeRecord, node kademlia.Node) error {
	p, ok := node.(*peer)
	if !ok {
		return fmt.Errorf("invalid type")
	}
	if record.Meta == nil {
		log.Debug(fmt.Sprintf("no sync state for node record %v setting default", record))
		p.syncState = &syncState{DbSyncState: &storage.DbSyncState{}}
		return nil
	}
	state, err := decodeSync(record.Meta)
	if err != nil {
		return fmt.Errorf("error decoding kddb record meta info into a sync state: %v", err)
	}
	log.Trace(fmt.Sprintf("sync state for node record %v read from Meta: %s", record, string(*(record.Meta))))
	p.syncState = state
	return err
}

// callback when saving a sync state
func saveSync(record *kademlia.NodeRecord, node kademlia.Node) {
	if p, ok := node.(*peer); ok {
		meta, err := encodeSync(p.syncState)
		if err != nil {
			log.Warn(fmt.Sprintf("error saving sync state for %v: %v", node, err))
			return
		}
		log.Trace(fmt.Sprintf("saved sync state for %v: %s", node, string(*meta)))
		record.Meta = meta
	}
}

// the immediate response to a retrieve request,
// sends relevant peer data given by the kademlia hive to the requester
// TODO: remember peers sent for duration of the session, only new peers sent
func (h *Hive) peers(req *retrieveRequestMsgData) {
	if req != nil {
		var addrs []*peerAddr
		if req.timeout == nil || time.Now().Before(*(req.timeout)) {
			key := req.Key
			// h lookup from remote peer
			if storage.IsZeroKey(key) {
				addr := req.from.Addr()
				key = storage.Key(addr[:])
				req.Key = nil
			}
			// get peer addresses from hive
			for _, peer := range h.getPeers(key, int(req.MaxPeers)) {
				addrs = append(addrs, peer.remoteAddr)
			}
			log.Debug(fmt.Sprintf("Hive sending %d peer addresses to %v. req.Id: %v, req.Key: %v", len(addrs), req.from, req.Id, req.Key.Log()))

			peersData := &peersMsgData{
				Peers: addrs,
				Key:   req.Key,
				Id:    req.Id,
			}
			peersData.setTimeout(req.timeout)
			req.from.peers(peersData)
		}
	}
}

func (h *Hive) String() string {
	return h.kad.String()
}
