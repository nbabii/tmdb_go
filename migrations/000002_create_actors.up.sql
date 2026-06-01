CREATE TABLE IF NOT EXISTS actors (
    id                  SERIAL      NOT NULL PRIMARY KEY,
    tmdb_person_id      INTEGER     NOT NULL,
    movie_tmdb_id       INTEGER     NOT NULL,
    name                TEXT        NOT NULL,
    character           TEXT        NOT NULL,
    UNIQUE (tmdb_person_id, movie_tmdb_id)
);
