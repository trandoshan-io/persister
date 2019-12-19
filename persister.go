package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/nats-io/nats.go"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	contentQueue   = "contentQueue"
	contentSubject = "contentSubject"
)

var (
	protocolRegex = regexp.MustCompile("https?://")
)

// Json data received from NATS
type resourceData struct {
	Url     string `json:"url"`
	Content string `json:"content"`
}

type resourceIndex struct {
	Url     string    `json:"url"`
	Content string    `json:"content"`
	Time    time.Time `json:"time"`
}

func main() {
	log.Print("Initializing persister")

	// connect to NATS server
	nc, err := nats.Connect(os.Getenv("NATS_URI"))
	if err != nil {
		log.Fatalf("Error while connecting to nats server: %s", err)
	}
	defer nc.Close()

	es, err := elasticsearch.NewDefaultClient()
	if err != nil {
		log.Fatalf("Error creating elasticsearch client: %s", err)
	}
	log.Printf("Elasticsearch client successfully created")

	// initialize queue subscriber
	if _, err := nc.QueueSubscribe(contentSubject, contentQueue, handleMessages(es)); err != nil {
		log.Fatalf("Error while trying to subscribe to server: %s", err)
	}
	log.Print("Consumer initialized successfully")

	// todo: better way
	select {}
}

func handleMessages(es *elasticsearch.Client) func(*nats.Msg) {
	return func(msg *nats.Msg) {
		var data resourceData

		// Unmarshal message
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Printf("Error while de-serializing payload: %sf", err)
			// todo: store in sort of DLQ?
			return
		}

		// Store content in the filesystem
		directory, fileName := computePath(data.Url, time.Now())
		storagePath := fmt.Sprintf("%s/%s", os.Getenv("STORAGE_PATH"), directory)
		filePath := fmt.Sprintf("%s/%s", storagePath, fileName)
		log.Printf("Storing content on path: %s", storagePath)

		if err := os.MkdirAll(storagePath, 0755); err != nil {
			log.Printf("Error while trying to create directory to save file: %s", err)
			return
		}
		if err := ioutil.WriteFile(filePath, []byte(data.Content), 0644); err != nil {
			log.Printf("Error while trying to save content: %s", err)
			return
		}

		// Create elasticsearch document
		doc := resourceIndex{
			Url: data.Url,
			Content: data.Content,
			Time: time.Now(),
		}

		// Serialize it into json
		docBytes, err := json.Marshal(&doc)
		if err != nil {
			log.Printf("Error while serializing document into json: %s", err)
			return
		}

		// Use Elasticsearch to index document
		req := esapi.IndexRequest{
			Index:   "resources",
			Body:    bytes.NewReader(docBytes),
			Refresh: "true",
		}
		res, err := req.Do(context.Background(), es)
		if err != nil {
			log.Printf("Error while creating elasticsearch index: %s", err)
		}
		defer res.Body.Close()
	}
}

// Compute path for resource storage using his URL and the crawling time
// Format is: resource-url/64bit-timestamp
// f.e: http://login.google.com/secure/createAccount.html -> login.google.com/secure/createAccount.html/1570788418
func computePath(resourceUrl string, crawlData time.Time) (string, string) {
	// first of all sanitize resource URL
	var sanitizedResourceUrl string
	// remove protocol
	sanitizedResourceUrl = protocolRegex.ReplaceAllLiteralString(resourceUrl, "")
	// remove any trailing '/'
	sanitizedResourceUrl = strings.TrimSuffix(sanitizedResourceUrl, "/")

	return sanitizedResourceUrl, strconv.FormatInt(crawlData.Unix(), 10)
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
