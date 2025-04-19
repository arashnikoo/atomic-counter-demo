package raftmod

import (
	"dist-counter/config"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
)

var (
	singletonFSM     *FSM
	singletonRaft    *raft.Raft
	singletonRaftDir string
)

func SetupRaft(appConfig *config.Config) (*raft.Raft, *FSM, string) {
	if singletonRaft != nil && singletonFSM != nil && singletonRaftDir != "" {
		return singletonRaft, singletonFSM, singletonRaftDir
	}

	fsm := NewFSM()
	hclogger := hclog.New(&hclog.LoggerOptions{
		Name:  "raft",
		Level: hclog.Error,
	})

	cfg := raft.DefaultConfig()
	cfg.LocalID = raft.ServerID(appConfig.NodeID)
	cfg.Logger = hclogger
	cfg.SnapshotInterval = 30 * time.Second
	cfg.SnapshotThreshold = 8

	raftDir := "raft-data/" + appConfig.NodeID
	os.MkdirAll(raftDir, 0700)

	addr, _ := net.ResolveTCPAddr("tcp", appConfig.RaftAddr)
	transport, _ := raft.NewTCPTransport(appConfig.RaftAddr, addr, 3, 10*time.Second, os.Stderr)

	storePath := filepath.Join(raftDir, "raft.db")
	snapshots, _ := raft.NewFileSnapshotStore(raftDir, 10, os.Stderr)
	logStore, _ := raftboltdb.NewBoltStore(storePath)

	raftNode, err := raft.NewRaft(cfg, fsm, logStore, logStore, snapshots, transport)
	if err != nil {
		log.Fatalf("raft init error: %v", err)
	}

	log.Printf("[bootstrap] Joinning leader cluster %s", appConfig.JoinAddr)

	if appConfig.JoinAddr == "" {
		if _, err := os.Stat(storePath); os.IsNotExist(err) {
			// Raft DB doesn't exist → safe to restore FSM state
			//RestoreFSM(fsm)
		} else {
			log.Printf("[bootstrap] Existing raft.db found, skipping FSM restore")
		}

		cfg := raft.Configuration{
			Servers: []raft.Server{{
				ID:      cfg.LocalID,
				Address: transport.LocalAddr(),
			}},
		}
		raftNode.BootstrapCluster(cfg)

	} else {
		joinURL := fmt.Sprintf("http://%s/join?id=%s&addr=%s", appConfig.JoinAddr, appConfig.NodeID, appConfig.RaftAddr)
		log.Printf("Trying to join %s", joinURL)
		resp, err := http.Get(joinURL)
		if err != nil || resp.StatusCode != 200 {
			log.Fatalf("failed to join cluster: %v", err)
		}

		raftNode.Snapshot()
	}

	singletonFSM = fsm
	singletonRaft = raftNode
	singletonRaftDir = raftDir

	return singletonRaft, singletonFSM, singletonRaftDir
}
