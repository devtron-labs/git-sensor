create table git_host(id INTEGER,name VARCHAR(20));

COPY git_host
FROM '/tmp/output.csv'
DELIMITER ',' ;

DO $$
DECLARE
a integer := (select id from git_host where name='Gitlab');
BEGIN
IF a NOT IN (select distinct git_host_id from git_host_webhook_event) THEN
INSERT INTO git_host_webhook_event (git_host_id, name, event_types_csv, action_type, is_active, created_on)
VALUES (a, 'Merge Request', 'Merge Request Hook', 'merged','t', NOW());
INSERT INTO git_host_webhook_event (git_host_id, name, event_types_csv, action_type, is_active, created_on)
VALUES (a, 'Tag Creation', 'Tag Push Hook', 'non-merged','t', NOW());
END IF;
END $$;


DO $$
DECLARE
a integer := (select id from git_host_webhook_event where name='Merge Request');
b integer := (select id from git_host_webhook_event where event_types_csv='Tag Push Hook');
BEGIN
IF a NOT IN (select distinct event_id from git_host_webhook_event_selectors) THEN
INSERT INTO git_host_webhook_event_selectors (event_id, name, selector, to_show, to_show_in_ci_filter, is_active, possible_values, created_on,fix_value)
VALUES (a, 'unique id', 'object_attributes.id', 'f', 'f', 't', NULL, NOW(),NULL),
       (a, 'repository url', 'repository.homepage', 'f', 'f', 't', NULL, NOW(),NULL),
       (a, 'title', 'object_attributes.title', 't', 't', 't', NULL, NOW(), NULL),
       (a, 'git url', 'object_attributes.url', 't', 'f', 't', NULL, NOW(), NULL),
       (a, 'author', 'object_attributes.source.namespace', 't', 't', 't', NULL, NOW(), NULL),
       (a, 'date', 'object_attributes.updated_at', 't', 'f', 't', NULL, NOW(), NULL),
       (a, 'target checkout', 'object_attributes.last_commit.id', 't', 'f', 't', NULL, NOW(), NULL),
       (a, 'source checkout', 'object_attributes.last_commit.id', 't', 'f', 't', NULL, NOW(), NULL),
       (a, 'target branch name', 'object_attributes.target_branch', 't', 't', 't', NULL, NOW(), NULL),
       (a, 'source branch name', 'object_attributes.source_branch', 't', 't', 't', NULL, NOW(), NULL),
       (a, 'state', 'object_attributes.state', 'f', 't', 't', 'opened', NOW(), '^opened$');
END IF;
IF b NOT IN (select distinct event_id from git_host_webhook_event_selectors) THEN
INSERT INTO git_host_webhook_event_selectors (event_id, name, selector, to_show, to_show_in_ci_filter, is_active, possible_values, created_on)
VALUES (b, 'repository url', 'project.web_url', 'f', 'f', 't', NULL, NOW()),
       (b, 'author', 'user_username', 't', 't', 't', NULL, NOW()),
       (b, 'date', 'commits.timestamp', 't', 'f', 't', NULL, NOW()),
       (b, 'target checkout', 'ref', 't', 'f', 't', NULL, NOW()),
       (b, 'tag name', 'ref', 'f', 't', 't', NULL, NOW()),
       (b, 'tag creation identifier', 'tag_push', 'f', 't', 't', NULL, NOW());
END IF;
END $$;
