// Copyright 2014 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.  See the License for the specific language governing
// permissions and limitations under the License. See the AUTHORS file
// for names of contributors.
//
// Author: Tobias Schottdorf (tobias.schottdorf@gmail.com)

// Package hlc implements the Hybrid Logical Clock outlined in
// "Logical Physical Clocks and Consistent Snapshots in Globally
// Distributed Databases", available online at
// http://www.cse.buffalo.edu/tech-reports/2014-04.pdf.
package hlc

import (
	"math"
	"sync"
	"time"

	"github.com/cockroachdb/cockroach/util"
)

// TODO(Tobias): Figure out if it would make sense to save some
// history of the physical clock and react if it jumps backwards
// repeatedly. This is expected during NTP updates, but may
// indicate a broken clock in some cases.

// Timestamp represents a state of the hybrid
// logical clock.
type Timestamp struct {
	// Holds a wall time, typically a unix epoch time
	// expressed in nanoseconds.
	WallTime int64
	// The logical component captures causality for
	// events whose wall times are equal. It is
	// effectively bounded by
	// (maximum clock skew)/(minimal ns between events)
	// and nearly impossible to overflow.
	Logical int64
}

// Timestamp constant values.
var (
	// MaxTimestamp is the max value allowed for Timestamp.
	MaxTimestamp = Timestamp{WallTime: math.MaxInt64, Logical: math.MaxInt64}
	// MinTimestamp is the min value allowed for Timestamp.
	MinTimestamp = Timestamp{WallTime: 0, Logical: 0}
)

// Less implements the util.Ordered interface, allowing
// the comparison of timestamps.
func (t Timestamp) Less(s Timestamp) bool {
	return t.WallTime < s.WallTime || (t.WallTime == s.WallTime && t.Logical < s.Logical)
}

// Equal returns whether two timestamps are the same.
func (t Timestamp) Equal(s Timestamp) bool {
	return t.WallTime == s.WallTime && t.Logical == s.Logical
}

// Clock is a hybrid logical clock. Objects of this
// type model causality while maintaining a relation
// to physical time. Roughly speaking, timestamps
// consist of the largest wall clock time among all
// events, and a logical clock that ticks whenever
// an event happens in the future of the local physical
// clock.
// The data structure is thread safe and thus can safely
// be shared by multiple goroutines.
//
// See NewClock for details.
type Clock struct {
	physicalClock func() int64
	// Clock contains a mutex used to lock the below
	// fields while methods operate on them.
	sync.Mutex
	state Timestamp
	// maxDrift specifies how far ahead of the physical
	// clock the wall time can be.
	// See SetMaxDrift.
	maxDrift time.Duration
}

// ManualClock is a convenience type to facilitate
// creating a hybrid logical clock whose physical clock
// is manually controlled.
type ManualClock int64

// UnixNano returns the underlying manual clock's timestamp.
func (m *ManualClock) UnixNano() int64 {
	return int64(*m)
}

// UnixNano returns the local machine's physical nanosecond
// unix epoch timestamp as a convenience to create a HLC via
// c := hlc.NewClock(hlc.UnixNano).
func UnixNano() int64 {
	return time.Now().UnixNano()
}

// NewClock creates a new hybrid logical clock associated
// with the given physical clock, initializing both wall time
// and logical time with zero.
//
// The physical clock is typically given by the wall time
// of the local machine in unix epoch nanoseconds, using
// hlc.UnixNano. This is not a requirement.
func NewClock(physicalClock func() int64) *Clock {
	return &Clock{
		physicalClock: physicalClock,
	}
}

// SetMaxDrift sets the maximal drift from the physical clock that a
// call to Update may cause. A well-chosen value is large enough to
// ignore a reasonable amount of clock skew but will prevent
// ill-configured nodes from dramatically skewing the wall time of the
// clock into the future.
//
// A value of zero disables this safety feature.
// The default value for a new instance is zero.
func (c *Clock) SetMaxDrift(delta time.Duration) {
	c.Lock()
	defer c.Unlock()
	c.maxDrift = delta
}

// MaxDrift returns the maximal drift allowed.
// A value of 0 means drift checking is disabled.
// See SetMaxDrift for details.
func (c *Clock) MaxDrift() time.Duration {
	c.Lock()
	defer c.Unlock()
	return c.maxDrift
}

// Timestamp returns a copy of the clock's current timestamp,
// without performing a clock adjustment.
func (c *Clock) Timestamp() Timestamp {
	c.Lock()
	defer c.Unlock()
	return c.timestamp()
}

// timestamp returns the state as a timestamp, without
// a lock on the clock's state, for internal usage.
func (c *Clock) timestamp() Timestamp {
	return Timestamp{
		WallTime: c.state.WallTime,
		Logical:  c.state.Logical,
	}
}

// Now returns a timestamp associated with an event from
// the local machine that may be sent to other members
// of the distributed network. This is the counterpart
// of Update, which is passed a timestamp received from
// another member of the distributed network.
func (c *Clock) Now() (result Timestamp) {
	c.Lock()
	defer c.Unlock()
	defer func() {
		result = c.timestamp()
	}()

	physicalClock := c.physicalClock()
	if c.state.WallTime >= physicalClock {
		// The wall time is ahead, so the logical clock ticks.
		c.state.Logical++
	} else {
		// Use the physical clock, and reset the logical one.
		c.state.WallTime = physicalClock
		c.state.Logical = 0
	}
	return
}

// Update takes a hybrid timestamp, usually originating from
// an event received from another member of a distributed
// system. The clock is updated and the hybrid timestamp
// associated to the receipt of the event returned.
// An error may only occur if drift checking is active and
// the remote timestamp was rejected due to clock drift,
// in which case the state of the clock will not have been
// altered.
// To timestamp events of local origin, use Now instead.
func (c *Clock) Update(rt Timestamp) (result Timestamp, err error) {
	c.Lock()
	defer c.Unlock()
	defer func() {
		result = c.timestamp()
	}()
	physicalClock := c.physicalClock()

	if physicalClock > c.state.WallTime && physicalClock > rt.WallTime {
		// Our physical clock is ahead of both wall times. It is used
		// as the new wall time and the logical clock is reset.
		c.state.WallTime = physicalClock
		c.state.Logical = 0
		return
	}

	// In the remaining cases, our physical clock plays no role
	// as it is behind the local and remote wall times. Instead,
	// the logical clock comes into play.
	if rt.WallTime > c.state.WallTime {
		if c.maxDrift.Nanoseconds() > 0 &&
			rt.WallTime-physicalClock > c.maxDrift.Nanoseconds() {
			// The remote wall time is too far ahead to be trustworthy.
			err = util.Errorf("Remote wall time drifts from local physical clock: %d (%dns ahead)",
				rt.WallTime, rt.WallTime-physicalClock)
			return
		}
		// The remote clock is ahead of ours, and we update
		// our own logical clock with theirs.
		c.state.WallTime = rt.WallTime
		c.state.Logical = rt.Logical + 1
	} else if c.state.WallTime > rt.WallTime {
		// Our wall time is larger, so it remains but we tick
		// the logical clock.
		c.state.Logical++
	} else {
		// Both wall times are equal, and the larger logical
		// clock is used for the update.
		if rt.Logical > c.state.Logical {
			c.state.Logical = rt.Logical
		}
		c.state.Logical++
	}
	// The variable result will be updated via defer just
	// before the object is unlocked.
	return
}
