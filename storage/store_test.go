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
// Author: Spencer Kimball (spencer.kimball@gmail.com)

package storage

import (
	"bytes"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/cockroachdb/cockroach/storage/engine"
	"github.com/cockroachdb/cockroach/util/hlc"
)

var testIdent = StoreIdent{
	ClusterID: "cluster",
	NodeID:    1,
	StoreID:   1,
}

// TestStoreInitAndBootstrap verifies store initialization and
// bootstrap.
func TestStoreInitAndBootstrap(t *testing.T) {
	manual := hlc.ManualClock(0)
	clock := hlc.NewClock(manual.UnixNano)
	eng := engine.NewInMem(engine.Attributes{}, 1<<20)
	store := NewStore(clock, eng, nil)
	defer store.Close()

	// Can't init as haven't bootstrapped.
	if err := store.Init(); err == nil {
		t.Error("expected failure init'ing un-bootstrapped store")
	}

	// Bootstrap with a fake ident.
	if err := store.Bootstrap(testIdent); err != nil {
		t.Errorf("error bootstrapping store: %v", err)
	}

	// Try to get 1st range--non-existent.
	if _, err := store.GetRange(1); err == nil {
		t.Error("expected error fetching non-existent range")
	}

	// Create range and fetch.
	if _, err := store.CreateRange(engine.KeyMin, engine.KeyMax, []Replica{}); err != nil {
		t.Errorf("failure to create first range: %v", err)
	}
	if _, err := store.GetRange(1); err != nil {
		t.Errorf("failure fetching 1st range: %v", err)
	}

	// Now, attempt to initialize a store with a now-bootstrapped engine.
	store = NewStore(clock, eng, nil)
	if err := store.Init(); err != nil {
		t.Errorf("failure initializing bootstrapped store: %v", err)
	}
	// 1st range should be available.
	if _, err := store.GetRange(1); err != nil {
		t.Errorf("failure fetching 1st range: %v", err)
	}
}

// TestBootstrapOfNonEmptyStore verifies bootstrap failure if engine
// is not empty.
func TestBootstrapOfNonEmptyStore(t *testing.T) {
	eng := engine.NewInMem(engine.Attributes{}, 1<<20)

	// Put some random garbage into the engine.
	if err := eng.Put(engine.Key("foo"), []byte("bar")); err != nil {
		t.Errorf("failure putting key foo into engine: %v", err)
	}
	manual := hlc.ManualClock(0)
	clock := hlc.NewClock(manual.UnixNano)
	store := NewStore(clock, eng, nil)
	defer store.Close()

	// Can't init as haven't bootstrapped.
	if err := store.Init(); err == nil {
		t.Error("expected failure init'ing un-bootstrapped store")
	}

	// Bootstrap should fail on non-empty engine.
	if err := store.Bootstrap(testIdent); err == nil {
		t.Error("expected bootstrap error on non-empty store")
	}
}

func TestRangeSliceSort(t *testing.T) {
	var rs RangeSlice
	for i := 4; i >= 0; i-- {
		key := engine.Key(fmt.Sprintf("foo%d", i))
		rs = append(rs, &Range{
			Meta: RangeMetadata{
				RangeDescriptor: RangeDescriptor{StartKey: key},
			},
		})
	}

	sort.Sort(rs)
	for i := 0; i < 5; i++ {
		expectedKey := engine.Key(fmt.Sprintf("foo%d", i))
		if !bytes.Equal(rs[i].Meta.StartKey, expectedKey) {
			t.Errorf("Expected %s, got %s", expectedKey, rs[i].Meta.StartKey)
		}
	}
}

// createTestStore creates a test store using an in-memory
// engine. Returns the store clock's manual unix nanos time and the
// store. A single range from key "a" to key "z" is setup in the store
// with a default replica descriptor (i.e. StoreID = 0, RangeID = 1,
// etc.). The caller is responsible for closing the store on exit.
func createTestStore(t *testing.T) (*Store, *hlc.ManualClock) {
	manual := hlc.ManualClock(0)
	clock := hlc.NewClock(manual.UnixNano)
	eng := engine.NewInMem(engine.Attributes{}, 1<<20)
	store := NewStore(clock, eng, nil)
	replica := Replica{RangeID: 1}
	_, err := store.CreateRange(engine.Key("a"), engine.Key("z"), []Replica{replica})
	if err != nil {
		t.Fatal(err)
	}
	return store, &manual
}

// TestStoreExecuteCmd verifies straightforward command execution
// of both a read-only and a read-write command.
func TestStoreExecuteCmd(t *testing.T) {
	store, _ := createTestStore(t)
	defer store.Close()
	args, reply := getArgs("a", 1)

	// Try a successful get request.
	err := store.ExecuteCmd("Get", args, reply)
	if err != nil {
		t.Fatal(err)
	}
}

// TestStoreExecuteCmdUpdateTime verifies that the node clock is updated.
func TestStoreExecuteCmdUpdateTime(t *testing.T) {
	store, _ := createTestStore(t)
	defer store.Close()
	args, reply := getArgs("a", 1)
	args.Timestamp = store.clock.Now()
	args.Timestamp.WallTime += (100 * time.Millisecond).Nanoseconds()
	err := store.ExecuteCmd("Get", args, reply)
	if err != nil {
		t.Fatal(err)
	}
	ts := store.clock.Timestamp()
	if ts.WallTime != args.Timestamp.WallTime || ts.Logical <= args.Timestamp.Logical {
		t.Errorf("expected store clock to advance to %+v; got %+v", args.Timestamp, ts)
	}
}

// TestStoreExecuteCmdWithZeroTime verifies that no timestamp causes
// the command to assume the node's wall time.
func TestStoreExecuteCmdWithZeroTime(t *testing.T) {
	store, mc := createTestStore(t)
	defer store.Close()
	args, reply := getArgs("a", 1)

	// Set clock to time 1.
	*mc = hlc.ManualClock(1)
	err := store.ExecuteCmd("Get", args, reply)
	if err != nil {
		t.Fatal(err)
	}
	// The Logical time will increase over the course of the command
	// execution so we can only rely on comparing the WallTime.
	if reply.Timestamp.WallTime != store.clock.Timestamp().WallTime {
		t.Errorf("expected reply to have store clock time %+v; got %+v",
			store.clock.Timestamp(), reply.Timestamp)
	}
}

// TestStoreExecuteCmdWithClockDrift verifies that if the request
// specifies a timestamp further into the future than the node's
// maximum allowed clock drift, the cmd fails with an error.
func TestStoreExecuteCmdWithClockDrift(t *testing.T) {
	store, mc := createTestStore(t)
	defer store.Close()
	args, reply := getArgs("a", 1)

	// Set clock to time 1.
	*mc = hlc.ManualClock(1)
	// Set clock max drift to 250ms.
	maxDrift := 250 * time.Millisecond
	store.clock.SetMaxDrift(maxDrift)
	// Set args timestamp to exceed max drift.
	args.Timestamp = store.clock.Now()
	args.Timestamp.WallTime += maxDrift.Nanoseconds() + 1
	err := store.ExecuteCmd("Get", args, reply)
	if err == nil {
		t.Error("expected max drift clock error")
	}
}

// TestStoreExecuteCmdBadRange passes a bad range replica.
func TestStoreExecuteCmdBadRange(t *testing.T) {
	store, _ := createTestStore(t)
	defer store.Close()
	// Range is from "a" to "z", so this value should fail.
	args, reply := getArgs("0", 1)
	args.Replica.RangeID = 2
	err := store.ExecuteCmd("Get", args, reply)
	if err == nil {
		t.Error("expected invalid range")
	}
}

// TestStoreExecuteCmdOutOfRange passes a key not contained
// within the range's key range.
func TestStoreExecuteCmdOutOfRange(t *testing.T) {
	store, _ := createTestStore(t)
	defer store.Close()
	// Range is from "a" to "z", so this value should fail.
	args, reply := getArgs("0", 1)
	err := store.ExecuteCmd("Get", args, reply)
	if err == nil {
		t.Error("expected key to be out of range")
	}
}
