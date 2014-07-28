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
// Author: Shawn Morel (shawn@strangemond.com)

package server

import "net/http"

const (
	// statsKeyPrefix is the root of the RESTful cluster statistics and metrics API.
	statsKeyPrefix = "/_stats/"

	// statsNodesKeyPrefix exposes stats for each nodes of the cluster.
	// GETing statsNodesKeyPrefix will list all nodes.
	// Individual node stats can be queries at statsNodesKeyPrefix/NodeID
	statsNodesKeyPrefix = statsKeyPrefix + "nodes/"

	// statsGossipKeyPrefix exposes a view of the gossip network
	statsGossipKeyPrefix = statsKeyPrefix + "gossip"

	// statsEnginesKeyPrefix exposes stats for each storage engine.
	statsEnginesKeyPrefix = statsKeyPrefix + "engines/"

	// statsTransactionsKeyPrefix exposes transaction statistics
	statsTransactionsKeyPrefix = statsKeyPrefix + "txns/"

	// statsLocalKeyPrefix exposes the stats of the node serving the request.
	// Useful for debuging nodes that aren't communicating with the cluster properly.
	statsLocalKeyPrefix = statsKeyPrefix + "local"
)

// A statsServer provides a RESTful stats API.
type statsServer struct {
}

// newStatsServer allocates and returns a statsServer.
func newStatsServer() *statsServer {
	return &statsServer{}
}

// TODO(shawn) lots of implementing - setting up a skeleton for hack week.

// handleStats handles GET requests for cluster stats.
func (s *statsServer) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("// Cockroach Cluster stats\n"))

	w.Write([]byte(`{}`))
}

// handleNodeStats handles GET requests for node stats.
func (s *statsServer) handleNodeStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("// Cockroach Nodes\n"))

	// TODO(shawn) parse node-id in path

	w.Write([]byte(`{"nodes":[]}`))
}

// handleGossipStats handles GET requests for gossip network stats.
func (s *statsServer) handleGossipStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("// Cockroach gossip network\n"))

	w.Write([]byte(`{}`))
}

// handleEngineStats handles GET requests for engine stats.
func (s *statsServer) handleEngineStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("// Cockroach Engines\n"))

	w.Write([]byte(`{"engines": []}`))
}

// handleTransactionStats handles GET requests for transaction stats.
func (s *statsServer) handleTransactionStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("// Cockroach Transaction stats\n"))

	w.Write([]byte(`{"engines": []}`))
}

// handleLocalStats handles GET requests for local-node stats.
func (s *statsServer) handleLocalStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("// Cockroach Local Node stats\n"))

	w.Write([]byte(`{}`))
}
