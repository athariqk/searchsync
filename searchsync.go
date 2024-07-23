package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	pgcdcmodels "github.com/athariqk/pgcdc-models"
	_ "github.com/lib/pq"
	"github.com/meilisearch/meilisearch-go"
	"github.com/nsqio/go-nsq"
)

type SearchSync struct {
	config       *Config
	db           *sql.DB
	searchClient *meilisearch.Client
	// batch        []*pgcdcmodels.Row
}

func NewSearchSync(config *Config) *SearchSync {
	db, err := sql.Open("postgres", config.Database.ConnectionString)
	if err != nil {
		log.Fatalln("Error connecting to DB:", err)
	}

	searchClientConfig := meilisearch.ClientConfig{
		Host:   config.Meilisearch.Host,
		APIKey: config.Meilisearch.ApiKey,
	}
	client := meilisearch.NewClient(searchClientConfig)

	log.Println("new meilisearch client:", searchClientConfig)

	_, err = client.GetVersion()
	if err != nil {
		log.Println("[warning] meilisearch doesn't seem to be reachable right now")
	}

	return &SearchSync{
		config:       config,
		db:           db,
		searchClient: client,
	}
}

func (s *SearchSync) HandleMessage(msg *nsq.Message) error {
	if len(msg.Body) <= 0 {
		return nil
	}

	replicationMsg := pgcdcmodels.ReplicationMessage{}
	superError := json.Unmarshal(msg.Body, &replicationMsg)
	if superError != nil {
		return superError
	}

	var tasks []*meilisearch.TaskInfo
	switch replicationMsg.ReplicationFlag {
	case pgcdcmodels.FULL_REPLICATION_PROGRESS:
		// s.batch = append(s.batch, &replicationMsg.Command.Data)
		rep, ok := s.config.Replicas[replicationMsg.Command.Data.RelName]
		if !ok {
			break
		}
		resp, err := s.handleUpsert(rep, &replicationMsg.Command.Data)
		tasks = []*meilisearch.TaskInfo{resp}
		superError = err
	case pgcdcmodels.FULL_REPLICATION_FINISHED:
		// WHY WON"T BATCHING WORK CORRECTLY GODDAMNIT
		// rep, ok := s.config.Replicas[replicationMsg.Command.Data.RelName]
		// if !ok {
		// 	log.Printf("Ignoring replica `%s`", replicationMsg.Command.Data.RelName)
		// 	break
		// }

		// if rep.Privacy.AnonymizedTable != "" {
		// 	tasks, superError = s.handlePrivacyInBatch(rep, replicationMsg.Command.Data.RelName)
		// 	break
		// }

		// documents := []map[string]interface{}{}
		// for _, r := range s.batch {
		// 	documents = append(documents, pgcdcmodels.Flatten(r.Fields, false))
		// }

		// tasks, superError = s.searchClient.Index(rep.Index).UpdateDocumentsInBatches(documents, 50, rep.PrimaryKey)
		// log.Println("replication message ended for index:", rep.Index)
		// s.batch = nil
	case pgcdcmodels.STREAM_REPLICATION:
		resp, err := s.handleStreamReplication(msg, replicationMsg.Command)
		tasks = []*meilisearch.TaskInfo{resp}
		superError = err
	}

	if superError != nil {
		return superError
	}

	if len(tasks) > 0 {
		failedTasks := 0
		successfulTasks := 0
		for _, task := range tasks {
			if task == nil {
				continue
			}
			waited, err := s.searchClient.WaitForTask(task.TaskUID)
			if err != nil {
				return err
			}
			if waited.Status == meilisearch.TaskStatusFailed {
				log.Println("meilisearch api error:", waited.Error)
				failedTasks++
			} else {
				successfulTasks++
			}
		}

		log.Printf("finished %v task(s), success: %v failed: %v", len(tasks), successfulTasks, failedTasks)
	}

	return nil
}

// func (s *SearchSync) handleFullReplication() ([]meilisearch.TaskInfo, error) {
// 	log.Printf("got %v batched DML commands of total %v, now upserting all", s.batch.receivedRowsCount, s.actualTotalRowsCount)
// 	resps, err := s.handleUpsertInBatch()
// 	if err != nil {
// 		return nil, err
// 	}
// 	return resps, err
// }

func (s *SearchSync) handleStreamReplication(_ *nsq.Message, command *pgcdcmodels.DmlCommand) (*meilisearch.TaskInfo, error) {
	rep, ok := s.config.Replicas[command.Data.RelName]
	if !ok {
		return nil, nil
	}

	switch command.CmdType {
	case pgcdcmodels.INSERT:
		fallthrough
	case pgcdcmodels.UPDATE:
		return s.handleUpsert(rep, &command.Data)
	case pgcdcmodels.DELETE:
		return s.handleDelete(rep, &command.Data)
	}

	return nil, nil
}

func (s *SearchSync) handleUpsert(rep Replica, row *pgcdcmodels.Row) (*meilisearch.TaskInfo, error) {
	document := pgcdcmodels.Flatten(row.Fields, false)
	return s.searchClient.Index(rep.Index).UpdateDocuments(document, rep.PrimaryKey)
}

// func (s *SearchSync) handleUpsertInBatch() ([]meilisearch.TaskInfo, error) {
// 	rep, ok := s.config.Replicas[s.batch.relName]
// 	if !ok {
// 		return nil, fmt.Errorf("Replica schema for `%s` is not specified", s.batch.relName)
// 	}

// 	if rep.Privacy.AnonymizedTable != "" {
// 		return s.handlePrivacyInBatch(rep, s.batch.relName)
// 	}

// 	documents := []map[string]interface{}{}
// 	for _, r := range s.batch.rows {
// 		documents = append(documents, pgcdcmodels.Flatten(r.Fields, false))
// 	}

// 	return s.searchClient.Index(rep.Index).UpdateDocumentsInBatches(documents, 50, rep.PrimaryKey)
// }

func (s *SearchSync) handleDelete(rep Replica, row *pgcdcmodels.Row) (*meilisearch.TaskInfo, error) {
	pk := s.getPrimaryKey(*row).Content.(float64)
	refNumber := fmt.Sprintf("%v", pk)
	return s.searchClient.Index(rep.Index).DeleteDocument(refNumber)
}

// func getRowsFieldValues(res *sql.Rows) []map[string]interface{} {
// 	m, err := res.ColumnTypes()
// 	if err != nil {
// 		return nil
// 	}

// 	result := []map[string]interface{}{}
// 	mLen := len(m)
// 	for res.Next() {
// 		mValStr := make([]sql.NullString, mLen)
// 		mVal := make([]interface{}, mLen)
// 		for i := range mVal {
// 			mVal[i] = &mValStr[i]
// 		}
// 		res.Scan(mVal...)

// 		row := map[string]interface{}{}
// 		for i, column := range m {
// 			row[column.Name()] = mVal[i].(*sql.NullString).String
// 		}

// 		result = append(result, row)
// 	}

// 	return result
// }

func (s *SearchSync) getPrimaryKey(data pgcdcmodels.Row) pgcdcmodels.Field {
	return data.Fields[fmt.Sprintf("%s.%s.%s", data.Namespace, data.RelName, s.config.Replicas[data.RelName].PrimaryKey)]
}
