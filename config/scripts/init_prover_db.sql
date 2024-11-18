CREATE DATABASE prover_db;
\connect prover_db;

CREATE SCHEMA state;

CREATE TABLE state.nodes (hash BYTEA PRIMARY KEY, data BYTEA NOT NULL);
CREATE TABLE state.program (hash BYTEA PRIMARY KEY, data BYTEA NOT NULL);

CREATE USER prover_user with password 'prover_pass';
ALTER DATABASE prover_db OWNER TO prover_user;
ALTER SCHEMA state OWNER TO prover_user;
ALTER SCHEMA public OWNER TO prover_user;
ALTER TABLE state.nodes OWNER TO prover_user;
ALTER TABLE state.program OWNER TO prover_user;
ALTER USER prover_user SET SEARCH_PATH=state;

CREATE DATABASE dac_db;
\connect dac_db;

CREATE USER dac_user with password 'dac_password';
ALTER DATABASE dac_db OWNER TO dac_user;
ALTER SCHEMA public OWNER TO dac_user;
