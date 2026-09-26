CREATE TABLE app_samples (
    appid BIGINT, 
    sampled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    player_count BIGINT NOT NULL,
    PRIMARY KEY (appid, sampled_at),
    FOREIGN KEY (appid) REFERENCES steam_apps(appid)
);
