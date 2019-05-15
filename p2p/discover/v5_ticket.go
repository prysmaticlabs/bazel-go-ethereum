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

package discover

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

const (
	ticketTimeBucketLen = time.Minute
	timeWindow          = 10 // * ticketTimeBucketLen
	wantTicketsInWindow = 10
	collectFrequency    = time.Second * 30
	registerFrequency   = time.Second * 60
	maxCollectDebt      = 10
	maxRegisterDebt     = 5
	keepTicketConst     = time.Minute * 10
	keepTicketExp       = time.Minute * 5
	targetWaitTime      = time.Minute * 10
	topicQueryTimeout   = time.Second * 5
	topicQueryResend    = time.Minute

	// topic radius detection
	maxRadius           = 0xffffffffffffffff
	radiusTC            = time.Minute * 20
	radiusBucketsPerBit = 8
	minSlope            = 1
	minPeakSize         = 40
	maxNoAdjust         = 20
	lookupWidth         = 8
	minRightSum         = 20
	searchForceQuery    = 4
)

// timeBucket represents absolute monotonic time in minutes.
// It is used as the index into the per-topic ticket buckets.
type timeBucket int

type ticket struct {
	topic      Topic
	regTime    mclock.AbsTime // time when the ticket can be used.
	serial     uint32         // serial number issued by registrar
	issueTime  mclock.AbsTime // creation time
	waitPeriod uint32

	// Fields used only by registrants
	node *enode.Node // the registrar node that signed this ticket
	pong []byte      // encoded pong packet signed by the registrar
}

func decodeTicket([]byte) (*ticket, error) {
	return nil, nil
}

func encodeTicket(*ticket) []byte {
	return nil
}

// func pongToTicket(localTime mclock.AbsTime, topics []Topic, node *enode.Node, p *ingressPacket) (*ticket, error) {
// 	wps := p.data.(*pong).WaitPeriods
// 	if len(topics) != len(wps) {
// 		return nil, fmt.Errorf("bad wait period list: got %d values, want %d", len(topics), len(wps))
// 	}
// 	if rlpHash(topics) != p.data.(*pong).TopicHash {
// 		return nil, fmt.Errorf("bad topic hash")
// 	}
// 	t := &ticket{
// 		issueTime: localTime,
// 		node:      node,
// 		topics:    topics,
// 		pong:      p.rawData,
// 		regTime:   make([]mclock.AbsTime, len(wps)),
// 	}
// 	// Convert wait periods to local absolute time.
// 	for i, wp := range wps {
// 		t.regTime[i] = localTime + mclock.AbsTime(time.Second*time.Duration(wp))
// 	}
// 	return t, nil
// }

// func ticketToPong(t *ticket, pong *pong) {
// 	pong.Expiration = uint64(t.issueTime / mclock.AbsTime(time.Second))
// 	pong.TopicHash = rlpHash(t.topics)
// 	pong.TicketSerial = t.serial
// 	pong.WaitPeriods = make([]uint32, len(t.regTime))
// 	for i, regTime := range t.regTime {
// 		pong.WaitPeriods[i] = uint32(time.Duration(regTime-t.issueTime) / time.Second)
// 	}
// }

type ticketStore struct {
	// radius detector and target address generator
	// exists for both searched and registered topics
	radius map[Topic]*topicRadius

	// Contains buckets (for each absolute minute) of tickets
	// that can be used in that minute.
	// This is only set if the topic is being registered.
	tickets map[Topic]*topicTickets

	regQueue []Topic            // Topic registration queue for round robin attempts
	regSet   map[Topic]struct{} // Topic registration queue contents for fast filling

	nodes       map[enode.ID]*ticket
	nodeLastReq map[enode.ID]reqInfo

	lastBucketFetched timeBucket
	nextTicketCached  *ticket
	nextTicketReg     mclock.AbsTime

	searchTopicMap        map[Topic]searchTopic
	nextTopicQueryCleanup mclock.AbsTime
	queriesSent           map[enode.ID]map[common.Hash]sentQuery
}

type searchTopic struct {
	foundChn chan<- *enode.Node
}

type sentQuery struct {
	sent   mclock.AbsTime
	lookup lookupInfo
}

type topicTickets struct {
	buckets    map[timeBucket][]*ticket
	nextLookup mclock.AbsTime
	nextReg    mclock.AbsTime
}

func newTicketStore() *ticketStore {
	return &ticketStore{
		radius:         make(map[Topic]*topicRadius),
		tickets:        make(map[Topic]*topicTickets),
		regSet:         make(map[Topic]struct{}),
		nodes:          make(map[enode.ID]*ticket),
		nodeLastReq:    make(map[enode.ID]reqInfo),
		searchTopicMap: make(map[Topic]searchTopic),
		queriesSent:    make(map[enode.ID]map[common.Hash]sentQuery),
	}
}

// addTopic starts tracking a topic. If register is true,
// the local node will register the topic and tickets will be collected.
func (s *ticketStore) addTopic(topic Topic, register bool) {
	log.Trace("Adding discovery topic", "topic", topic, "register", register)
	if s.radius[topic] == nil {
		s.radius[topic] = newTopicRadius(topic)
	}
	if register && s.tickets[topic] == nil {
		s.tickets[topic] = &topicTickets{buckets: make(map[timeBucket][]*ticket)}
	}
}

func (s *ticketStore) addSearchTopic(t Topic, foundChn chan<- *enode.Node) {
	s.addTopic(t, false)
	if s.searchTopicMap[t].foundChn == nil {
		s.searchTopicMap[t] = searchTopic{foundChn: foundChn}
	}
}

func (s *ticketStore) removeSearchTopic(t Topic) {
	if st := s.searchTopicMap[t]; st.foundChn != nil {
		delete(s.searchTopicMap, t)
	}
}

// removeRegisterTopic deletes all tickets for the given topic.
func (s *ticketStore) removeRegisterTopic(topic Topic) {
	log.Trace("Removing discovery topic", "topic", topic)
	if s.tickets[topic] == nil {
		log.Warn("Removing non-existent discovery topic", "topic", topic)
		return
	}
	for _, list := range s.tickets[topic].buckets {
		for _, ticket := range list {
			delete(s.nodes, ticket.node.ID())
			delete(s.nodeLastReq, ticket.node.ID())
		}
	}
	delete(s.tickets, topic)
}

func (s *ticketStore) regTopicSet() []Topic {
	topics := make([]Topic, 0, len(s.tickets))
	for topic := range s.tickets {
		topics = append(topics, topic)
	}
	return topics
}

// nextRegisterLookup returns the target of the next lookup for ticket collection.
func (s *ticketStore) nextRegisterLookup() (lookupInfo, time.Duration) {
	// Queue up any new topics (or discarded ones), preserving iteration order
	for topic := range s.tickets {
		if _, ok := s.regSet[topic]; !ok {
			s.regQueue = append(s.regQueue, topic)
			s.regSet[topic] = struct{}{}
		}
	}
	// Iterate over the set of all topics and look up the next suitable one
	for len(s.regQueue) > 0 {
		// Fetch the next topic from the queue, and ensure it still exists
		topic := s.regQueue[0]
		s.regQueue = s.regQueue[1:]
		delete(s.regSet, topic)

		if s.tickets[topic] == nil {
			continue
		}
		// If the topic needs more tickets, return it
		if s.tickets[topic].nextLookup < mclock.Now() {
			next, delay := s.radius[topic].nextTarget(false), 100*time.Millisecond
			log.Trace("Found discovery topic to register", "topic", topic, "target", next.target, "delay", delay)
			return next, delay
		}
	}
	// No registration topics found or all exhausted, sleep
	delay := 40 * time.Second
	log.Trace("No topic found to register", "delay", delay)
	return lookupInfo{}, delay
}

func (s *ticketStore) nextSearchLookup(topic Topic) lookupInfo {
	tr := s.radius[topic]
	target := tr.nextTarget(tr.radiusLookupCnt >= searchForceQuery)
	if target.radiusLookup {
		tr.radiusLookupCnt++
	} else {
		tr.radiusLookupCnt = 0
	}
	return target
}

// ticketsInWindow returns the tickets of a given topic in the registration window.
func (s *ticketStore) ticketsInWindow(topic Topic) []*ticket {
	// Sanity check that the topic still exists before operating on it
	if s.tickets[topic] == nil {
		log.Warn("Listing non-existing discovery tickets", "topic", topic)
		return nil
	}
	// Gather all the tickers in the next time window
	var tickets []*ticket
	buckets := s.tickets[topic].buckets
	for idx := timeBucket(0); idx < timeWindow; idx++ {
		tickets = append(tickets, buckets[s.lastBucketFetched+idx]...)
	}
	log.Trace("Retrieved discovery registration tickets", "topic", topic, "from", s.lastBucketFetched, "tickets", len(tickets))
	return tickets
}

func (s *ticketStore) removeExcessTickets(t Topic) {
	tickets := s.ticketsInWindow(t)
	if len(tickets) <= wantTicketsInWindow {
		return
	}
	sort.Sort(ticketsByWaitTime(tickets))
	for _, r := range tickets[wantTicketsInWindow:] {
		s.removeTicketRef(r)
	}
}

type ticketsByWaitTime []*ticket

// Len is the number of elements in the collection.
func (s ticketsByWaitTime) Len() int {
	return len(s)
}

func (t *ticket) waitTime() mclock.AbsTime {
	return t.regTime - t.issueTime
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (s ticketsByWaitTime) Less(i, j int) bool {
	return s[i].waitTime() < s[j].waitTime()
}

// Swap swaps the elements with indexes i and j.
func (s ticketsByWaitTime) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s *ticketStore) addTicketRef(t *ticket) {
	tickets := s.tickets[t.topic]
	if tickets == nil {
		log.Warn("Adding ticket to non-existent topic", "topic", t.topic)
		return
	}
	bucket := timeBucket(t.regTime / mclock.AbsTime(ticketTimeBucketLen))
	tickets.buckets[bucket] = append(tickets.buckets[bucket], t)
	min := mclock.Now() - mclock.AbsTime(collectFrequency)*maxCollectDebt
	if tickets.nextLookup < min {
		tickets.nextLookup = min
	}
	tickets.nextLookup += mclock.AbsTime(collectFrequency)

	//s.removeExcessTickets(topic)
}

func (s *ticketStore) nextFilteredTicket() (*ticket, time.Duration) {
	now := mclock.Now()
	for {
		ticket, wait := s.nextRegisterableTicket()
		if ticket == nil {
			return ticket, wait
		}
		log.Trace("Found discovery ticket to register", "node", ticket.node, "serial", ticket.serial, "wait", wait)

		regTime := now + mclock.AbsTime(wait)
		topic := ticket.topic
		if s.tickets[topic] != nil && regTime >= s.tickets[topic].nextReg {
			return ticket, wait
		}
		s.removeTicketRef(ticket)
	}
}

func (s *ticketStore) ticketRegistered(t *ticket) {
	now := mclock.Now()

	tickets := s.tickets[t.topic]
	min := now - mclock.AbsTime(registerFrequency)*maxRegisterDebt
	if min > tickets.nextReg {
		tickets.nextReg = min
	}
	tickets.nextReg += mclock.AbsTime(registerFrequency)
	s.tickets[t.topic] = tickets
	s.removeTicketRef(t)
}

// nextRegisterableTicket returns the next ticket that can be used
// to register.
//
// If the returned wait time <= zero the ticket can be used. For a positive
// wait time, the caller should requery the next ticket later.
//
// A ticket can be returned more than once with <= zero wait time in case
// the ticket contains multiple topics.
func (s *ticketStore) nextRegisterableTicket() (*ticket, time.Duration) {
	now := mclock.Now()
	if s.nextTicketCached != nil {
		return s.nextTicketCached, time.Duration(s.nextTicketCached.regTime - now)
	}

	for bucket := s.lastBucketFetched; ; bucket++ {
		var (
			empty      = true  // true if there are no tickets
			nextTicket *ticket // uninitialized if this bucket is empty
		)
		for _, tickets := range s.tickets {
			//s.removeExcessTickets(topic)
			if len(tickets.buckets) != 0 {
				empty = false

				list := tickets.buckets[bucket]
				for _, ticket := range list {
					//debugLog(fmt.Sprintf(" nrt bucket = %d node = %x sn = %v wait = %v", bucket, ref.t.node.ID[:8], ref.t.serial, time.Duration(ref.topicRegTime()-now)))
					if nextTicket == nil || ticket.regTime < nextTicket.regTime {
						nextTicket = ticket
					}
				}
			}
		}
		if empty {
			return nil, 0
		}
		if nextTicket != nil {
			s.nextTicketCached = nextTicket
			return nextTicket, time.Duration(nextTicket.regTime - now)
		}
		s.lastBucketFetched = bucket
	}
}

// removeTicket removes a ticket from the ticket store
func (s *ticketStore) removeTicketRef(t *ticket) {
	log.Trace("Removing discovery ticket", "node", t.node.ID(), "serial", t.serial)

	// Make nextRegisterableTicket return the next available ticket.
	s.nextTicketCached = nil

	tickets := s.tickets[t.topic]
	if tickets == nil {
		log.Trace("Removing tickets from unknown topic", "topic", t.topic)
		return
	}
	bucket := timeBucket(t.regTime / mclock.AbsTime(ticketTimeBucketLen))
	list := tickets.buckets[bucket]
	idx := -1
	for i, bt := range list {
		if bt == t {
			idx = i
			break
		}
	}
	if idx == -1 {
		panic(nil)
	}
	list = append(list[:idx], list[idx+1:]...)
	if len(list) != 0 {
		tickets.buckets[bucket] = list
	} else {
		delete(tickets.buckets, bucket)
	}
	delete(s.nodes, t.node.ID())
	delete(s.nodeLastReq, t.node.ID())
}

type lookupInfo struct {
	target       enode.ID
	topic        Topic
	radiusLookup bool
}

type reqInfo struct {
	pingHash []byte
	lookup   lookupInfo
	time     mclock.AbsTime
}

func (s *ticketStore) registerLookupDone(lookup lookupInfo, nodes []*enode.Node, ping func(n *enode.Node) []byte) {
	now := mclock.Now()
	for i, n := range nodes {
		id := n.ID()
		if i == 0 || (binary.BigEndian.Uint64(id[:8])^binary.BigEndian.Uint64(lookup.target[:8])) < s.radius[lookup.topic].minRadius {
			if lookup.radiusLookup {
				if lastReq, ok := s.nodeLastReq[id]; !ok || time.Duration(now-lastReq.time) > radiusTC {
					s.nodeLastReq[id] = reqInfo{pingHash: ping(n), lookup: lookup, time: now}
				}
			} else {
				if s.nodes[id] == nil {
					s.nodeLastReq[id] = reqInfo{pingHash: ping(n), lookup: lookup, time: now}
				}
			}
		}
	}
}

func (s *ticketStore) searchLookupDone(lookup lookupInfo, nodes []*enode.Node, query func(n *enode.Node, topic Topic) []byte) {
	now := mclock.Now()
	for i, n := range nodes {
		id := n.ID()
		if i == 0 || (binary.BigEndian.Uint64(id[:8])^binary.BigEndian.Uint64(lookup.target[:8])) < s.radius[lookup.topic].minRadius {
			if lookup.radiusLookup {
				if lastReq, ok := s.nodeLastReq[id]; !ok || time.Duration(now-lastReq.time) > radiusTC {
					s.nodeLastReq[id] = reqInfo{pingHash: nil, lookup: lookup, time: now}
				}
			} // else {
			if s.canQueryTopic(n, lookup.topic) {
				hash := query(n, lookup.topic)
				if hash != nil {
					s.addTopicQuery(common.BytesToHash(hash), n, lookup)
				}
			}
			//}
		}
	}
}

func (s *ticketStore) adjustWithTicket(now mclock.AbsTime, targetHash enode.ID, t *ticket) {
	if tt, ok := s.radius[t.topic]; ok {
		tt.adjustWithTicket(now, targetHash, t)
	}
}

func (s *ticketStore) addTicket(localTime mclock.AbsTime, pingHash []byte, ticket *ticket) {
	log.Trace("Adding discovery ticket", "node", ticket.node.ID, "serial", ticket.serial)

	lastReq, ok := s.nodeLastReq[ticket.node.ID()]
	if !(ok && bytes.Equal(pingHash, lastReq.pingHash)) {
		return
	}
	s.adjustWithTicket(localTime, lastReq.lookup.target, ticket)

	if lastReq.lookup.radiusLookup || s.nodes[ticket.node.ID()] != nil {
		return
	}
	if ticket.topic != lastReq.lookup.topic {
		return
	}

	bucket := timeBucket(localTime / mclock.AbsTime(ticketTimeBucketLen))
	if s.lastBucketFetched == 0 || bucket < s.lastBucketFetched {
		s.lastBucketFetched = bucket
	}

	if _, ok := s.tickets[ticket.topic]; ok {
		wait := ticket.regTime - localTime
		rnd := rand.ExpFloat64()
		if rnd > 10 {
			rnd = 10
		}
		if float64(wait) < float64(keepTicketConst)+float64(keepTicketExp)*rnd {
			// use the ticket to register this topic
			//fmt.Println("addTicket", ticket.node.ID[:8], ticket.node.addr().String(), ticket.serial, ticket.pong)
			s.addTicketRef(ticket)
		}
	}
	s.nextTicketCached = nil
	s.nodes[ticket.node.ID()] = ticket
}

func (s *ticketStore) getNodeTicket(node *enode.Node) *ticket {
	if s.nodes[node.ID()] == nil {
		log.Trace("Retrieving node ticket", "node", node.ID, "serial", nil)
	} else {
		log.Trace("Retrieving node ticket", "node", node.ID, "serial", s.nodes[node.ID()].serial)
	}
	return s.nodes[node.ID()]
}

func (s *ticketStore) canQueryTopic(node *enode.Node, topic Topic) bool {
	qq := s.queriesSent[node.ID()]
	if qq != nil {
		now := mclock.Now()
		for _, sq := range qq {
			if sq.lookup.topic == topic && sq.sent > now-mclock.AbsTime(topicQueryResend) {
				return false
			}
		}
	}
	return true
}

func (s *ticketStore) addTopicQuery(hash common.Hash, node *enode.Node, lookup lookupInfo) {
	now := mclock.Now()
	qq := s.queriesSent[node.ID()]
	if qq == nil {
		qq = make(map[common.Hash]sentQuery)
		s.queriesSent[node.ID()] = qq
	}
	qq[hash] = sentQuery{sent: now, lookup: lookup}
	s.cleanupTopicQueries(now)
}

func (s *ticketStore) cleanupTopicQueries(now mclock.AbsTime) {
	if s.nextTopicQueryCleanup > now {
		return
	}
	exp := now - mclock.AbsTime(topicQueryResend)
	for n, qq := range s.queriesSent {
		for h, q := range qq {
			if q.sent < exp {
				delete(qq, h)
			}
		}
		if len(qq) == 0 {
			delete(s.queriesSent, n)
		}
	}
	s.nextTopicQueryCleanup = now + mclock.AbsTime(topicQueryTimeout)
}

func (s *ticketStore) gotTopicNodes(from *node, hash common.Hash, nodes []*enode.Node) (timeout bool) {
	now := mclock.Now()
	//fmt.Println("got", from.addr().String(), hash, len(nodes))
	qq := s.queriesSent[from.ID()]
	if qq == nil {
		return true
	}
	q, ok := qq[hash]
	if !ok || now > q.sent+mclock.AbsTime(topicQueryTimeout) {
		return true
	}
	inside := float64(0)
	if len(nodes) > 0 {
		inside = 1
	}
	s.radius[q.lookup.topic].adjust(now, q.lookup.target, from.ID(), inside)
	chn := s.searchTopicMap[q.lookup.topic].foundChn
	if chn == nil {
		//fmt.Println("no channel")
		return false
	}
	for _, node := range nodes {
		select {
		case chn <- node:
		default:
			return false
		}
	}
	return false
}
