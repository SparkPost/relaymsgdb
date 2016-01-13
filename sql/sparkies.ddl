CREATE SCHEMA IF NOT EXISTS sparkies;
/*
Need to support multiple polls.
Need to support configurable auto-responses (one per email).
What email template to trigger, etc.
UI for seeing results.
 */

CREATE TABLE sparkies.polls (
  poll_id serial primary key,
  poll_name text not null
);
CREATE UNIQUE INDEX polls_lower_poll_name_uidx ON sparkies.polls(lower(poll_name));

CREATE TABLE sparkies.choices (
  choice_id serial primary key,
  poll_id   integer not null,
  choice    text not null
);
CREATE UNIQUE INDEX choices_poll_id_lower_choice_uidx ON sparkies.choices (poll_id, lower(choice));

CREATE TABLE sparkies.voters (
  voter_id serial primary key,
  opted_in boolean not null default false,
  email    text not null
);
CREATE UNIQUE INDEX voters_lower_email_uidx ON sparkies.voters (lower(email));

CREATE TABLE sparkies.votes (
  choice_id integer not null,
  voter_id  integer not null,
  vote_ts   timestamptz not null default clock_timestamp()
);
CREATE UNIQUE INDEX voters_voter_id_uidx ON sparkies.votes (voter_id);
