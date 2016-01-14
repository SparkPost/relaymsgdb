# Summary

    $ git clone git@github.com:SparkPost/sparkies.git
    $ cd $GOPATH/src/github.com/SparkPost/sparkies
    $ (cd sql ; ./create.sh)
    $ go build
    $ SPARKIES_PG_DB=sparkies SPARKIES_HTTP_PORT=8080 ./sparkies
    $ echo '{"abc":123,"def":456}' > ./test.json
    $ curl -XPOST -H 'Content-Type: application/json' --data @test.json http://127.0.0.1:8080/cast_vote
    $ echo 'select * from request_dump.raw_requests' | psql -d sparkies

