package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	re "regexp"
	"strings"

	"github.com/SparkPost/gopg"
	"github.com/SparkPost/gosparkpost/events"
	"github.com/SparkPost/httpdump/storage"

	"github.com/husobee/vestigo"
)

type RelayMsgParser struct {
	Schema string
	Domain string
	Dbh    *sql.DB
}

func SchemaInit(dbh *sql.DB, schema string) error {
	if schema == "" {
		schema = "request_dump"
	}
	if strings.Index(schema, " ") >= 0 {
		return fmt.Errorf("SchemaInit: schemas containing a space are not supported")
	}

	exists, err := gopg.SchemaExists(dbh, schema)
	if err != nil {
		return err
	}
	if exists == false {
		return fmt.Errorf("PostgreSQL schema [%s] does not exist - did you run httpdump/storage/pg.SchemaInit?", schema)
	}

	table := "relay_messages"
	exists, err = gopg.TableExistsInSchema(dbh, table, schema)
	if err != nil {
		return err
	}
	if exists == false {
		log.Printf("SchemaInit: creating table [%s.%s]\n", schema, table)
		ddls := []string{
			fmt.Sprintf(`
				CREATE TABLE %s.%s (
					message_id  bigserial primary key,
					webhook_id  text,
					smtp_from   text,
					smtp_to     text,
					subject     text,
					rfc822      bytea,
					is_base64   bool,
					created     timestamptz default clock_timestamp(),
					status_id   integer default 0
				)
			`, schema, table),
			fmt.Sprintf("CREATE INDEX relay_messages_smtp_to_idx ON %s.%s (smtp_to)",
				schema, table),
		}
		for _, ddl := range ddls {
			_, err := dbh.Exec(ddl)
			if err != nil {
				return fmt.Errorf("SchemaInit: %s", err)
			}
		}
	}

	return nil
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
				err := p.ParseEvent(event)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

var relayMsg *re.Regexp = re.MustCompile(`^\s*\{\s*"msys"\s*:\s*{\s*"relay_message"\s*:`)

func (p *RelayMsgParser) ParseEvent(j *json.RawMessage) error {
	if j == nil {
		return nil
	}

	idx := relayMsg.FindStringIndex(string(*j))
	if len(idx) == 0 || idx[0] < 0 {
		log.Printf("ParseEvent ignored event: %s\n", string(*j))
		return nil
	}

	var blob map[string]map[string]events.RelayMessage
	err := json.Unmarshal([]byte(*j), &blob)
	if err != nil {
		log.Printf("ParseEvent failed to parse JSON:\n%s\n", string(*j))
	} else {
		msys, ok := blob["msys"]
		if !ok {
			log.Printf("ParseEvent ignored event with no \"msys\" key: %s\n", string(*j))
			return nil
		}
		msg, ok := msys["relay_message"]
		if !ok {
			log.Printf("ParseEvent ignored event with no \"relay_message\" key: %s\n", string(*j))
			return nil
		}
		log.Printf("%s => %s (%s)\n", msg.From, msg.To, msg.WebhookID)

		err := p.StoreEvent(&msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *RelayMsgParser) StoreEvent(msg *events.RelayMessage) error {
	_, err := p.Dbh.Exec(fmt.Sprintf(`
		INSERT INTO %s.relay_messages (
			webhook_id, smtp_from, smtp_to,
			subject, rfc822, is_base64
		) VALUES ($1, $2, $3, $4, $5, $6)
	`, p.Schema),
		msg.WebhookID, msg.From, msg.To,
		msg.Content.Subject, msg.Content.Email, msg.Content.Base64)
	if err != nil {
		return fmt.Errorf("StoreEvent (INSERT): %s", err)
	}
	return nil
}

func (p *RelayMsgParser) SummaryHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		localpart := vestigo.Param(r, "localpart")
		rows, err := p.Dbh.Query(fmt.Sprintf(`
			SELECT subject, count(*)
				FROM %s.relay_messages
			 WHERE smtp_to = $1 ||'@'|| $2
			 GROUP BY 1
		`, p.Schema), localpart, p.Domain)
		if err != nil {
			log.Printf("SummarizeEvents (SELECT): %s", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var res map[string]int
		for rows.Next() {
			if rows.Err() == io.EOF {
				break
			}
			var subject string
			var count int
			if err = rows.Scan(&subject, &count); err != nil {
				log.Printf("SummarizeEvents (Scan): %s", err)
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			res[subject] = count
		}
		if err = rows.Err(); err != nil {
			log.Printf("SummarizeEvents (Err): %s", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
	}
}
