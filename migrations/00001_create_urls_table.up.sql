CREATE TABLE IF NOT EXISTS t_urls (
    n_id SERIAL PRIMARY KEY,
    s_long_url VARCHAR(1000) NOT NULL,
    s_short_url VARCHAR(50) NOT NULL
);

