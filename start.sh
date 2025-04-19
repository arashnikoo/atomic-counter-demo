#!/bin/bash
NODE_ID=$1
OFFSET=$(((NODE_ID - 1)*2))
HTTP_PORT=$((9000 + OFFSET))
RAFT_PORT=$((9000 + OFFSET+1))
go run main.go -id node$NODE_ID -http 127.0.0.1:$HTTP_PORT -raft 127.0.0.1:$RAFT_PORT
