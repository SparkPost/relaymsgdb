package main

import (
	"database/sql"
	"encoding/json"
	"log"
	re "regexp"

	"github.com/SparkPost/gosparkpost/events"
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
	for i, req := range reqs {
		var events []*json.RawMessage
		err := json.Unmarshal([]byte(req.Data), &events)
		if err != nil {
			log.Printf("ProcessRequests failed to parse JSON:\n%s\n", req.Data)
		} else {
			log.Printf("ProcessRequests found %d events from request %d\n", len(events), i)
			for _, event := range events {
				err := p.ProcessEvent(event)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

var relayMsg *re.Regexp = re.MustCompile(`^\s*\{\s*"msys"\s*:\s*{\s*"relay_message"\s*:`)

func (p *RelayMsgParser) ProcessEvent(j *json.RawMessage) error {
	if j == nil {
		return nil
	}

	idx := relayMsg.FindStringIndex(string(*j))
	if len(idx) == 0 || idx[0] < 0 {
		log.Printf("ProcessEvent ignored event: %s\n", string(*j))
		return nil
	}

	var blob map[string]map[string]events.RelayMessage
	err := json.Unmarshal([]byte(*j), &blob)
	if err != nil {
		log.Printf("ProcessEvent failed to parse JSON:\n%s\n", string(*j))
	} else {
		msys, ok := blob["msys"]
		if !ok {
			log.Printf("ProcessEvent ignored event with no \"msys\" key: %s\n", string(*j))
			return nil
		}
		msg, ok := msys["relay_message"]
		if !ok {
			log.Printf("ProcessEvent ignored event with no \"relay_message\" key: %s\n", string(*j))
			return nil
		}
		log.Printf("%s => %s (%s)\n", msg.From, msg.To, msg.WebhookID)
	}
	return nil
}
