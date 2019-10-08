package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nats-io/nats.go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
	"os"
	"strings"
	"time"
)

const (
	contentQueue   = "contentQueue"
	contentSubject = "contentSubject"

	trandoshanDatabase = "trandoshan"
	resourceCollection = "resources"
)

type resourceData struct {
	Url     string `json:"url"`
	Content string `json:"content"`
}

func main() {
	log.Print("Initializing persister")

	// initialize and validate database connection
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_URI")))
	if err != nil {
		log.Fatalf("Unable to create database connection: %s", err)
	}
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		log.Fatalf("Unable to connect to database: %s", err)
	}

	// connect to NATS server
	nc, err := nats.Connect(os.Getenv("NATS_URI"))
	if err != nil {
		log.Fatalf("Error while connecting to nats server: %s", err)
	}
	defer nc.Close()

	// initialize queue subscriber
	if _, err := nc.QueueSubscribe(contentSubject, contentQueue, handleMessages(client)); err != nil {
		log.Fatalf("Error while trying to subscribe to server: %s", err)
	}

	log.Print("Consumer initialized successfully")

	// todo: better way
	select {}
}

func handleMessages(client *mongo.Client) func(*nats.Msg) {
	resourceCollection := client.Database(trandoshanDatabase).Collection(resourceCollection)
	return func(msg *nats.Msg) {
		// setup production context
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

		var data resourceData

		// Unmarshal message
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Printf("Error while de-serializing payload: %sf", err)
			// todo: store in sort of DLQ?
			return
		}

		// Determinate if entry does not already exist
		resource, err := getResource(client, data.Url)
		if err != nil {
			log.Printf("Error while checking if url exist. Considering not. %s", err)
		}
		// there is a previous result
		if resource != nil {
			log.Printf("Url: %s is already crawled. Deleting old entry", data.Url)

			// delete old entry
			if _, err := resourceCollection.DeleteOne(ctx, bson.M{"url": data.Url}); err != nil {
				log.Printf("Error while deleting old entry. Cancelling persist request. %s", err)
				// todo: store in sort of DLQ?
				return
			}
		}

		// Finally create entry in database
		_, err = resourceCollection.InsertOne(ctx, bson.M{"url": data.Url, "crawlDate": time.Now(), "title": extractTitle(data.Content), "content": data.Content})
		if err != nil {
			log.Printf("Error while saving content: %s", err)
			// todo: store in sort of DLQ?
			return
		}
	}
}

// Extract title from given html
func extractTitle(body string) string {
	cleanBody := strings.ToLower(body)
	startPos := strings.Index(cleanBody, "<title>") + len("<title>")
	endPos := strings.Index(cleanBody, "</title>")

	// html tag absent of malformed
	if startPos == -1 || endPos == -1 {
		return ""
	}
	return body[startPos:endPos]
}

// Get resource using his url
// todo: call API instead?
func getResource(client *mongo.Client, url string) (*resourceData, error) {
	// Setup production context and acquire database collection
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	resourceCollection := client.Database(trandoshanDatabase).Collection(resourceCollection)

	// Query database for result
	var resource resourceData
	if err := resourceCollection.FindOne(ctx, bson.M{"url": url}).Decode(&resource); err != nil {
		return nil, fmt.Errorf("error while decoding result: %s", err)
	}

	return &resource, nil
}
