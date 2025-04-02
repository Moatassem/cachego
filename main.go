package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("CacheGo Server v1.0")
	startServer(readEnvVars())
	WtGrp.Wait()
}

func readEnvVars() (string, string) {
	skt, ok := os.LookupEnv("serversocket")
	if !ok {
		os.Exit(1)
	}

	dur := os.Getenv("defaultduration")

	return skt, dur
}
