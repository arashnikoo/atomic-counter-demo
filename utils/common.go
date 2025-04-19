package utils

import (
	"fmt"
	"regexp"
	"strconv"
)

type Leader struct {
	RaftAddress string `json:"raftAddress"`
	HttpAddress string `json:"httpAddress"`
	NodeID      string `json:"nodeID"`
}

type Node struct {
	Address string `json:"address"`
	ID      string `json:"id"`
	Role    string `json:"role"`
}

type ClusterState struct {
	LastIndex uint64 `json:"lastIndex"`
	Leader    Leader `json:"leader"`
	Nodes     []Node `json:"nodes"`
	State     string `json:"state"`
}

func GetHttpAddressFromNodeID(nodeID string) string {
	re := regexp.MustCompile("[0-9]+")
	matches := re.FindAllString(nodeID, -1)
	if len(matches) == 0 {
		return ""
	}

	nodeIDInt, err := strconv.Atoi(matches[0])
	if err != nil {
		return ""
	}

	nodeOffset := (nodeIDInt - 1) * 2
	httpPort := 9000 + nodeOffset

	httpAddr := fmt.Sprintf("127.0.0.1:%d", httpPort)
	return httpAddr
}
