package main

import (
	"log"
	"os"

	"strconv"

	"github.com/joho/godotenv"
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

func NewConfig(envFile string, replicaYaml string) *Config {
	// Load .env file if it exists
	if _, err := os.Stat(envFile); err == nil {
		if err := godotenv.Load(envFile); err != nil {
			log.Printf("Warning: failed to load env file: %v", err)
		}
	}

	bytes, err := os.ReadFile(replicaYaml)
	if err != nil {
		log.Fatal("Failed reading config YAML: ", err)
	}

	config := &Config{}
	err = yaml.Unmarshal(bytes, config)
	if err != nil {
		log.Fatal("Failed parsing config YAML: ", err)
	}

	if v, ok := os.LookupEnv("DATABASE_URL"); ok {
		config.Database.ConnectionString = v
	}
	if v, ok := os.LookupEnv("MEILI_HOST_URL"); ok {
		config.Meilisearch.Host = v
	}
	if v, ok := os.LookupEnv("MEILI_API_KEY"); ok {
		config.Meilisearch.ApiKey = v
	}
	if v, ok := os.LookupEnv("NSQ_ADDRESS"); ok {
		config.Nsq.Address = v
	}
	if v, ok := os.LookupEnv("NSQ_CHANNEL"); ok {
		config.Nsq.Channel = v
	}
	if v, ok := os.LookupEnv("NSQ_TOPIC"); ok {
		config.Nsq.Topic = v
	}
	if v, ok := os.LookupEnv("NSQ_MAX_IN_FLIGHT"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			config.Nsq.MaxInFlight = n
		}
	}
	if v, ok := os.LookupEnv("NSQ_CONCURRENCY"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			config.Nsq.Concurrency = n
		}
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
