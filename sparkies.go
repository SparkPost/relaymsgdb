package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	re "regexp"

	"github.com/SparkPost/httpdump/storage"
	"github.com/SparkPost/httpdump/storage/pg"
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

	// Configure PostgreSQL dumper with connection details.
	pgDumper := &pg.PgDumper{
		Db:     cfg["SPARKIES_PG_DB"],
		Schema: cfg["SPARKIES_PG_SCHEMA"],
		User:   cfg["SPARKIES_PG_USER"],
		Pass:   cfg["SPARKIES_PG_PASS"],
	}
	err := pg.DbConnect(pgDumper)
	if err != nil {
		log.Fatal(err)
	}

	// Set up our handler which writes to, and reads from PostgreSQL.
	reqDumper := storage.HandlerFactory(pgDumper)

	// TODO: recurring job to parse votes and increment counters
	// TODO: handler to generate html with mailto links for each entry

	// Install handler to store votes in database (incoming webhook events)
	http.HandleFunc("/cast_vote", reqDumper)

	portSpec := fmt.Sprintf(":%s", cfg["SPARKIES_HTTP_PORT"])
	log.Fatal(http.ListenAndServe(portSpec, nil))
}
