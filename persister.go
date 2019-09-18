package main

import (
   "context"
   "encoding/json"
   "github.com/joho/godotenv"
   "github.com/nats-io/nats.go"
   "go.mongodb.org/mongo-driver/bson"
   "go.mongodb.org/mongo-driver/mongo"
   "go.mongodb.org/mongo-driver/mongo/options"
   "go.mongodb.org/mongo-driver/mongo/readpref"
   "log"
   "os"
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

   //TODO: better way
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

      // Finally create entry in database
      //TODO: extract title from content
      //TODO: make sure url does not exist before and if so update entry instead
      _, err := pageCollection.InsertOne(ctx, bson.M{"url": data.Url, "crawlDate": time.Now(), "content": data.Content})
      if err != nil {
         log.Println("Error while saving content: ", err)
         // todo: store in sort of DLQ?
         return
      }
   }
}
