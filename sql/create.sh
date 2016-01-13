#!/bin/sh

createdb -E utf8 sparkies
psql -d sparkies -c '\i ./sparkies.ddl'
psql -d sparkies -c '\i ./sparkies-funcs.sql'
