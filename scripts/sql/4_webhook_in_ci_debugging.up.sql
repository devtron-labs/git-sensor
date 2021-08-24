--- add column is_active in ci_pipeline_material_webhook_data_mapping table
alter table ci_pipeline_material_webhook_data_mapping
add column is_active bool NOT NULL DEFAULT TRUE;

--- add column payload_data_id in webhook_event_parsed_data table
alter table webhook_event_parsed_data
add column payload_data_id INTEGER;