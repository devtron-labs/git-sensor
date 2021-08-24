--- drop column is_active in ci_pipeline_material_webhook_data_mapping table
alter table ci_pipeline_material_webhook_data_mapping
drop column is_active;

--- drop column payload_data_id in webhook_event_parsed_data table
alter table webhook_event_parsed_data
drop column payload_data_id;