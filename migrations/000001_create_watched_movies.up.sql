CREATE TABLE IF NOT EXISTS watched_movies (
    id              UUID        NOT NULL PRIMARY KEY,
    tmdb_id         INTEGER     NOT NULL,
    title           TEXT        NOT NULL,
    release_date    DATE,
    my_rating       INTEGER,
    my_overview     TEXT,
    my_date_watched DATE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
