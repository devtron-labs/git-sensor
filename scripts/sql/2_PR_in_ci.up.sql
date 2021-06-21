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
    [pr_title character varying(500)] NOT NULL,
    pr_url character varying(500) NOT NULL,
    source_branch_name character varying(250) NOT NULL,
    source_branch_hash character varying(250) NOT NULL,
    target_branch_name character varying(250) NOT NULL,
    target_branch_hash character varying(250) NOT NULL,
    repository_url character varying(250) NOT NULL,
    author_name character varying(250) NOT NULL,
    is_open bool NOT NULL,
    actual_state character varying(250) NOT NULL,
    pr_created_on timestamptz NOT NULL,
    pr_updated_on timestamptz,
    created_on timestamptz NOT NULL,
    updated_on timestamptz,
    PRIMARY KEY ("id")
);


--- Create index on git_host_name and pr_id
CREATE INDEX webhook_event_pr_data_ghname_prid ON public.webhook_event_pr_data (git_host_name, pr_id);