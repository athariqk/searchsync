package main

import (
	"database/sql"
	"encoding/json"
	"errors"
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
	err := json.Unmarshal(msg.Body, &replicationMsg)
	if err != nil {
		return err
	}

	switch replicationMsg.ReplicationFlag {
	case pgcdcmodels.FULL_REPLICATION:
		return s.handleFullReplication(replicationMsg.Command)
	case pgcdcmodels.STREAM_REPLICATION:
		return s.handleStreamReplication(replicationMsg.Command)
	}

	return nil
}

func (s *SearchSync) handleFullReplication(command *pgcdcmodels.DmlCommand) error {
	rep, ok := s.config.Replicas[command.Data.RelName]
	if !ok {
		return nil
	}

	if s.searchClient == nil {
		return errors.New("meilisearch client is null")
	}

	return s.handleUpsert(rep, &command.Data)
}

func (s *SearchSync) handleStreamReplication(command *pgcdcmodels.DmlCommand) error {
	rep, ok := s.config.Replicas[command.Data.RelName]
	if !ok {
		return nil
	}

	switch command.CmdType {
	case pgcdcmodels.INSERT:
		fallthrough
	case pgcdcmodels.UPDATE:
		err := s.handleUpsert(rep, &command.Data)
		if err != nil {
			return err
		}
	case pgcdcmodels.DELETE:
		err := s.handleDelete(rep, &command.Data)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *SearchSync) handleUpsert(rep Replica, row *pgcdcmodels.Row) error {
	document := pgcdcmodels.Flatten(row.Fields, false)
	resp, err := s.searchClient.Index(rep.Index).UpdateDocuments(document, rep.PrimaryKey)
	if err != nil {
		return err
	}

	task, err := s.searchClient.WaitForTask(resp.TaskUID)
	if err != nil {
		return err
	}

	log.Printf("finished task UID: %v of Type: %s status: %s",
		task.TaskUID,
		task.Type,
		task.Status)

	if rep.Privacy.AnonymizedTable != "" {
		return s.handlePrivacy(rep, row)
	}

	return nil
}

func (s *SearchSync) handleDelete(rep Replica, row *pgcdcmodels.Row) error {
	pk := s.getPrimaryKey(*row).Content.(float64)
	refNumber := fmt.Sprintf("%v", pk)

	resp, err := s.searchClient.Index(rep.Index).DeleteDocument(refNumber)
	if err != nil {
		return err
	}

	task, err := s.searchClient.WaitForTask(resp.TaskUID)
	if err != nil {
		return err
	}

	log.Printf("finished task UID: %v of Type: %s status: %s",
		task.TaskUID,
		task.Type,
		task.Status)

	return nil
}

func (s *SearchSync) handlePrivacy(rep Replica, row *pgcdcmodels.Row) error {
	// k := 0
	// kRow, err := a.db.Query(`SELECT anon.k_anonymity($1)`, cmd.Data.TableName)
	// if err != nil {
	// 	return err
	// }
	// kRow.Next()
	// kRow.Scan(k)

	// if k > table.kVarying {
	// 	continue
	// }

	pivotFQ := fmt.Sprintf("%s.%s.%s", row.Namespace, row.RelName, rep.Privacy.Pivot)
	pivotField := row.Fields[pivotFQ]
	query := fmt.Sprintf(`SELECT * FROM %s.%s WHERE %s = %v`,
		rep.Privacy.Namespace,
		row.RelName,
		rep.Privacy.Pivot,
		pivotField.Content)

	rows, err := s.db.Query(query)
	if err != nil {
		return err
	}
	values := getRowsFieldValues(rows)
	rows.Close()

	resp, err := s.searchClient.Index(rep.Index).UpdateDocuments(values, rep.PrimaryKey)
	if err != nil {
		return err
	}

	task, err := s.searchClient.WaitForTask(resp.TaskUID)
	if err != nil {
		return err
	}

	log.Printf("finished task UID: %v of Type: %s status: %s",
		task.TaskUID,
		task.Type,
		task.Status)

	return nil
}

func getRowsFieldValues(res *sql.Rows) []map[string]interface{} {
	m, err := res.ColumnTypes()
	if err != nil {
		return nil
	}

	result := []map[string]interface{}{}
	mLen := len(m)
	for res.Next() {
		mValStr := make([]sql.NullString, mLen)
		mVal := make([]interface{}, mLen)
		for i := range mVal {
			mVal[i] = &mValStr[i]
		}
		res.Scan(mVal...)

		row := map[string]interface{}{}
		for i, column := range m {
			row[column.Name()] = mVal[i].(*sql.NullString).String
		}

		result = append(result, row)
	}

	return result
}

func (s *SearchSync) getPrimaryKey(data pgcdcmodels.Row) pgcdcmodels.Field {
	return data.Fields[fmt.Sprintf("%s.%s.%s", data.Namespace, data.RelName, s.config.Replicas[data.RelName].PrimaryKey)]
}
