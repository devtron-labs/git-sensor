DELETE FROM git_host_webhook_event_selectors WHERE event_id = 1 and name = 'description';

DELETE FROM git_host_webhook_event_selectors WHERE event_id = 2 and name = 'description';

ALTER TABLE ci_pipeline_material_webhook_data_mapping_filter_result DROP COLUMN IF EXISTS matched_groups;