package main

import (
	"log"

	"github.com/tomialagbe/nustomp"
)

// var serverPort int

// func init() {
// 	flag.IntVar(&serverPort, "port", 8081, "The port the server binds to")
// }

func main() {
	bindPort := 8086
	log.Printf("Starting STOMP Server on port %d", bindPort)

	server := nustomp.NewServer(bindPort)
	// this server expects heartbeats from clients (that support heartbeats)
	// every 30 seconds and guarantees approximately a minute between the heartbeats it sends
	server.SetHeartBeat(30000, 60000)
	server.Start()
}
