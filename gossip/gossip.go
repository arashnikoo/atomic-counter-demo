package gossip

import (
	"dist-counter/config"
	httpapi "dist-counter/http"
	raftmod "dist-counter/raft" // Import the raftmod package
	"dist-counter/utils"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/hashicorp/memberlist"
)

var memberInfoList []MemberInfo
var gossipStarted bool = false
var raftStarted bool = false

type MemberInfo struct {
	Address string
	NodeId  string
}

func StartGossip(appConfig *config.Config) {
	if gossipStarted {
		return
	}

	config := memberlist.DefaultLANConfig()
	config.Name = appConfig.NodeID

	re := regexp.MustCompile("[0-9]+")
	num, _ := strconv.Atoi(re.FindAllString(appConfig.NodeID, -1)[0])
	config.BindPort = 7945 + num

	mlist, err := memberlist.Create(config)
	if err != nil {
		log.Fatalf("gossip init failed: %v", err)
	}

	FindParticipants(mlist)
	log.Printf("[http] http api started with  address: %s", appConfig.HTTPAddr)

	// print the list of the members
	_members := mlist.Members()
	memberInfoList = make([]MemberInfo, len(_members))

	for i, m := range _members {
		memberInfoList[i] = MemberInfo{
			Address: m.Addr.String(),
			NodeId:  m.Name,
		}

		log.Printf("Found %s (%s)", m.Name, m.Addr.String())
	}

	go func() {
		for {
			time.Sleep(10 * time.Second)
			FindParticipants(mlist)

			if !raftStarted {
				joinAddress := GetLeaderJoinAddress()

				appConfig.JoinAddr = joinAddress
				raftSystem, fsmInstance, raftDir := raftmod.SetupRaft(appConfig)
				httpapi.AddRaftHandlers(appConfig, raftSystem, fsmInstance, raftDir)
				raftStarted = true
				if appConfig.JoinAddr != "" {
					log.Printf("[gossip] raft system started with join address: %s", appConfig.JoinAddr)
				} else {
					log.Printf("[gossip] raft system started as the leader")
				}
			}
		}
	}()
}

func FindParticipants(mlist *memberlist.Memberlist) {
	for i := 0; i < 10; i++ {
		addr := fmt.Sprintf("127.0.0.1:%d", 7945+i)
		_, err := mlist.Join([]string{addr})
		if err == nil {
			log.Printf("[gossip] joined %s", addr)
		}
	}
}

func GetMembers() []MemberInfo {
	return memberInfoList
}

func GetLeaderJoinAddress() string {
	nodes := GetMembers()
	if len(nodes) == 1 {
		log.Println("[gossip] this is the only node in the cluster")
		return ""
	}

	client := http.Client{Timeout: 30 * time.Second}

	for _, node := range nodes {
		log.Printf("[gossip] discovering %s for leader", node.NodeId)

		httpAddress := utils.GetHttpAddressFromNodeID(node.NodeId)
		log.Printf("[gossip] found %s with http address %s", node.NodeId, httpAddress)

		httpStatusAddr := fmt.Sprintf("http://%s/status", httpAddress)

		resp, err := client.Get(httpStatusAddr)
		if err != nil {
			log.Printf("[gossip] failed to query %s: %v", httpStatusAddr, err)
			continue
		}

		log.Printf("[gossip] called %s, response status %s", httpStatusAddr, resp.Status)
		var clusterState utils.ClusterState

		//decode leader address from the response
		err = json.NewDecoder(resp.Body).Decode(&clusterState)
		if err != nil {
			log.Printf("[gossip] failed to decode response from %s: %v", httpStatusAddr, err)
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		log.Printf("[gossip] node %s is %s", node.NodeId, clusterState.State)

		// if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		// 	log.Printf("[gossip] failed to decode response from %s: %v", httpStatusAddr, err)
		// 	resp.Body.Close()
		// 	continue
		// }

		// Check if this node is the leader
		if leaderAddr := clusterState.Leader.HttpAddress; leaderAddr != "" {
			log.Printf("[gossip] discovered leader: %s", leaderAddr)
			return leaderAddr
		}
	}

	log.Println("[gossip] no leader found via gossip")
	return ""
}
