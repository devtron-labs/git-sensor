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

--- add column is_active in ci_pipeline_material_webhook_data_mapping table
alter table ci_pipeline_material_webhook_data_mapping
    add column is_active bool NOT NULL DEFAULT TRUE;

--- add column payload_data_id in webhook_event_parsed_data table
alter table webhook_event_parsed_data
    add column payload_data_id INTEGER;


--- add column created_on in ci_pipeline_material_webhook_data_mapping table
alter table ci_pipeline_material_webhook_data_mapping
    add column created_on timestamptz;


--- add column updated_on in ci_pipeline_material_webhook_data_mapping table
alter table ci_pipeline_material_webhook_data_mapping
    add column updated_on timestamptz;


--
-- Name: ci_pipeline_material_webhook_data_mapping_filter_result_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.ci_pipeline_material_webhook_data_mapping_filter_result_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE CACHE 1;



--
-- Name: ci_pipeline_material_webhook_data_mapping_filter_result; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.ci_pipeline_material_webhook_data_mapping_filter_result
(
    id                      INTEGER                NOT NULL DEFAULT nextval(
            'ci_pipeline_material_webhook_data_mapping_filter_result_id_seq'::regclass),
    webhook_data_mapping_id INTEGER                NOT NULL,
    selector_name           character varying(250) NOT NULL,
    selector_condition      character varying(1000),
    selector_value          character varying(1000),
    condition_matched       bool                   NOT NULL,
    is_active               bool                   NOT NULL,
    created_on              timestamptz            NOT NULL,
    PRIMARY KEY ("id")
);


---- Add Foreign key constraint on webhook_data_mapping_id in Table ci_pipeline_material_webhook_data_mapping_filter_result
ALTER TABLE ci_pipeline_material_webhook_data_mapping_filter_result
    ADD CONSTRAINT webhook_data_mapping_id_fkey FOREIGN KEY (webhook_data_mapping_id) REFERENCES public.ci_pipeline_material_webhook_data_mapping (id);


--- Create index on ci_pipeline_material_webhook_data_mapping_filter_result.webhook_data_mapping_id
CREATE
INDEX ci_pipeline_material_webhook_data_mapping_filter_result_IX1 ON public.ci_pipeline_material_webhook_data_mapping_filter_result (webhook_data_mapping_id);