package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nsqio/go-nsq"
)

func main() {
	appConfig := NewConfig("config.yaml")

	nsqConfig := nsq.NewConfig()
	nsqConfig.MaxInFlight = appConfig.Nsq.MaxInFlight
	consumer, err := nsq.NewConsumer(appConfig.Nsq.Topic, appConfig.Nsq.Channel, nsqConfig)
	if err != nil {
		log.Fatal(err)
	}

	consumer.AddConcurrentHandlers(NewSearchSync(appConfig), appConfig.Nsq.Concurrency)

	err = consumer.ConnectToNSQLookupd(appConfig.Nsq.Address)
	if err != nil {
		log.Fatal(err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	consumer.Stop()
}
