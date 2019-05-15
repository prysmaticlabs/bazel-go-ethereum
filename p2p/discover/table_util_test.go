// Copyright 2018 The go-ethereum Authors
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

package discover

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
)

var nullNode *enode.Node

func init() {
	var r enr.Record
	r.Set(enr.IP{0, 0, 0, 0})
	nullNode = enode.SignNull(&r, enode.ID{})
}

func newTestTable(t transport) (*Table, *enode.DB) {
	db, _ := enode.OpenDB("")
	tab, _ := newTable(t, db, nil, log.Root())
	go tab.loop()
	return tab, db
}

// nodeAtDistance creates a node for which enode.LogDist(base, n.id) == ld.
func nodeAtDistance(base enode.ID, ld int, ip net.IP) *node {
	var r enr.Record
	r.Set(enr.IP(ip))
	return wrapNode(enode.SignNull(&r, idAtDistance(base, ld)))
}

// idAtDistance returns a random hash such that enode.LogDist(a, b) == n
func idAtDistance(a enode.ID, n int) (b enode.ID) {
	if n == 0 {
		return a
	}
	// flip bit at position n, fill the rest with random bits
	b = a
	pos := len(a) - n/8 - 1
	bit := byte(0x01) << (byte(n%8) - 1)
	if bit == 0 {
		pos++
		bit = 0x80
	}
	b[pos] = a[pos]&^bit | ^a[pos]&bit // TODO: randomize end bits
	for i := pos + 1; i < len(a); i++ {
		b[i] = byte(rand.Intn(255))
	}
	return b
}

func intIP(i int) net.IP {
	return net.IP{byte(i), 0, 2, byte(i)}
}

// fillBucket inserts nodes into the given bucket until it is full.
func fillBucket(tab *Table, n *node) (last *node) {
	ld := enode.LogDist(tab.self().ID(), n.ID())
	b := tab.bucket(n.ID())
	for len(b.entries) < bucketSize {
		b.entries = append(b.entries, nodeAtDistance(tab.self().ID(), ld, intIP(ld)))
	}
	return b.entries[bucketSize-1]
}

// fillTable adds nodes the table to the end of their corresponding bucket
// if the bucket is not full. The caller must not hold tab.mutex.
func fillTable(tab *Table, nodes []*node) {
	for _, n := range nodes {
		tab.addSeenNode(n)
	}
}

// pingRecorder is a table transport that records sent ping messages in a map.
type pingRecorder struct {
	mu           sync.Mutex
	dead, pinged map[enode.ID]bool
	records      map[enode.ID]*enode.Node
	n            *enode.Node
}

func newPingRecorder() *pingRecorder {
	var r enr.Record
	r.Set(enr.IP{0, 0, 0, 0})
	n := enode.SignNull(&r, enode.ID{})

	return &pingRecorder{
		dead:    make(map[enode.ID]bool),
		pinged:  make(map[enode.ID]bool),
		records: make(map[enode.ID]*enode.Node),
		n:       n,
	}
}

// setRecord updates a node record. Future calls to ping and
// requestENR will return this record.
func (t *pingRecorder) updateRecord(n *enode.Node) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.records[n.ID()] = n
}

// Stubs to satisfy the transport interface.
func (t *pingRecorder) Self() *enode.Node           { return nullNode }
func (t *pingRecorder) lookupSelf() []*enode.Node   { return nil }
func (t *pingRecorder) lookupRandom() []*enode.Node { return nil }

// ping simulates a ping request.
func (t *pingRecorder) ping(n *enode.Node) (seq uint64, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.pinged[n.ID()] = true
	if t.dead[n.ID()] {
		return 0, errTimeout
	}
	if t.records[n.ID()] != nil {
		seq = t.records[n.ID()].Seq()
	}
	return seq, nil
}

// requestENR simulates an ENR request.
func (t *pingRecorder) RequestENR(n *enode.Node) (*enode.Node, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.dead[n.ID()] || t.records[n.ID()] == nil {
		return nil, errTimeout
	}
	return t.records[n.ID()], nil
}

func hasDuplicates(slice []*node) bool {
	seen := make(map[enode.ID]bool)
	for i, e := range slice {
		if e == nil {
			panic(fmt.Sprintf("nil *Node at %d", i))
		}
		if seen[e.ID()] {
			return true
		}
		seen[e.ID()] = true
	}
	return false
}

func sortedByDistanceTo(distbase enode.ID, slice []*node) bool {
	return sort.SliceIsSorted(slice, func(i, j int) bool {
		return enode.DistCmp(distbase, slice[i].ID(), slice[j].ID()) < 0
	})
}

func hexEncPrivkey(h string) *ecdsa.PrivateKey {
	b, err := hex.DecodeString(h)
	if err != nil {
		panic(err)
	}
	key, err := crypto.ToECDSA(b)
	if err != nil {
		panic(err)
	}
	return key
}

func hexEncPubkey(h string) (ret encPubkey) {
	b, err := hex.DecodeString(h)
	if err != nil {
		panic(err)
	}
	if len(b) != len(ret) {
		panic("invalid length")
	}
	copy(ret[:], b)
	return ret
}

// checkLookupResults verifies that the results of a lookup are the closest nodes to
// the testnet's target.
func checkLookupResults(t *testing.T, tn *preminedTestnet, results []*enode.Node) {
	t.Helper()
	t.Logf("results:")
	for _, e := range results {
		t.Logf("  ld=%d, %x", enode.LogDist(tn.target.id(), e.ID()), e.ID().Bytes())
	}
	if hasDuplicates(wrapNodes(results)) {
		t.Errorf("result set contains duplicate entries")
	}
	if !sortedByDistanceTo(tn.target.id(), wrapNodes(results)) {
		t.Errorf("result set not sorted by distance to target")
	}
	wantNodes := tn.closest(tn.target.id(), len(results))
	for i := range results {
		if wantNodes[i].ID() != results[i].ID() {
			t.Errorf("results aren't the closest %d nodes", len(results))
			return
		}
	}
}

// This is the test network for the Lookup test.
// The nodes were obtained by running lookupTestnet.mine with a random NodeID as target.
var lookupTestnet = &preminedTestnet{
	target: hexEncPubkey("5d485bdcbe9bc89314a10ae9231e429d33853e3a8fa2af39f5f827370a2e4185e344ace5d16237491dad41f278f1d3785210d29ace76cd627b9147ee340b1125"),
	dists: [257][]*ecdsa.PrivateKey{
		251: {
			hexEncPrivkey("29738ba0c1a4397d6a65f292eee07f02df8e58d41594ba2be3cf84ce0fc58169"),
			hexEncPrivkey("511b1686e4e58a917f7f848e9bf5539d206a68f5ad6b54b552c2399fe7d174ae"),
			hexEncPrivkey("d09e5eaeec0fd596236faed210e55ef45112409a5aa7f3276d26646080dcfaeb"),
			hexEncPrivkey("c1e20dbbf0d530e50573bd0a260b32ec15eb9190032b4633d44834afc8afe578"),
			hexEncPrivkey("ed5f38f5702d92d306143e5d9154fb21819777da39af325ea359f453d179e80b"),
		},
		252: {
			hexEncPrivkey("1c9b1cafbec00848d2c174b858219914b42a7d5c9359b1ca03fd650e8239ae94"),
			hexEncPrivkey("e0e1e8db4a6f13c1ffdd3e96b72fa7012293ced187c9dcdcb9ba2af37a46fa10"),
			hexEncPrivkey("3d53823e0a0295cb09f3e11d16c1b44d07dd37cec6f739b8df3a590189fe9fb9"),
		},
		253: {
			hexEncPrivkey("2d0511ae9bf590166597eeab86b6f27b1ab761761eaea8965487b162f8703847"),
			hexEncPrivkey("6cfbd7b8503073fc3dbdb746a7c672571648d3bd15197ccf7f7fef3d904f53a2"),
			hexEncPrivkey("a30599b12827b69120633f15b98a7f6bc9fc2e9a0fd6ae2ebb767c0e64d743ab"),
			hexEncPrivkey("14a98db9b46a831d67eff29f3b85b1b485bb12ae9796aea98d91be3dc78d8a91"),
			hexEncPrivkey("2369ff1fc1ff8ca7d20b17e2673adc3365c3674377f21c5d9dafaff21fe12e24"),
			hexEncPrivkey("9ae91101d6b5048607f41ec0f690ef5d09507928aded2410aabd9237aa2727d7"),
			hexEncPrivkey("05e3c59090a3fd1ae697c09c574a36fcf9bedd0afa8fe3946f21117319ca4973"),
			hexEncPrivkey("06f31c5ea632658f718a91a1b1b9ae4b7549d7b3bc61cbc2be5f4a439039f3ad"),
		},
		254: {
			hexEncPrivkey("dec742079ec00ff4ec1284d7905bc3de2366f67a0769431fd16f80fd68c58a7c"),
			hexEncPrivkey("ff02c8861fa12fbd129d2a95ea663492ef9c1e51de19dcfbbfe1c59894a28d2b"),
			hexEncPrivkey("4dded9e4eefcbce4262be4fd9e8a773670ab0b5f448f286ec97dfc8cf681444a"),
			hexEncPrivkey("750d931e2a8baa2c9268cb46b7cd851f4198018bed22f4dceb09dd334a2395f6"),
			hexEncPrivkey("ce1435a956a98ffec484cd11489c4f165cf1606819ab6b521cee440f0c677e9e"),
			hexEncPrivkey("996e7f8d1638be92d7328b4770f47e5420fc4bafecb4324fd33b1f5d9f403a75"),
			hexEncPrivkey("ebdc44e77a6cc0eb622e58cf3bb903c3da4c91ca75b447b0168505d8fc308b9c"),
			hexEncPrivkey("46bd1eddcf6431bea66fc19ebc45df191c1c7d6ed552dcdc7392885009c322f0"),
		},
		255: {
			hexEncPrivkey("da8645f90826e57228d9ea72aff84500060ad111a5d62e4af831ed8e4b5acfb8"),
			hexEncPrivkey("3c944c5d9af51d4c1d43f5d0f3a1a7ef65d5e82744d669b58b5fed242941a566"),
			hexEncPrivkey("5ebcde76f1d579eebf6e43b0ffe9157e65ffaa391175d5b9aa988f47df3e33da"),
			hexEncPrivkey("97f78253a7d1d796e4eaabce721febcc4550dd68fb11cc818378ba807a2cb7de"),
			hexEncPrivkey("a38cd7dc9b4079d1c0406afd0fdb1165c285f2c44f946eca96fc67772c988c7d"),
			hexEncPrivkey("d64cbb3ffdf712c372b7a22a176308ef8f91861398d5dbaf326fd89c6eaeef1c"),
			hexEncPrivkey("d269609743ef29d6446e3355ec647e38d919c82a4eb5837e442efd7f4218944f"),
			hexEncPrivkey("d8f7bcc4a530efde1d143717007179e0d9ace405ddaaf151c4d863753b7fd64c"),
		},
		256: {
			hexEncPrivkey("8c5b422155d33ea8e9d46f71d1ad3e7b24cb40051413ffa1a81cff613d243ba9"),
			hexEncPrivkey("937b1af801def4e8f5a3a8bd225a8bcff1db764e41d3e177f2e9376e8dd87233"),
			hexEncPrivkey("120260dce739b6f71f171da6f65bc361b5fad51db74cf02d3e973347819a6518"),
			hexEncPrivkey("1fa56cf25d4b46c2bf94e82355aa631717b63190785ac6bae545a88aadc304a9"),
			hexEncPrivkey("3c38c503c0376f9b4adcbe935d5f4b890391741c764f61b03cd4d0d42deae002"),
			hexEncPrivkey("3a54af3e9fa162bc8623cdf3e5d9b70bf30ade1d54cc3abea8659aba6cff471f"),
			hexEncPrivkey("6799a02ea1999aefdcbcc4d3ff9544478be7365a328d0d0f37c26bd95ade0cda"),
			hexEncPrivkey("e24a7bc9051058f918646b0f6e3d16884b2a55a15553b89bab910d55ebc36116"),
		},
	},
}

type preminedTestnet struct {
	target encPubkey
	dists  [hashBits + 1][]*ecdsa.PrivateKey
	mu     sync.Mutex
	ncache map[*ecdsa.PrivateKey]*enode.Node
}

func (tn *preminedTestnet) node(dist, index int) *enode.Node {
	tn.mu.Lock()
	defer tn.mu.Unlock()

	key := tn.dists[dist][index]
	if tn.ncache[key] == nil {
		return tn.makeNode(dist, index)
	}
	return tn.ncache[key]
}

func (tn *preminedTestnet) makeNode(dist, index int) *enode.Node {
	if tn.ncache == nil {
		tn.ncache = make(map[*ecdsa.PrivateKey]*enode.Node)
	}

	var record enr.Record
	record.Set(enr.IP{127, byte(dist >> 8), byte(dist), byte(index)})
	record.Set(enr.UDP(5000))
	key := tn.dists[dist][index]
	if err := enode.SignV4(&record, key); err != nil {
		panic("can't sign: " + err.Error())
	}
	n, _ := enode.New(enode.ValidSchemes, &record)
	tn.ncache[key] = n
	return n
}

func (tn *preminedTestnet) nodeByAddr(addr *net.UDPAddr) (*enode.Node, *ecdsa.PrivateKey) {
	dist := int(addr.IP[1])<<8 + int(addr.IP[2])
	index := int(addr.IP[3])
	key := tn.dists[dist][index]
	return tn.node(dist, index), key
}

func (tn *preminedTestnet) nodesAtDistance(dist int) []rpcNode {
	result := make([]rpcNode, len(tn.dists[dist]))
	for i := range result {
		result[i] = nodeToRPC(wrapNode(tn.node(dist, i)))
	}
	return result
}

func (tn *preminedTestnet) recordsAtDistance(dist int) []*enr.Record {
	result := make([]*enr.Record, len(tn.dists[dist]))
	for i := range result {
		result[i] = tn.node(dist, i).Record()
	}
	return result
}

func (tn *preminedTestnet) closest(target enode.ID, n int) (nodes []*enode.Node) {
	for d := range tn.dists {
		for i := range tn.dists[d] {
			nodes = append(nodes, tn.node(d, i))
		}
	}
	sort.Slice(nodes, func(i, j int) bool {
		return enode.DistCmp(target, nodes[i].ID(), nodes[j].ID()) < 0
	})
	return nodes[:n]
}

var _ = (*preminedTestnet).mine // avoid linter warning about mine being dead code.

// mine generates a testnet struct literal with nodes at
// various distances to the network's target.
func (tn *preminedTestnet) mine() {
	// Clear existing slices first (useful when re-mining).
	for i := range tn.dists {
		tn.dists[i] = nil
	}

	targetSha := tn.target.id()
	found, need := 0, 40
	for found < need {
		k := newkey()
		ld := enode.LogDist(targetSha, encodePubkey(&k.PublicKey).id())
		if len(tn.dists[ld]) < 8 {
			tn.dists[ld] = append(tn.dists[ld], k)
			found++
			fmt.Printf("found ID with ld %d (%d/%d)\n", ld, found, need)
		}
	}
	fmt.Printf("&preminedTestnet{\n")
	fmt.Printf("	target: hexEncPubkey(\"%x\"),\n", tn.target[:])
	fmt.Printf("	dists: [%d][]*ecdsa.PrivateKey{\n", len(tn.dists))
	for ld, ns := range tn.dists {
		if len(ns) == 0 {
			continue
		}
		fmt.Printf("		%d: {\n", ld)
		for _, key := range ns {
			fmt.Printf("			hexEncPrivkey(\"%x\"),\n", crypto.FromECDSA(key))
		}
		fmt.Printf("		},\n")
	}
	fmt.Printf("	},\n")
	fmt.Printf("}\n")
}

// dgramPipe is a fake UDP socket. It queues all sent datagrams.
type dgramPipe struct {
	mu      *sync.Mutex
	cond    *sync.Cond
	closing chan struct{}
	closed  bool
	queue   []dgram
}

type dgram struct {
	to   net.UDPAddr
	data []byte
}

func newpipe() *dgramPipe {
	mu := new(sync.Mutex)
	return &dgramPipe{
		closing: make(chan struct{}),
		cond:    &sync.Cond{L: mu},
		mu:      mu,
	}
}

// WriteToUDP queues a datagram.
func (c *dgramPipe) WriteToUDP(b []byte, to *net.UDPAddr) (n int, err error) {
	msg := make([]byte, len(b))
	copy(msg, b)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return 0, errors.New("closed")
	}
	c.queue = append(c.queue, dgram{*to, b})
	c.cond.Signal()
	return len(b), nil
}

// ReadFromUDP just hangs until the pipe is closed.
func (c *dgramPipe) ReadFromUDP(b []byte) (n int, addr *net.UDPAddr, err error) {
	<-c.closing
	return 0, nil, io.EOF
}

func (c *dgramPipe) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		close(c.closing)
		c.closed = true
	}
	c.cond.Broadcast()
	return nil
}

func (c *dgramPipe) LocalAddr() net.Addr {
	return &net.UDPAddr{IP: testLocal.IP, Port: int(testLocal.UDP)}
}

func (c *dgramPipe) receive() (dgram, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var timedOut bool
	timer := time.AfterFunc(3*time.Second, func() {
		c.mu.Lock()
		timedOut = true
		c.mu.Unlock()
		c.cond.Broadcast()
	})
	defer timer.Stop()

	for len(c.queue) == 0 && !c.closed && !timedOut {
		c.cond.Wait()
	}
	if c.closed {
		return dgram{}, errClosed
	}
	if timedOut {
		return dgram{}, errTimeout
	}
	p := c.queue[0]
	copy(c.queue, c.queue[1:])
	c.queue = c.queue[:len(c.queue)-1]
	return p, nil
}
