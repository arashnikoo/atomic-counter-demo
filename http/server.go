package httpapi

import (
	utils "dist-counter/utils"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"dist-counter/config"
	raftmod "dist-counter/raft"
	"dist-counter/types"

	"github.com/hashicorp/raft"
)

func StartServer(cfg *config.Config) {

	log.Printf("[%s] HTTP API listening on %s", cfg.NodeID, cfg.HTTPAddr)
	http.ListenAndServe(cfg.HTTPAddr, nil)
}

func AddRaftHandlers(cfg *config.Config, raftNode *raft.Raft, fsm *raftmod.FSM, raftDir string) {

	http.HandleFunc("/next", func(w http.ResponseWriter, r *http.Request) {
		if raftNode.State() != raft.Leader {
			http.Error(w, "not leader", http.StatusBadRequest)
			return
		}
		cmd := types.Command{Op: "increment"}
		b, _ := json.Marshal(cmd)
		resp := raftNode.Apply(b, 5*time.Second)
		if err := resp.Error(); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		fmt.Fprintf(w, "%v\n", resp.Response())
	})

	http.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		fsmMu := fsm.Mutex()
		fsmMu.Lock()
		defer fsmMu.Unlock()
		json.NewEncoder(w).Encode(map[string]uint64{"counter": fsm.Counter})
	})

	http.HandleFunc("/backup", func(w http.ResponseWriter, r *http.Request) {
		err := raftmod.BackupFSM(fsm)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write([]byte("FSM state backed up"))
	})

	http.HandleFunc("/restore", func(w http.ResponseWriter, r *http.Request) {
		err := raftmod.RestoreFSM(fsm)
		if err != nil {
			http.Error(w, err.Error(), 404)
			return
		}
		w.Write([]byte("FSM state restored from backup"))
		log.Printf("Current counter: %d", fsm.Counter)

		raftNode.Snapshot()
	})

	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		cfgFuture := raftNode.GetConfiguration()
		var clusterState utils.ClusterState

		leader, nodeId := raftNode.LeaderWithID()
		leaderNode := utils.Leader{
			RaftAddress: string(leader),
			NodeID:      string(nodeId),
			HttpAddress: utils.GetHttpAddressFromNodeID(string(nodeId)),
		}

		nodes := []utils.Node{}

		if err := cfgFuture.Error(); err == nil {
			for _, srv := range cfgFuture.Configuration().Servers {
				role := "nonvoter"
				if srv.Suffrage == raft.Voter {
					role = "voter"
				}
				nodes = append(nodes, utils.Node{
					ID:      string(srv.ID),
					Address: string(srv.Address),
					Role:    role,
				})
			}
		}

		clusterState = utils.ClusterState{
			LastIndex: raftNode.LastIndex(),
			Leader:    leaderNode,
			Nodes:     nodes,
			State:     raftNode.State().String(),
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(clusterState)
	})

	http.HandleFunc("/join", func(w http.ResponseWriter, r *http.Request) {
		if raftNode.State() != raft.Leader {
			leader := string(raftNode.Leader())
			if leader == "" {
				http.Error(w, "no leader found", http.StatusServiceUnavailable)
				return
			}
			// Forward the join request to the leader
			fwdURL := fmt.Sprintf("http://%s/join?id=%s&addr=%s", leader, r.URL.Query().Get("id"), r.URL.Query().Get("addr"))
			resp, err := http.Get(fwdURL)
			if err != nil || resp.StatusCode != http.StatusOK {
				http.Error(w, "failed to forward join to leader: "+err.Error(), http.StatusBadGateway)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("join forwarded to leader\n"))
			return
		}

		nodeID := r.URL.Query().Get("id")
		nodeAddr := r.URL.Query().Get("addr")
		if nodeID == "" || nodeAddr == "" {
			http.Error(w, "missing params", http.StatusBadRequest)
			return
		}

		// Remove any dead server with same ID first
		configurationFuture := raftNode.GetConfiguration()
		if err := configurationFuture.Error(); err != nil {
			http.Error(w, "failed to get configuration: "+err.Error(), http.StatusInternalServerError)
			return
		}
		for _, srv := range configurationFuture.Configuration().Servers {
			if srv.ID == raft.ServerID(nodeID) && srv.Address != raft.ServerAddress(nodeAddr) {
				raftNode.RemoveServer(srv.ID, 0, 0)
			}
		}
		if raftNode.State() != raft.Leader {
			http.Error(w, "not leader", http.StatusBadRequest)
			return
		}
		if nodeID == "" || nodeAddr == "" {
			http.Error(w, "missing params", http.StatusBadRequest)
			return
		}
		err := raftNode.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(nodeAddr), 0, 0).Error()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		w.WriteHeader(200)
	})
}
