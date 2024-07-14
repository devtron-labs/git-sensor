/*
 * Copyright (c) 2020-2024. Devtron Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
create table if not exists git_provider
(
    id           int primary key,
    name         varchar(250) unique not null,
    url          varchar(250) unique not null,
    user_name    varchar(25),
    password     varchar(250),
    ssh_key      varchar(250),
    access_token varchar(250),
    auth_mode    varchar(250),
    active       boolean             NOT NULL
);

CREATE TABLE IF NOT EXISTS git_material
(
    id                     int primary key,
    git_provider_id        INT REFERENCES git_provider (id),
    deleted                boolean NOT NULL,
    name                   varchar(250),
    url                    varchar(250),
    checkout_location      varchar(250),
    checkout_msg_any       varchar(250),
    checkout_status        boolean NOT NULL,
    last_fetch_time        timestamptz,
    fetch_status           boolean,
    last_fetch_error_count int,
    fetch_error_message    text
);

CREATE TABLE IF NOT EXISTS ci_pipeline_material
(
    id              int primary key,
    git_material_id int references git_material (id),
    type            varchar(250),
    value           varchar(250),
    active          boolean,
    last_seen_hash  varchar(250),
    commit_author   varchar(250),
    commit_date     timestamptz
);

ALTER TABLE ci_pipeline_material ADD COLUMN commit_history text;
ALTER TABLE ci_pipeline_material ADD COLUMN errored boolean;
ALTER TABLE ci_pipeline_material ADD COLUMN error_msg text;

INSERT INTO "public"."git_provider" ("id", "name", "url", "user_name", "password", "ssh_key", "access_token", "auth_mode", "active") VALUES
('1', 'Github Public', 'github.com', NULL, NULL, NULL, NULL, 'ANONYMOUS', 't');