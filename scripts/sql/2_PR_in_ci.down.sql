---- drop table ci_pipeline_material_pr_webhook_mapping
DROP TABLE IF EXISTS public.ci_pipeline_material_pr_webhook_mapping;

---- DROP sequence
DROP SEQUENCE IF EXISTS public.ci_pipeline_material_pr_webhook_mapping_id_seq;

---- drop table webhook_event_pr_data
DROP TABLE IF EXISTS public.webhook_event_pr_data;

---- DROP sequence
DROP SEQUENCE IF EXISTS public.webhook_event_pr_data_id_seq;

---- drop table webhook_event_json
DROP TABLE IF EXISTS public.webhook_event_json;

---- DROP sequence
DROP SEQUENCE IF EXISTS public.webhook_event_json_id_seq;

---- DROP index
DROP INDEX public.git_material_url_IX;


