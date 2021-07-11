---- drop table ci_pipeline_material_webhook_data_mapping
DROP TABLE IF EXISTS public.ci_pipeline_material_webhook_data_mapping;

---- DROP sequence
DROP SEQUENCE IF EXISTS public.ci_pipeline_material_webhook_data_mapping_id_seq;

---- drop table webhook_event_parsed_data
DROP TABLE IF EXISTS public.webhook_event_parsed_data;

---- DROP sequence
DROP SEQUENCE IF EXISTS public.webhook_event_parsed_data_id_seq;

---- drop table webhook_event_data
DROP TABLE IF EXISTS public.webhook_event_data;

---- DROP sequence
DROP SEQUENCE IF EXISTS public.webhook_event_data_id_seq;

---- drop table git_host_webhook_event_selectors
DROP TABLE IF EXISTS public.git_host_webhook_event_selectors;

---- DROP sequence
DROP SEQUENCE IF EXISTS public.git_host_webhook_event_selector_id_seq;

---- drop table git_host_webhook_event
DROP TABLE IF EXISTS public.git_host_webhook_event;

---- DROP sequence
DROP SEQUENCE IF EXISTS public.git_host_webhook_event_id_seq;

---- DROP index
DROP INDEX IF EXISTS public.git_material_url_IX;


