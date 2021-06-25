--
-- Name: webhook_event_json_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.webhook_event_json_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: webhook_event_json; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.webhook_event_json (
     id INTEGER NOT NULL DEFAULT nextval('webhook_event_json_id_seq'::regclass),
     git_host_name character varying(250) NOT NULL,
     payload_json JSON NOT NULL,
     created_on timestamptz NOT NULL,
     PRIMARY KEY ("id")
);


--
-- Name: webhook_event_data_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.webhook_event_pr_data_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: webhook_event_pr_data; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.webhook_event_pr_data (
    id INTEGER NOT NULL DEFAULT nextval('webhook_event_pr_data_id_seq'::regclass),
    git_host_name character varying(250) NOT NULL,
    pr_id character varying(250) NOT NULL,
    pr_title character varying(500) NOT NULL,
    pr_url character varying(500) NOT NULL,
    source_branch_name character varying(250) NOT NULL,
    source_branch_hash character varying(250) NOT NULL,
    target_branch_name character varying(250) NOT NULL,
    target_branch_hash character varying(250) NOT NULL,
    repository_url character varying(250) NOT NULL,
    author_name character varying(250) NOT NULL,
    is_open bool NOT NULL,
    actual_state character varying(250) NOT NULL,
    last_commit_message character varying(1000),
    pr_created_on timestamptz NOT NULL,
    pr_updated_on timestamptz,
    created_on timestamptz NOT NULL,
    updated_on timestamptz,
    PRIMARY KEY ("id")
);


--- Create index on git_host_name and pr_id
CREATE INDEX webhook_event_pr_data_ghname_prid ON public.webhook_event_pr_data (git_host_name, pr_id);



--
-- Name: ci_pipeline_material_pr_webhook_mapping_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.ci_pipeline_material_pr_webhook_mapping_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;



--
-- Name: ci_pipeline_material_pr_webhook_mapping; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.ci_pipeline_material_pr_webhook_mapping (
    id INTEGER NOT NULL DEFAULT nextval('ci_pipeline_material_pr_webhook_mapping_id_seq'::regclass),
    ci_pipeline_material_id INTEGER NOT NULL,
    pr_webhook_data_id INTEGER NOT NULL,
    PRIMARY KEY ("id")
);


---- Add Foreign key constraint on ci_pipeline_material_id in Table ci_pipeline_material_pr_webhook_mapping
ALTER TABLE ci_pipeline_material_pr_webhook_mapping
    ADD CONSTRAINT ci_pipeline_material_id_fkey FOREIGN KEY (ci_pipeline_material_id) REFERENCES public.ci_pipeline_material(id);



---- Add Foreign key constraint on pr_webhook_data_id in Table ci_pipeline_material_pr_webhook_mapping
ALTER TABLE ci_pipeline_material_pr_webhook_mapping
    ADD CONSTRAINT pr_webhook_data_id_fkey FOREIGN KEY (pr_webhook_data_id) REFERENCES public.webhook_event_pr_data(id);



--- Create index on ci_pipeline_material_pr_webhook_mapping.ci_pipeline_material_id
CREATE INDEX ci_pipeline_material_pr_webhook_mapping_IX ON public.ci_pipeline_material_pr_webhook_mapping (ci_pipeline_material_id);


--- Create index on ci_pipeline_material_pr_webhook_mapping.ci_pipeline_material_id and ci_pipeline_material_pr_webhook_mapping.pr_webhook_data_id
CREATE INDEX ci_pipeline_material_pr_webhook_mapping_IX2 ON public.ci_pipeline_material_pr_webhook_mapping (ci_pipeline_material_id, pr_webhook_data_id);



--- Create index on git_material.url
CREATE INDEX git_material_url_IX ON public.git_material (url);