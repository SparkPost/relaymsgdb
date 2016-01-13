CREATE OR REPLACE FUNCTION sparkies.id_for_email(
  in_email text
) RETURNS integer LANGUAGE sql STRICT AS $$
  SELECT voter_id
    FROM sparkies.voters
   WHERE lower($1) = lower(email);
$$;

CREATE OR REPLACE FUNCTION sparkies.id_for_choice(
  in_choice text
) RETURNS integer LANGUAGE sql STRICT AS $$
  SELECT choice_id
    FROM sparkies.choices
   WHERE lower($1) = lower(choice);
$$;

CREATE OR REPLACE FUNCTION sparkies.already_voted(
  in_email text
) RETURNS boolean LANGUAGE plpgsql STRICT AS $$
DECLARE
  v_voter_id integer := sparkies.id_for_email(in_email);
  v_vote_ts timestamptz;
BEGIN
  -- If email doesn't exist, can't have voted.
  IF v_voter_id IS NULL THEN
    RETURN false;
  END IF;
  IF NOT EXISTS (
    SELECT vote_ts FROM sparkies.votes WHERE voter_id = v_voter_id
  ) THEN
    RETURN false;
  END IF;
  RETURN true;
END;
$$;

CREATE OR REPLACE FUNCTION sparkies.add_choice(
  in_choice text
) RETURNS integer LANGUAGE plpgsql STRICT AS $$
DECLARE
  v_choice_id integer := sparkies.id_for_choice(in_choice);
BEGIN
  IF v_choice_id IS NOT NULL THEN
    RAISE NOTICE 'Avoided choice duplication for id %', v_choice_id;
  ELSE
    INSERT INTO sparkies.choices (choice) VALUES (in_choice)
    RETURNING choice_id INTO v_choice_id;
  END IF;
  RETURN v_choice_id;
END;
$$;

CREATE OR REPLACE FUNCTION sparkies.cast_vote(
  in_email text, in_choice text
) RETURNS void LANGUAGE plpgsql STRICT AS $$
DECLARE
  v_voter_id integer := sparkies.id_for_email(in_email);
  v_choice_id integer := sparkies.id_for_choice(in_choice);
BEGIN
  IF v_choice_id IS NULL THEN
    RAISE NOTICE 'Ignoring vote from % for unknown choice [%]', in_email, in_choice;
    RETURN;
  END IF;
  -- NB: first poll; an email will only exist if they've already voted
  IF v_voter_id IS NOT NULL THEN
    RAISE NOTICE 'Ignoring duplicate vote from voter %', v_voter_id;
    RETURN;
  END IF;

  INSERT INTO sparkies.voters (email) VALUES (in_email)
  RETURNING voter_id INTO v_voter_id;

  INSERT INTO sparkies.votes (choice_id, voter_id) VALUES (v_choice_id, v_voter_id);
END;
$$;

