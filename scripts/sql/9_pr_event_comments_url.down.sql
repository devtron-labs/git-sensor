ALTER TABLE git_host_webhook_event_selectors DROP COLUMN IF EXISTS to_use_in_ci_env_variable;

ALTER TABLE webhook_event_parsed_data DROP COLUMN IF EXISTS ci_env_variable_data;

DELETE FROM git_host_webhook_event_selectors WHERE event_id = 1 and name = 'comments url';