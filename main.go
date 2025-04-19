package main

import (
	"dist-counter/config"
	gossip "dist-counter/gossip"
	httpapi "dist-counter/http"
)

func main() {
	cfg := config.ParseFlags()

	gossip.StartGossip(cfg)
	httpapi.StartServer(cfg)

}
