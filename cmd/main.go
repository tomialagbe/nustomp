package main

import (
	"flag"
)

var serverPort int

func init() {
	flag.IntVar(&serverPort, "port", 8081, "The port the server binds to")
}

func main() {

}
