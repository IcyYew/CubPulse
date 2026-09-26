CREATE TABLE steam_apps (
    appid BIGINT PRIMARY KEY,
    app_name TEXT NOT NULL,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
