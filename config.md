# GITSENSOR CONFIGMAP

| Key                         | Value                           | Description                                                         |
|-----------------------------|---------------------------------|---------------------------------------------------------------------|
| PG_ADDR                     | postgresql-postgresql.devtroncd | PostgreSQL Server Address                                           |
| PG_USER                     | postgres                        | PostgreSQL User                                                     |
| PG_DATABASE                 | git_sensor                      | PostgreSQL Database Name                                            |
| SENTRY_ENV                  | prod                            | Sentry Environment                                                  |
| SENTRY_ENABLED              | "false"                         | Sentry Enabled (boolean)                                            |
| POLL_DURATION               | "1"                             | Polling Duration (in seconds)                                       |
| POLL_WORKER                 | "2"                             | Number of Polling Workers                                           |
| PG_LOG_QUERY                | "false"                         | PostgreSQL Query Logging (boolean)                                  |
| COMMIT_STATS_TIMEOUT_IN_SEC | "2"                             | Commit Stats Timeout (in seconds)                                   |
| ENABLE_FILE_STATS           | "false"                         | Enable File Stats (boolean)                                         |
| GIT_HISTORY_COUNT           | "15"                            | Git History Count                                                   |
| CLONING_MODE                | FULL                            | Cloning Mode (Possible values: SHALLOW, FULL)                       |
| USE_GIT_CLI                 | "false"                         | Use git cli commands directly for all git operations                |
| USE_GIT_CLI_ANALYTICS       | "false"                         | Use git cli commands directly for getting commit data for analytics |
