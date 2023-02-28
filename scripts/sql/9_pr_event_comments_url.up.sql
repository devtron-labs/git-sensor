---- add to_use_in_ci_env_variable column
ALTER TABLE git_host_webhook_event_selectors ADD COLUMN IF NOT EXISTS to_use_in_ci_env_variable bool;

---- insert PR - comments_url for github into git_host_webhook_event_selectors
---- event_id : 1 - PR for github
INSERT INTO git_host_webhook_event_selectors (event_id, name, selector, to_show, to_show_in_ci_filter, to_use_in_ci_env_variable, is_active, possible_values, created_on)
VALUES (1, 'comments url', 'pull_request.comments_url', 'f', 'f', 't', 't', NULL, NOW());

---- add ci_env_variable_data column
ALTER TABLE webhook_event_parsed_data ADD COLUMN IF NOT EXISTS ci_env_variable_data JSON;
