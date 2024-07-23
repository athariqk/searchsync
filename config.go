package main

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Replica struct {
	Namespace  string
	Index      string
	PrimaryKey string `yaml:"pk"`
}

type Config struct {
	Database struct {
		ConnectionString string
	}
	Meilisearch struct {
		Host   string
		ApiKey string
	}
	Nsq struct {
		Address     string
		Channel     string
		Topic       string
		MaxInFlight int
		Concurrency int
	}
	Replicas map[string]Replica
}

func NewConfig(filePath string) *Config {
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatal("Failed reading config.yaml: ", err)
	}

	config := &Config{}
	err = yaml.Unmarshal(bytes, config)
	if err != nil {
		log.Fatal("Failed parsing config.yaml: ", err)
	}

	config.init()

	return config
}

func (s *Config) init() {
	if s.Nsq.Topic == "" {
		s.Nsq.Topic = "replication"
	}
	if s.Nsq.Channel == "" {
		s.Nsq.Channel = "searchsyncer"
	}

	for name, rep := range s.Replicas {
		if rep.Namespace == "" {
			rep.Namespace = "public"
		}
		if rep.Index == "" {
			rep.Index = name
		}
		if rep.PrimaryKey == "" {
			rep.PrimaryKey = "id"
		}

		s.Replicas[name] = rep
	}
}
