package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
)

func main() {
	address := flag.String("address", "127.0.0.1:25565", "Java server address")
	username := flag.String("username", "GeyserProbe", "offline-mode Java username")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	client, err := javaprotocol.DialAndLogin(ctx, *address, javaprotocol.Java1214, *username, nil)
	if err != nil {
		panic(err)
	}
	defer client.Close()
	fmt.Printf("Java login succeeded: profile=%s protocol=%d username=%s state=%d\n", client.Profile.Name, client.Profile.ProtocolVersion, client.Login.Username, client.Conn.State())
}
