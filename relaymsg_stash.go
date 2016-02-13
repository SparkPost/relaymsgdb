package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	re "regexp"
	"strconv"
	"time"

	"github.com/SparkPost/go_relaymsg_stash/Godeps/_workspace/src/github.com/SparkPost/gopg"
	"github.com/SparkPost/go_relaymsg_stash/Godeps/_workspace/src/github.com/SparkPost/httpdump/storage"
	"github.com/SparkPost/go_relaymsg_stash/Godeps/_workspace/src/github.com/SparkPost/httpdump/storage/pg"

	"github.com/SparkPost/go_relaymsg_stash/Godeps/_workspace/src/github.com/husobee/vestigo"
)

var word *re.Regexp = re.MustCompile(`^\w*$`)
var nows *re.Regexp = re.MustCompile(`^\S*$`)
var digits *re.Regexp = re.MustCompile(`^\d*$`)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Set up validation for config from our environment.
	envVars := map[string]*re.Regexp{
		"PORT":                    digits,
		"DATABASE_URL":            nows,
		"RELAYMSG_PG_DB":          word,
		"RELAYMSG_PG_SCHEMA":      word,
		"RELAYMSG_PG_USER":        word,
		"RELAYMSG_PG_PASS":        nows,
		"RELAYMSG_BATCH_INTERVAL": digits,
		"RELAYMSG_INBOUND_DOMAIN": nows,
		"SPARKIES_ALLOWED_ORIGIN": nows,
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
	if cfg["PORT"] == "" {
		cfg["PORT"] = "5000"
	}
	if cfg["RELAYMSG_BATCH_INTERVAL"] == "" {
		cfg["RELAYMSG_BATCH_INTERVAL"] = "10"
	}
	batchInterval, err := strconv.Atoi(cfg["RELAYMSG_BATCH_INTERVAL"])
	if err != nil {
		log.Fatal(err)
	}
	if cfg["RELAYMSG_INBOUND_DOMAIN"] == "" {
		cfg["RELAYMSG_INBOUND_DOMAIN"] = "hey.avocado.industries"
	}

	pgcfg := &gopg.Config{
		Db:   cfg["RELAYMSG_PG_DB"],
		User: cfg["RELAYMSG_PG_USER"],
		Pass: cfg["RELAYMSG_PG_PASS"],
		Opts: map[string]string{
			"sslmode": "disable",
		},
		Url: cfg["DATABASE_URL"],
	}
	dbh, err := gopg.Connect(pgcfg)
	if err != nil {
		log.Fatal(err)
	}

	// Configure PostgreSQL dumper with connection details.
	schema := cfg["RELAYMSG_PG_SCHEMA"]
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
	msgParser := &RelayMsgParser{
		Dbh:    dbh,
		Schema: schema,
		Domain: cfg["RELAYMSG_INBOUND_DOMAIN"],
	}

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

	router := vestigo.NewRouter()

	router.SetGlobalCors(&vestigo.CorsAccessControl{
		AllowOrigin:   []string{cfg["SPARKIES_ALLOWED_ORIGIN"]},
		ExposeHeaders: []string{"accept"},
		AllowHeaders:  []string{"accept"},
	})

	// Install handler to store votes in database (incoming webhook events)
	router.Post("/incoming", reqDumper)
	router.Get("/summary/:localpart", msgParser.SummaryHandler())

	portSpec := fmt.Sprintf(":%s", cfg["PORT"])
	log.Fatal(http.ListenAndServe(portSpec, router))
}
