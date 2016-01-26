# Local development

## Prerequisites

* Ensure you have PostgreSQL installed and running. You can do this through [homebrew](http://brew.sh/) or by installing [Postgres.app](http://postgresapp.com/).
* Ensure you have autoenv installed. You can do this through homebrew.
* Ensure you have the [Heroku toolbelt](https://toolbelt.heroku.com/) installed
* Ensure you have Go installed. You can do this through homebrew.
* Ensure your `$GOPATH` is set.

## Get the code

```bash
$ cd $GOPATH/src/github.com/SparkPost
$ git clone git@github.com:SparkPost/sparkies.git
$ cd sparkies
```

## Build the Go project

```bash
$ go build
```

## Run the project

```bash
$ heroku local web
```

This will start the app on port 5000.

## Send a simulated relay webhook

Create a JSON file called test.json to simulate a relay webhook with the following contents:

```json
[{
  "msys": {
    "relay_message": {
      "friendly_from": "\"SparkPost Developers\" <devlopers@sparkpost.com>",
      "msg_from": "developers@sparkpost.com",
      "rcpt_to": "hello@messagesystems.com",
      "webhook_id": "66177122594674207",
      "content": {
        "html": "<html><head><title>Yay An HTML Title!</title></head><body><h1>And the html body.</h1></body></html>",
        "text": "Yay A Text Title!\nAnd the text body.",
        "subject": "Super Sweet Relay Message",
        "to": [
          "hello@messagesystems.com"
        ],
        "headers": [
          { "Received": "from the internet." }
        ],
        "email_rfc822": "In case it wasn't obvious, this isn't a valid MIME message.",
        "email_rfc822_is_base64": false
      }
    }
  }
}]
```

Call the `incoming` endpoint with the simulated relay webhook data:

```bash
$ curl -XPOST -H 'Content-Type: application/json' --data @test.json http://127.0.0.1:5000/incoming
```

## Viewing data

You can launch psql and inspect the data. To see the raw incoming data:

```bash
$ psql
user=# select * from request_dump.raw_requests;
```

To see the processed data:

```bash
$ psql
user=# select * from request_dump.relay_messages;
```

# Deploying

To deploy the code to Heroku, ensure you are authenticated as the appteam user. Then run the following commands to deploy:

```bash
$ git push heroku && git push heroku master
```
