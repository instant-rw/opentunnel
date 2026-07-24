ALTER TABLE replay_attempts
    ADD COLUMN response_status integer,
    ADD COLUMN duration_ms bigint;
