package raftmod

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"sync"

	"dist-counter/types"

	"github.com/hashicorp/raft"
)

type FSM struct {
	mu      sync.Mutex
	Counter uint64
}

func NewFSM() *FSM {
	return &FSM{}
}

func (f *FSM) Apply(logEntry *raft.Log) interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()

	var c types.Command
	if err := json.Unmarshal(logEntry.Data, &c); err != nil {
		log.Printf("FSM apply unmarshal error: %v", err)
		return nil
	}

	if c.Op == "increment" {
		f.Counter++
		return f.Counter
	}

	log.Printf("FSM apply unknown command: %s", c.Op)
	return nil
}

func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return &snapshot{Counter: f.Counter}, nil
}

func (f *FSM) Restore(rc io.ReadCloser) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return json.NewDecoder(rc).Decode(&f.Counter)
}

func (f *FSM) Mutex() *sync.Mutex {
	return &f.mu
}

type snapshot struct {
	Counter uint64
}

func (s *snapshot) Persist(sink raft.SnapshotSink) error {
	data, err := json.Marshal(s.Counter)
	if err != nil {
		return err
	}
	if _, err := sink.Write(data); err != nil {
		return err
	}
	return sink.Close()
}

func (s *snapshot) Release() {}

var backupFile = "fsm-backup.json"

func BackupFSM(fsm *FSM) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()
	data, err := json.MarshalIndent(fsm.Counter, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(backupFile, data, 0644)
}

func RestoreFSM(fsm *FSM) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()
	data, err := os.ReadFile(backupFile)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &fsm.Counter)
}

func SynFromLeader(fsm *FSM) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()
	data, err := os.ReadFile(backupFile)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &fsm.Counter)
}
