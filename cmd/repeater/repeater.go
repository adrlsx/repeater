package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/Escape-Technologies/repeater/internal"
	"github.com/Escape-Technologies/repeater/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var UUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var url = "0.0.0.0:8080"

func main() {
	fmt.Printf("Running Escape repeater version %s, commit %s, built at %s\n", version, commit, date)

	repeaterId := os.Getenv("ESCAPE_REPEATER_ID")
	if !UUID.MatchString(repeaterId) {
		log.Println("ESCAPE_REPEATER_ID must be a UUID in lowercase")
		log.Println("To get your repeater id, go to https://app.escape.tech/repeaters/")
		log.Println("For more information, read the docs at https://docs.escape.tech/enterprise/repeater")
		os.Exit(1)
	}

	start(repeaterId)
}

func start(repeaterId string) {
	fmt.Println("Starting repeater client...")

	con, err := grpc.Dial("localhost:8081", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Error connecting: %v \n", err)
	}

	defer con.Close()

	// Set the client UUID in the metadata
	md := metadata.Pairs("client_uuid", repeaterId)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	client := proto.NewRepeaterClient(con)

	for {
		alreadyConnected := connectAndRun(client, ctx)
		log.Println("Disconnected...")
		if alreadyConnected {
			continue
		}
		log.Println("Reconnecting in 5 seconds...")
		time.Sleep(5 * time.Second)
	}
}

func connectAndRun(client proto.RepeaterClient, ctx context.Context) (hasConnected bool) {
	hasConnected = false
	stream, err := client.Stream(ctx)
	if err != nil {
		log.Printf("Error creating stream: %v \n", err)
		return hasConnected
	}
	hasConnected = true
	log.Println("Connected to server...")

	for {
		req, err := stream.Recv()
		if err != nil {
			log.Printf("Error receiving: %v \n", err)
			return hasConnected
		}
		log.Println("Got work")

		// Send request to server
		// Use a go func to avoid blocking the stream
		go internal.HandleRequest(req, &stream)
	}
}
