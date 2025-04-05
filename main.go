package main

import (
	"cachego/server"
	"fmt"
	"os"
)

func main() {
	fmt.Println("CacheGo Server v1.0")
	if len(os.Args) == 3 {
		skt, dur := os.Args[1], os.Args[2]
		server.Start(skt, dur)
	} else {
		server.Start(readEnvVars())
	}
	server.WtGrp.Wait()
}

func readEnvVars() (string, string) {
	skt, ok := os.LookupEnv("serversocket")
	if !ok {
		os.Exit(1)
	}

	dur := os.Getenv("defaultduration")

	return skt, dur
}
