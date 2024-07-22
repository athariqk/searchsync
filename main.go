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

	consumer, err := nsq.NewConsumer(appConfig.Nsq.Topic, appConfig.Nsq.Channel, nsq.NewConfig())
	if err != nil {
		log.Fatal(err)
	}

	consumer.AddHandler(NewSearchSync(appConfig))

	err = consumer.ConnectToNSQLookupd(appConfig.Nsq.Address)
	if err != nil {
		log.Fatal(err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	consumer.Stop()
}
