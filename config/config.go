package config

import "flag"

type Config struct {
	NodeID   string
	HTTPAddr string
	RaftAddr string
	JoinAddr string
}

func ParseFlags() *Config {
	nodeID := flag.String("id", "node1", "Node ID")
	httpAddr := flag.String("http", ":9000", "HTTP bind address")
	raftAddr := flag.String("raft", ":9001", "Raft bind address")
	joinAddr := flag.String("join", "", "Leader address to join")
	flag.Parse()

	return &Config{
		NodeID:   *nodeID,
		HTTPAddr: *httpAddr,
		RaftAddr: *raftAddr,
		JoinAddr: *joinAddr,
	}
}
