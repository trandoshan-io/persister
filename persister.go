package main

import (
   "context"
   "encoding/json"
   "fmt"
   "github.com/joho/godotenv"
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
)

type PageData struct {
   Url  string `json:"url"`
   Content string `json:"content"`
}

func main() {
   log.Println("Initializing persister")

   // load .env
   if err := godotenv.Load(); err != nil {
      log.Fatal("Unable to load .env file: ", err)
   }
   log.Println("Loaded .env file")

   // initialize and validate database connection
   ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
   client, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_URI")))
   if err != nil {
      log.Fatal("Unable to create database connection: ", err)
   }
   if err := client.Ping(ctx, readpref.Primary()); err != nil {
      log.Fatal("Unable to connect to database: ", err)
   }

   // connect to NATS server
   nc, err := nats.Connect(os.Getenv("NATS_URI"))
   if err != nil {
      log.Fatal("Error while connecting to nats server: ", err)
   }
   defer nc.Close()

   // initialize queue subscriber
   if _, err := nc.QueueSubscribe(contentSubject, contentQueue, handleMessages(client)); err != nil {
      log.Fatal("Error while trying to subscribe to server: ", err)
   }

   log.Println("Consumer initialized successfully")

   // todo: better way
   select {}
}

func handleMessages(client *mongo.Client) func(*nats.Msg) {
   pageCollection := client.Database("trandoshan").Collection("pages")
   return func(msg *nats.Msg) {
      // setup production context
      ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

      var data PageData

      // Unmarshal message
      if err := json.Unmarshal(msg.Data, &data); err != nil {
         log.Println("Error while de-serializing payload: ", err)
         // todo: store in sort of DLQ?
         return
      }

      // Determinate if entry does not already exist
      page, err := getPage(client, data.Url)
      if err != nil {
         log.Println("Error while checking if url exist. Considering not. (", err, ")")
      }
      // there is a previous result
      if page != nil {
         // todo: put update with entry lifespan policy ?
         log.Println("Url: " + data.Url + " is already crawled. Deleting old entry")

         // delete old entry
         if _, err := pageCollection.DeleteOne(ctx, bson.M{"url": data.Url}); err != nil {
            log.Println("Error while deleting old entry. Cancelling persist request. (", err, ")")
            // todo: store in sort of DLQ?
            return
         }
      }

      // Finally create entry in database
      _, err = pageCollection.InsertOne(ctx, bson.M{"url": data.Url, "crawlDate": time.Now(), "title": extractTitle(data.Content), "content": data.Content})
      if err != nil {
         log.Println("Error while saving content: ", err)
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

// Get page using his url
// todo: call API instead?
func getPage(client *mongo.Client, url string) (*PageData, error) {
   // Setup production context and acquire database collection
   ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
   pageCollection := client.Database("trandoshan").Collection("pages")

   // Query database for result
   var page PageData
   if err := pageCollection.FindOne(ctx, bson.M{"url": url}).Decode(&page); err != nil {
      return nil, fmt.Errorf("Error while decoding result: " + err.Error())
   }

   return &page, nil
}