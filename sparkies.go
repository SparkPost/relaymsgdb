package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	re "regexp"
	"strconv"
	"time"

	"github.com/SparkPost/gopg"
	"github.com/SparkPost/httpdump/storage"
	"github.com/SparkPost/httpdump/storage/pg"

	"github.com/husobee/vestigo"
)

var word *re.Regexp = re.MustCompile(`^\w*$`)
var nows *re.Regexp = re.MustCompile(`^\S*$`)
var digits *re.Regexp = re.MustCompile(`^\d*$`)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Set up validation for config from our environment.
	envVars := map[string]*re.Regexp{
		"SPARKIES_HTTP_PORT":      digits,
		"SPARKIES_PG_DB":          word,
		"SPARKIES_PG_SCHEMA":      word,
		"SPARKIES_PG_USER":        word,
		"SPARKIES_PG_PASS":        nows,
		"SPARKIES_BATCH_INTERVAL": digits,
	}
	// Config container
	cfg := map[string]string{}
	for k, v := range envVars {
		cfg[k] = os.Getenv(k)
		if !v.MatchString(cfg[k]) {
			log.Fatalf("Unsupported value for %s, double check your parameters.", k)
		}
	}

	// Set defaults
	if cfg["SPARKIES_HTTP_PORT"] == "" {
		cfg["SPARKIES_HTTP_PORT"] = "80"
	}
	if cfg["SPARKIES_BATCH_INTERVAL"] == "" {
		cfg["SPARKIES_BATCH_INTERVAL"] = "10"
	}
	batchInterval, err := strconv.Atoi(cfg["SPARKIES_BATCH_INTERVAL"])
	if err != nil {
		log.Fatal(err)
	}

	pgcfg := &gopg.Config{
		Db:   cfg["SPARKIES_PG_DB"],
		User: cfg["SPARKIES_PG_USER"],
		Pass: cfg["SPARKIES_PG_PASS"],
		Opts: map[string]string{
			"sslmode": "disable",
		},
	}
	dbh, err := gopg.Connect(pgcfg)
	if err != nil {
		log.Fatal(err)
	}

	// Configure PostgreSQL dumper with connection details.
	schema := cfg["SPARKIES_PG_SCHEMA"]
	if schema == "" {
		schema = "request_dump"
	}
	pgDumper := &pg.PgDumper{Schema: schema}

	// make sure schema and raw_requests table exist
	err = pg.SchemaInit(dbh, schema)
	if err != nil {
		log.Fatal(err)
	}
	// make sure relay_messages table exists
	err = SchemaInit(dbh, schema)
	if err != nil {
		log.Fatal(err)
	}

	pgDumper.Dbh = dbh

	// Set up our handler which writes to, and reads from PostgreSQL.
	reqDumper := storage.HandlerFactory(pgDumper)

	// Set up our handler which writes individual events to PostgreSQL.
	msgParser := &RelayMsgParser{Dbh: dbh, Schema: schema}

	// recurring job to transform blobs of webhook data into relay_messages
	interval := time.Duration(batchInterval) * time.Second
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				go func() {
					_, err := storage.ProcessBatch(pgDumper, msgParser)
					if err != nil {
						log.Printf("%s\n", err)
					}
				}()
			}
		}
	}()

	// TODO: handler to generate html with mailto links for each entry

	router := vestigo.NewRouter()

	// Install handler to store votes in database (incoming webhook events)
	router.Post("/incoming", reqDumper)

	portSpec := fmt.Sprintf(":%s", cfg["SPARKIES_HTTP_PORT"])
	log.Fatal(http.ListenAndServe(portSpec, router))
}
