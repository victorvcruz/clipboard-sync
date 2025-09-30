package main

import (
	"cp-sync/internal/clipboard"
	"cp-sync/internal/conn"
	"log"
)

func main() {

	err := clipboard.Init()
	if err != nil {
		log.Fatal(err)
	}

	conn.StartServer()

	//conn.StartClient("192.168.31.93")
}
