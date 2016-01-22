package main

import (
	"database/sql"
	"encoding/json"
	"log"

	_ "github.com/SparkPost/gosparkpost/events"
	"github.com/SparkPost/httpdump/storage"
)

type RelayMsgParser struct {
	Schema string
	Dbh    *sql.DB
}

// ProcessBatches splits webhook payloads into individual events and stores
// data about each message in the relay_messages table.
func (p *RelayMsgParser) ProcessRequests(reqs []storage.Request) error {
	log.Printf("ProcessRequests called with %d requests\n", len(reqs))
	for _, req := range reqs {
		var events []*json.RawMessage
		err := json.Unmarshal([]byte(req.Data), &events)
		if err != nil {
			log.Printf("ProcessRequets failed to parse JSON:\n%s\n", req.Data)
		} else {
			log.Printf("ProcessRequests got %d events from %s\n", len(events), req.Data)
		}
	}
	return nil
}
