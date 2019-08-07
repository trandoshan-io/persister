package main

import (
   "context"
   "encoding/json"
   "github.com/joho/godotenv"
   "github.com/streadway/amqp"
   tamqp "github.com/trandoshan-io/amqp"
   "go.mongodb.org/mongo-driver/bson"
   "go.mongodb.org/mongo-driver/mongo"
   "go.mongodb.org/mongo-driver/mongo/options"
   "go.mongodb.org/mongo-driver/mongo/readpref"
   "log"
   "os"
   "strconv"
   "time"
)

const (
   contentQueue = "content"
)

type WebsiteData struct {
   Url  string `json:"url"`
   Data string `json:"data"`
}

func main() {
   log.Println("Initializing persister")

   // load .env
   if err := godotenv.Load(); err != nil {
      log.Fatal("Unable to load .env file: ", err.Error())
   }
   log.Println("Loaded .env file")

   // initialize and validate database connection
   ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
   client, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_URI")))
   if err != nil {
      log.Fatal("Unable to create database connection: ", err.Error())
   }
   if err := client.Ping(ctx, readpref.Primary()); err != nil {
      log.Fatal("Unable to connect to database: ", err.Error())
   }

   // todo initialize API in goroutine (separate go file)

   prefetch, err := strconv.Atoi(os.Getenv("AMQP_PREFETCH"))
   if err != nil {
      log.Fatal(err)
   }

   // initialize consumer & start him
   consumer, err := tamqp.NewConsumer(os.Getenv("AMQP_URI"), prefetch)
   if err != nil {
      log.Fatal("Unable to create consumer: ", err.Error())
   }
   if err := consumer.Consume(contentQueue, false, handleMessages(client)); err != nil {
      log.Fatal("Unable to consume message: ", err.Error())
   }
   log.Println("Consumer initialized successfully")

   //TODO: better way
   select {}

   _ = consumer.Shutdown()
}

func handleMessages(client *mongo.Client) func(deliveries <-chan amqp.Delivery, done chan error) {
   contentCollection := client.Database("trandoshan").Collection("content")
   return func(deliveries <-chan amqp.Delivery, done chan error) {
      for delivery := range deliveries {
         var data WebsiteData

         // Unmarshal message
         if err := json.Unmarshal(delivery.Body, &data); err != nil {
            log.Println("Error while de-serializing payload: ", err.Error())
            _ = delivery.Reject(false)
            continue
         }

         // Finally create entry in database
         _, err := contentCollection.InsertOne(context.TODO(), bson.M{"url": data.Url, "data": data.Data})
         if err != nil {
            log.Println("Error while saving content: ", err.Error())
            _ = delivery.Reject(false)
            continue
         }

         _ = delivery.Ack(false)
      }
   }
}
