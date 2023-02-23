---- insert PR - message for github into git_host_webhook_event_selectors
---- event_id : 1 - PR for github
INSERT INTO git_host_webhook_event_selectors (event_id, name, selector, to_show, to_show_in_ci_filter, is_active, possible_values, created_on)
VALUES (1, 'description', 'pull_request.body', 'f', 't', 't', NULL, NOW());

---- 2 - PR for bitbucket
INSERT INTO git_host_webhook_event_selectors (event_id, name, selector, to_show, to_show_in_ci_filter, is_active, possible_values, created_on)
VALUES (2, 'description', 'pullrequest.description', 'f', 't', 't', NULL, NOW());

---- add matched_groups column if condition matched
ALTER TABLE ci_pipeline_material_webhook_data_mapping_filter_result ADD COLUMN IF NOT EXISTS matched_groups JSON;