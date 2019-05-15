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
	"encoding/binary"
	"math"
	"math/rand"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

// topicRadius estimates the radius of a topic.
type topicRadius struct {
	topic             Topic
	topicHashPrefix   uint64
	radius, minRadius uint64
	buckets           []topicRadiusBucket
	converged         bool
	radiusLookupCnt   int
	rand              *rand.Rand
}

type topicRadiusEvent int

const (
	trOutside topicRadiusEvent = iota
	trInside
	trNoAdjust
	trCount
)

type topicRadiusBucket struct {
	weights    [trCount]float64
	lastTime   mclock.AbsTime
	value      float64
	lookupSent map[enode.ID]mclock.AbsTime
}

func (b *topicRadiusBucket) update(now mclock.AbsTime) {
	if now == b.lastTime {
		return
	}
	exp := math.Exp(-float64(now-b.lastTime) / float64(radiusTC))
	for i, w := range b.weights {
		b.weights[i] = w * exp
	}
	b.lastTime = now

	for target, tm := range b.lookupSent {
		if now-tm > mclock.AbsTime(respTimeout) {
			b.weights[trNoAdjust] += 1
			delete(b.lookupSent, target)
		}
	}
}

func (b *topicRadiusBucket) adjust(now mclock.AbsTime, inside float64) {
	b.update(now)
	if inside <= 0 {
		b.weights[trOutside] += 1
	} else {
		if inside >= 1 {
			b.weights[trInside] += 1
		} else {
			b.weights[trInside] += inside
			b.weights[trOutside] += 1 - inside
		}
	}
}

func newTopicRadius(t Topic) *topicRadius {
	topicHash := crypto.Keccak256Hash([]byte(t))
	topicHashPrefix := binary.BigEndian.Uint64(topicHash[0:8])

	return &topicRadius{
		topic:           t,
		topicHashPrefix: topicHashPrefix,
		radius:          maxRadius,
		minRadius:       maxRadius,
		rand:            rand.New(rand.NewSource(1)), // TODO seed
	}
}

func (r *topicRadius) getBucketIdx(id enode.ID) int {
	prefix := binary.BigEndian.Uint64(id[0:8])
	var log2 float64
	if prefix != r.topicHashPrefix {
		log2 = math.Log2(float64(prefix ^ r.topicHashPrefix))
	}
	bucket := int((64 - log2) * radiusBucketsPerBit)
	max := 64*radiusBucketsPerBit - 1
	if bucket > max {
		return max
	}
	if bucket < 0 {
		return 0
	}
	return bucket
}

func (r *topicRadius) targetForBucket(bucket int) enode.ID {
	min := math.Pow(2, 64-float64(bucket+1)/radiusBucketsPerBit)
	max := math.Pow(2, 64-float64(bucket)/radiusBucketsPerBit)
	a := uint64(min)
	b := r.randUint64n(uint64(max - min))
	xor := a + b
	if xor < a {
		xor = ^uint64(0)
	}
	prefix := r.topicHashPrefix ^ xor
	var target enode.ID
	binary.BigEndian.PutUint64(target[0:8], prefix)
	r.rand.Read(target[8:])
	return target
}

func (r *topicRadius) randUint64n(n uint64) uint64 {
	return r.rand.Uint64() % n // TODO check implications of modulo bias
}

func (r *topicRadius) isInRadius(addrHash common.Hash) bool {
	nodePrefix := binary.BigEndian.Uint64(addrHash[0:8])
	dist := nodePrefix ^ r.topicHashPrefix
	return dist < r.radius
}

func (r *topicRadius) chooseLookupBucket(a, b int) int {
	if a < 0 {
		a = 0
	}
	if a > b {
		return -1
	}
	c := 0
	for i := a; i <= b; i++ {
		if i >= len(r.buckets) || r.buckets[i].weights[trNoAdjust] < maxNoAdjust {
			c++
		}
	}
	if c == 0 {
		return -1
	}
	rnd := uint32(r.randUint64n(uint64(c)))
	for i := a; i <= b; i++ {
		if i >= len(r.buckets) || r.buckets[i].weights[trNoAdjust] < maxNoAdjust {
			if rnd == 0 {
				return i
			}
			rnd--
		}
	}
	panic(nil) // should never happen
}

func (r *topicRadius) needMoreLookups(a, b int, maxValue float64) bool {
	var max float64
	if a < 0 {
		a = 0
	}
	if b >= len(r.buckets) {
		b = len(r.buckets) - 1
		if r.buckets[b].value > max {
			max = r.buckets[b].value
		}
	}
	if b >= a {
		for i := a; i <= b; i++ {
			if r.buckets[i].value > max {
				max = r.buckets[i].value
			}
		}
	}
	return maxValue-max < minPeakSize
}

func (r *topicRadius) recalcRadius() (radius uint64, radiusLookup int) {
	maxBucket := 0
	maxValue := float64(0)
	now := mclock.Now()
	v := float64(0)
	for i := range r.buckets {
		r.buckets[i].update(now)
		v += r.buckets[i].weights[trOutside] - r.buckets[i].weights[trInside]
		r.buckets[i].value = v
		//fmt.Printf("%v %v | ", v, r.buckets[i].weights[trNoAdjust])
	}
	//fmt.Println()
	slopeCross := -1
	for i, b := range r.buckets {
		v := b.value
		if v < float64(i)*minSlope {
			slopeCross = i
			break
		}
		if v > maxValue {
			maxValue = v
			maxBucket = i + 1
		}
	}

	minRadBucket := len(r.buckets)
	sum := float64(0)
	for minRadBucket > 0 && sum < minRightSum {
		minRadBucket--
		b := r.buckets[minRadBucket]
		sum += b.weights[trInside] + b.weights[trOutside]
	}
	r.minRadius = uint64(math.Pow(2, 64-float64(minRadBucket)/radiusBucketsPerBit))

	lookupLeft := -1
	if r.needMoreLookups(0, maxBucket-lookupWidth-1, maxValue) {
		lookupLeft = r.chooseLookupBucket(maxBucket-lookupWidth, maxBucket-1)
	}
	lookupRight := -1
	if slopeCross != maxBucket && (minRadBucket <= maxBucket || r.needMoreLookups(maxBucket+lookupWidth, len(r.buckets)-1, maxValue)) {
		for len(r.buckets) <= maxBucket+lookupWidth {
			r.buckets = append(r.buckets, topicRadiusBucket{lookupSent: make(map[enode.ID]mclock.AbsTime)})
		}
		lookupRight = r.chooseLookupBucket(maxBucket, maxBucket+lookupWidth-1)
	}
	if lookupLeft == -1 {
		radiusLookup = lookupRight
	} else {
		if lookupRight == -1 {
			radiusLookup = lookupLeft
		} else {
			if r.randUint64n(2) == 0 {
				radiusLookup = lookupLeft
			} else {
				radiusLookup = lookupRight
			}
		}
	}

	//fmt.Println("mb", maxBucket, "sc", slopeCross, "mrb", minRadBucket, "ll", lookupLeft, "lr", lookupRight, "mv", maxValue)

	if radiusLookup == -1 {
		// no more radius lookups needed at the moment, return a radius
		r.converged = true
		rad := maxBucket
		if minRadBucket < rad {
			rad = minRadBucket
		}
		radius = ^uint64(0)
		if rad > 0 {
			radius = uint64(math.Pow(2, 64-float64(rad)/radiusBucketsPerBit))
		}
		r.radius = radius
	}

	return
}

func (r *topicRadius) nextTarget(forceRegular bool) lookupInfo {
	if !forceRegular {
		_, radiusLookup := r.recalcRadius()
		if radiusLookup != -1 {
			target := r.targetForBucket(radiusLookup)
			r.buckets[radiusLookup].lookupSent[target] = mclock.Now()
			return lookupInfo{target: target, topic: r.topic, radiusLookup: true}
		}
	}

	radExt := r.radius / 2
	if radExt > maxRadius-r.radius {
		radExt = maxRadius - r.radius
	}
	rnd := r.randUint64n(r.radius) + r.randUint64n(2*radExt)
	if rnd > radExt {
		rnd -= radExt
	} else {
		rnd = radExt - rnd
	}

	prefix := r.topicHashPrefix ^ rnd
	var target enode.ID
	binary.BigEndian.PutUint64(target[0:8], prefix)
	rand.Read(target[8:])
	return lookupInfo{target: target, topic: r.topic, radiusLookup: false}
}

func (r *topicRadius) adjustWithTicket(now mclock.AbsTime, targetHash enode.ID, t *ticket) {
	wait := t.regTime - t.issueTime
	inside := float64(wait)/float64(targetWaitTime) - 0.5
	if inside > 1 {
		inside = 1
	}
	if inside < 0 {
		inside = 0
	}
	r.adjust(now, targetHash, t.node.ID(), inside)
}

func (r *topicRadius) adjust(now mclock.AbsTime, targetHash, id enode.ID, inside float64) {
	bucket := r.getBucketIdx(id)
	//fmt.Println("adjust", bucket, len(r.buckets), inside)
	if bucket >= len(r.buckets) {
		return
	}
	r.buckets[bucket].adjust(now, inside)
	delete(r.buckets[bucket].lookupSent, id)
}
