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

package util

import (
	"fmt"
	"testing"
	"time"
)

func TestRetry(t *testing.T) {
	opts := RetryOptions{"test", time.Microsecond * 10, time.Second, 2, 10}
	var retries int
	err := RetryWithBackoff(opts, func() (bool, error) {
		retries++
		if retries >= 3 {
			return true, nil
		}
		return false, nil
	})
	if err != nil || retries != 3 {
		t.Error("expected 3 retries, got", retries, ":", err)
	}
}

func TestRetryExceedsMaxBackoff(t *testing.T) {
	timer := time.AfterFunc(time.Second, func() {
		t.Error("max backoff not respected")
	})
	opts := RetryOptions{"test", time.Microsecond * 10, time.Microsecond * 10, 1000, 3}
	err := RetryWithBackoff(opts, func() (bool, error) {
		return false, nil
	})
	if err == nil {
		t.Error("should receive max attempts error on retry")
	}
	timer.Stop()
}

func TestRetryExceedsMaxAttempts(t *testing.T) {
	var retries int
	opts := RetryOptions{"test", time.Microsecond * 10, time.Second, 2, 3}
	err := RetryWithBackoff(opts, func() (bool, error) {
		retries++
		return false, nil
	})
	if err == nil || retries != 3 {
		t.Error("expected 3 retries, got", retries, ":", err)
	}
}

func TestRetryFunctionReturnsError(t *testing.T) {
	opts := RetryOptions{"test", time.Microsecond * 10, time.Second, 2, 0 /* indefinite */}
	err := RetryWithBackoff(opts, func() (bool, error) {
		return false, fmt.Errorf("something went wrong")
	})
	if err == nil {
		t.Error("expected an error")
	}
}
