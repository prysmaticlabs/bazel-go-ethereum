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
	"container/heap"
	"math"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

const (
	maxEntries         = 10000
	maxEntriesPerTopic = 50

	fallbackRegistrationExpiry = 1 * time.Hour
)

type Topic string

type topicEntry struct {
	topic   Topic
	fifoIdx uint64
	node    *enode.Node
	expire  mclock.AbsTime
}

type topicInfo struct {
	entries            map[uint64]*topicEntry
	fifoHead, fifoTail uint64
	rqItem             *topicRequestQueueItem
	wcl                waitControlLoop
}

// removes tail element from the fifo
func (t *topicInfo) getFifoTail() *topicEntry {
	for t.entries[t.fifoTail] == nil {
		t.fifoTail++
	}
	tail := t.entries[t.fifoTail]
	t.fifoTail++
	return tail
}

type nodeInfo struct {
	entries                          map[Topic]*topicEntry
	lastIssuedTicket, lastUsedTicket uint32
	// you can't register a ticket newer than lastUsedTicket before noRegUntil (absolute time)
	noRegUntil mclock.AbsTime
}

type topicTable struct {
	db                    *enode.DB
	self                  *enode.LocalNode
	nodes                 map[enode.ID]*nodeInfo
	topics                map[Topic]*topicInfo
	globalEntries         uint64
	requested             topicRequestQueue
	requestCnt            uint64
	lastGarbageCollection mclock.AbsTime
}

func newTopicTable(self *enode.LocalNode) *topicTable {
	return &topicTable{
		self:   self,
		db:     self.Database(),
		nodes:  make(map[enode.ID]*nodeInfo),
		topics: make(map[Topic]*topicInfo),
	}
}

func (t *topicTable) getOrNewTopic(topic Topic) *topicInfo {
	ti := t.topics[topic]
	if ti == nil {
		rqItem := &topicRequestQueueItem{
			topic:    topic,
			priority: t.requestCnt,
		}
		ti = &topicInfo{
			entries: make(map[uint64]*topicEntry),
			rqItem:  rqItem,
		}
		t.topics[topic] = ti
		heap.Push(&t.requested, rqItem)
	}
	return ti
}

func (t *topicTable) checkDeleteTopic(topic Topic) {
	ti := t.topics[topic]
	if ti == nil {
		return
	}
	if len(ti.entries) == 0 && ti.wcl.hasMinimumWaitPeriod() {
		delete(t.topics, topic)
		heap.Remove(&t.requested, ti.rqItem.index)
	}
}

func (t *topicTable) getOrNewNode(id enode.ID) *nodeInfo {
	n := t.nodes[id]
	if n == nil {
		//fmt.Printf("newNode %016x %016x\n", t.self.sha[:8], node.sha[:8])
		var issued, used uint32
		if t.db != nil {
			issued, used = t.db.TopicRegTicketsV5(id)
		}
		n = &nodeInfo{
			entries:          make(map[Topic]*topicEntry),
			lastIssuedTicket: issued,
			lastUsedTicket:   used,
		}
		t.nodes[id] = n
	}
	return n
}

func (t *topicTable) checkDeleteNode(node enode.ID) {
	if n, ok := t.nodes[node]; ok && len(n.entries) == 0 && n.noRegUntil < mclock.Now() {
		//fmt.Printf("deleteNode %016x %016x\n", t.self.sha[:8], node.sha[:8])
		delete(t.nodes, node)
	}
}

func (t *topicTable) storeTicketCounters(id enode.ID) {
	n := t.getOrNewNode(id)
	if t.db != nil {
		t.db.UpdateTopicRegTicketsV5(id, n.lastIssuedTicket, n.lastUsedTicket)
	}
}

func (t *topicTable) getEntries(topic Topic) []*enode.Node {
	t.collectGarbage()

	te := t.topics[topic]
	if te == nil {
		return nil
	}
	nodes := make([]*enode.Node, len(te.entries))
	i := 0
	for _, e := range te.entries {
		nodes[i] = e.node
		i++
	}
	t.requestCnt++
	t.requested.update(te.rqItem, t.requestCnt)
	return nodes
}

func (t *topicTable) addEntry(node *enode.Node, topic Topic) {
	n := t.getOrNewNode(node.ID())
	// clear previous entries by the same node
	for _, e := range n.entries {
		t.deleteEntry(e)
	}
	// ***
	n = t.getOrNewNode(node.ID())

	tm := mclock.Now()
	te := t.getOrNewTopic(topic)

	if len(te.entries) == maxEntriesPerTopic {
		t.deleteEntry(te.getFifoTail())
	}

	if t.globalEntries == maxEntries {
		t.deleteEntry(t.leastRequested()) // not empty, no need to check for nil
	}

	fifoIdx := te.fifoHead
	te.fifoHead++
	entry := &topicEntry{
		topic:   topic,
		fifoIdx: fifoIdx,
		node:    node,
		expire:  tm + mclock.AbsTime(fallbackRegistrationExpiry),
	}
	te.entries[fifoIdx] = entry
	n.entries[topic] = entry
	t.globalEntries++
	te.wcl.registered(tm)
}

// removes least requested element from the fifo
func (t *topicTable) leastRequested() *topicEntry {
	for t.requested.Len() > 0 && t.topics[t.requested[0].topic] == nil {
		heap.Pop(&t.requested)
	}
	if t.requested.Len() == 0 {
		return nil
	}
	return t.topics[t.requested[0].topic].getFifoTail()
}

// deleteEntry removes an entry. The entry must exist.
func (t *topicTable) deleteEntry(e *topicEntry) {
	ne := t.nodes[e.node.ID()].entries
	delete(ne, e.topic)
	if len(ne) == 0 {
		t.checkDeleteNode(e.node.ID())
	}
	te := t.topics[e.topic]
	delete(te.entries, e.fifoIdx)
	if len(te.entries) == 0 {
		t.checkDeleteTopic(e.topic)
	}
	t.globalEntries--
}

// It is assumed that topics and waitPeriods have the same length.
func (t *topicTable) useTicket(node *enode.Node, serialNo uint32, topic Topic, issueTime uint64, waitPeriod uint32) (registered bool) {
	log.Trace("Using discovery ticket", "serial", serialNo, "topic", topic, "waitp", waitPeriod)
	//fmt.Println("useTicket", serialNo, topics, waitPeriods)
	t.collectGarbage()

	n := t.getOrNewNode(node.ID())
	if serialNo < n.lastUsedTicket {
		return false
	}

	tm := mclock.Now()
	if serialNo > n.lastUsedTicket && tm < n.noRegUntil {
		return false
	}
	if serialNo != n.lastUsedTicket {
		n.lastUsedTicket = serialNo
		n.noRegUntil = tm + mclock.AbsTime(noRegTimeout())
		t.storeTicketCounters(node.ID())
	}

	currTime := uint64(tm / mclock.AbsTime(time.Second))
	regTime := issueTime + uint64(waitPeriod)
	relTime := int64(currTime - regTime)
	if relTime >= -1 && relTime <= regTimeWindow+1 { // give clients a little security margin on both ends
		if e := n.entries[topic]; e == nil {
			t.addEntry(node, topic)
		} else {
			// if there is an active entry, don't move to the front of the FIFO but prolong expire time
			e.expire = tm + mclock.AbsTime(fallbackRegistrationExpiry)
		}
		return true
	}

	return false
}

func (t *topicTable) getTicket(id enode.ID, topic Topic) *ticket {
	t.collectGarbage()

	now := mclock.Now()
	n := t.getOrNewNode(id)
	n.lastIssuedTicket++
	t.storeTicketCounters(id)

	ticket := &ticket{
		issueTime: now,
		topic:     topic,
		serial:    n.lastIssuedTicket,
	}
	waitPeriod := minWaitPeriod
	if topic := t.topics[topic]; topic != nil {
		waitPeriod = topic.wcl.waitPeriod
	}
	ticket.regTime = now + mclock.AbsTime(waitPeriod)
	return ticket
}

const gcInterval = time.Minute

func (t *topicTable) collectGarbage() {
	tm := mclock.Now()
	if time.Duration(tm-t.lastGarbageCollection) < gcInterval {
		return
	}
	t.lastGarbageCollection = tm

	for node, n := range t.nodes {
		for _, e := range n.entries {
			if e.expire <= tm {
				t.deleteEntry(e)
			}
		}

		t.checkDeleteNode(node)
	}

	for topic := range t.topics {
		t.checkDeleteTopic(topic)
	}
}

const (
	minWaitPeriod   = time.Minute
	regTimeWindow   = 10 // seconds
	avgnoRegTimeout = time.Minute * 10
	// target average interval between two incoming ad requests
	wcTargetRegInterval = time.Minute * 10 / maxEntriesPerTopic
	//
	wcTimeConst = time.Minute * 10
)

// initialization is not required, will set to minWaitPeriod at first registration
type waitControlLoop struct {
	lastIncoming mclock.AbsTime
	waitPeriod   time.Duration
}

func (w *waitControlLoop) registered(tm mclock.AbsTime) {
	w.waitPeriod = w.nextWaitPeriod(tm)
	w.lastIncoming = tm
}

func (w *waitControlLoop) nextWaitPeriod(tm mclock.AbsTime) time.Duration {
	period := tm - w.lastIncoming
	wp := time.Duration(float64(w.waitPeriod) * math.Exp((float64(wcTargetRegInterval)-float64(period))/float64(wcTimeConst)))
	if wp < minWaitPeriod {
		wp = minWaitPeriod
	}
	return wp
}

func (w *waitControlLoop) hasMinimumWaitPeriod() bool {
	return w.nextWaitPeriod(mclock.Now()) == minWaitPeriod
}

func noRegTimeout() time.Duration {
	e := rand.ExpFloat64()
	if e > 100 {
		e = 100
	}
	return time.Duration(float64(avgnoRegTimeout) * e)
}

type topicRequestQueueItem struct {
	topic    Topic
	priority uint64
	index    int
}

// A topicRequestQueue implements heap.Interface and holds topicRequestQueueItems.
type topicRequestQueue []*topicRequestQueueItem

func (tq topicRequestQueue) Len() int { return len(tq) }

func (tq topicRequestQueue) Less(i, j int) bool {
	return tq[i].priority < tq[j].priority
}

func (tq topicRequestQueue) Swap(i, j int) {
	tq[i], tq[j] = tq[j], tq[i]
	tq[i].index = i
	tq[j].index = j
}

func (tq *topicRequestQueue) Push(x interface{}) {
	n := len(*tq)
	item := x.(*topicRequestQueueItem)
	item.index = n
	*tq = append(*tq, item)
}

func (tq *topicRequestQueue) Pop() interface{} {
	old := *tq
	n := len(old)
	item := old[n-1]
	item.index = -1
	*tq = old[0 : n-1]
	return item
}

func (tq *topicRequestQueue) update(item *topicRequestQueueItem, priority uint64) {
	item.priority = priority
	heap.Fix(tq, item.index)
}
