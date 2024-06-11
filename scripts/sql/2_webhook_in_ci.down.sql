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

/*
 * Copyright (c) 2024. Devtron Inc.
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

---- DROP index
DROP INDEX IF EXISTS public.git_material_url_IX;


