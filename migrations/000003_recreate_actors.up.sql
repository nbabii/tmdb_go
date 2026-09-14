DROP TABLE IF EXISTS actors;

CREATE TABLE actors (
    id          INTEGER          NOT NULL PRIMARY KEY,
    name        TEXT             NOT NULL,
    popularity  DOUBLE PRECISION,
    biography   TEXT,
    birthday    DATE
);
