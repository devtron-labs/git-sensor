--
-- Name: git_host_event_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.git_host_webhook_event_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: git_host_webhook_event; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.git_host_webhook_event (
    id INTEGER NOT NULL DEFAULT nextval('git_host_webhook_event_id_seq'::regclass),
    git_host_id INTEGER NOT NULL,
    name character varying(250) NOT NULL,
    event_types_csv character varying(250) NOT NULL,
    action_type character varying(250) NOT NULL,
    is_active bool NOT NULL,
    created_on timestamptz NOT NULL,
    updated_on timestamptz,
    PRIMARY KEY ("id")
);


--
-- Name: git_host_webhook_event_selector_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.git_host_webhook_event_selector_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: git_host_webhook_event_selectors; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.git_host_webhook_event_selectors (
     id INTEGER NOT NULL DEFAULT nextval('git_host_webhook_event_selector_id_seq'::regclass),
     event_id INTEGER NOT NULL,
     name character varying(250) NOT NULL,
     selector character varying(250) NOT NULL,
     to_show bool NOT NULL,
     possible_values character varying(1000),
     is_active bool NOT NULL,
     created_on timestamptz NOT NULL,
     updated_on timestamptz,
     PRIMARY KEY ("id")
);


---- Add Foreign key constraint on event_id in Table git_host_event_selectors
ALTER TABLE git_host_webhook_event_selectors
    ADD CONSTRAINT git_host_webhook_event_selectors_eventId_fkey FOREIGN KEY (event_id) REFERENCES public.git_host_webhook_event(id);




--
-- Name: webhook_event_data_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.webhook_event_data_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: webhook_event_data; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.webhook_event_data (
   id INTEGER NOT NULL DEFAULT nextval('webhook_event_data_id_seq'::regclass),
   event_id INTEGER NOT NULL,
   payload_json JSON NOT NULL,
   created_on timestamptz NOT NULL,
   PRIMARY KEY ("id")
);



---- Add Foreign key constraint on event_id in Table webhook_event_data
ALTER TABLE webhook_event_data
    ADD CONSTRAINT webhook_event_data_eventId_fkey FOREIGN KEY (event_id) REFERENCES public.git_host_webhook_event(id);




--
-- Name: webhook_event_parsed_data_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.webhook_event_parsed_data_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: webhook_event_parsed_data; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.webhook_event_parsed_data (
    id INTEGER NOT NULL DEFAULT nextval('webhook_event_parsed_data_id_seq'::regclass),
    event_id INTEGER NOT NULL,
    unique_id character varying(250),
    event_action_type character varying(250),
    data JSON NOT NULL,
    created_on timestamptz NOT NULL,
    updated_on timestamptz,
    PRIMARY KEY ("id")
);



---- Add Foreign key constraint on event_id in Table webhook_event_parsed_data
ALTER TABLE webhook_event_parsed_data
    ADD CONSTRAINT webhook_event_parsed_data_eventId_fkey FOREIGN KEY (event_id) REFERENCES public.git_host_webhook_event(id);



--
-- Name: ci_pipeline_material_webhook_data_mapping_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.ci_pipeline_material_webhook_data_mapping_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;




--
-- Name: ci_pipeline_material_webhook_data_mapping; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.ci_pipeline_material_webhook_data_mapping (
    id INTEGER NOT NULL DEFAULT nextval('ci_pipeline_material_webhook_data_mapping_id_seq'::regclass),
    ci_pipeline_material_id INTEGER NOT NULL,
    webhook_data_id INTEGER NOT NULL,
    condition_matched bool NOT NULL,
    PRIMARY KEY ("id")
);


---- Add Foreign key constraint on ci_pipeline_material_id in Table ci_pipeline_material_webhook_data_mapping
ALTER TABLE ci_pipeline_material_webhook_data_mapping
    ADD CONSTRAINT ci_pipeline_material_id_fkey FOREIGN KEY (ci_pipeline_material_id) REFERENCES public.ci_pipeline_material(id);



---- Add Foreign key constraint on webhook_data_id in Table ci_pipeline_material_webhook_data_mapping
ALTER TABLE ci_pipeline_material_webhook_data_mapping
    ADD CONSTRAINT webhook_data_id_fkey FOREIGN KEY (webhook_data_id) REFERENCES public.webhook_event_parsed_data(id);



---- insert PR data into git_host_webhook_event
---- git_host_id : 1 - Github, 2 - Bitbucket
INSERT INTO git_host_webhook_event (git_host_id, name, event_types_csv, action_type, is_active, created_on)
VALUES (1, 'Pull Request', 'pull_request', 'merged', 't', NOW()),
       (2, 'Pull Request', 'pullrequest:created,pullrequest:updated,pullrequest:changes_request_created,pullrequest:approved', 'merged', 't', NOW());



---- insert PR data for github into git_host_webhook_event_selectors
---- event_id : 1 - PR for github, 2 - PR for bitbucket
INSERT INTO git_host_webhook_event_selectors (event_id, name, selector, to_show, is_active, possible_values, created_on)
VALUES (1, 'unique id', 'pull_request.id', 'f', 't', NULL, NOW()),
       (1, 'repository url', 'repository.html_url', 'f', 't', NULL, NOW()),
       (1, 'header', 'pull_request.title', 't', 't', NULL, NOW()),
       (1, 'git url', 'pull_request.html_url', 't', 't', NULL, NOW()),
       (1, 'author', 'sender.login', 't', 't', NULL, NOW()),
       (1, 'date', 'pull_request.updated_at', 't', 't', NULL, NOW()),
       (1, 'target checkout', 'pull_request.base.sha', 't', 't', NULL, NOW()),
       (1, 'source checkout', 'pull_request.head.sha', 't', 't', NULL, NOW()),
       (1, 'target branch name', 'pull_request.base.ref', 't', 't', NULL, NOW()),
       (1, 'source branch name', 'pull_request.head.ref', 't', 't', NULL, NOW()),
       (1, 'state', 'pull_request.state', 'f', 't', 'open', NOW());



---- insert PR data for bitbucket into git_host_webhook_event_selectors
---- event_id : 1 - PR for github, 2 - PR for bitbucket
INSERT INTO git_host_webhook_event_selectors (event_id, name, selector, to_show, is_active, possible_values, created_on)
VALUES (2, 'unique id', 'pullrequest.id', 'f', 't', NULL, NOW()),
       (2, 'repository url', 'repository.links.html.href', 'f', 't', NULL, NOW()),
       (2, 'header', 'pullrequest.title', 't', 't', NULL, NOW()),
       (2, 'git url', 'pullrequest.links.html.href', 't', 't', NULL, NOW()),
       (2, 'author', 'actor.display_name', 't', 't', NULL, NOW()),
       (2, 'date', 'pullrequest.updated_on', 't', 't', NULL, NOW()),
       (2, 'target checkout', 'pullrequest.destination.commit.hash', 't', 't', NULL, NOW()),
       (2, 'source checkout', 'pullrequest.source.commit.hash', 't', 't', NULL, NOW()),
       (2, 'target branch name', 'pullrequest.destination.branch.name', 't', 't', NULL, NOW()),
       (2, 'source branch name', 'pullrequest.source.branch.name', 't', 't', NULL, NOW()),
       (2, 'state', 'pullrequest.state', 'f', 't', 'OPEN', NOW());



---- insert tag creation into git_host_webhook_event
---- git_host_id : 1 - Github, 2 - Bitbucket
INSERT INTO git_host_webhook_event (git_host_id, name, event_types_csv, action_type, is_active, created_on)
VALUES (1, 'Tag Creation', 'create', 'non-merged', 't', NOW()),
       (2, 'Tag Creation', 'repo:push', 'non-merged', 't', NOW());


---- insert tag creation data for github into git_host_webhook_event_selectors
---- event_id : 3 - Tag creation for github, 4 - Tag creation for bitbucket
INSERT INTO git_host_webhook_event_selectors (event_id, name, selector, to_show, is_active, possible_values, created_on)
VALUES (3, 'repository url', 'repository.html_url', 'f', 't', NULL, NOW()),
       (3, 'author', 'sender.login', 't', 't', NULL, NOW()),
       (3, 'date', 'repository.pushed_at', 't', 't', NULL, NOW()),
       (3, 'tag creation identifier', 'ref_type', 'f', 't', NULL, NOW()),
       (3, 'tag name', 'ref', 'f', 't', NULL, NOW()),
       (3, 'target checkout', 'ref', 't', 't', NULL, NOW());



---- insert tag creation data for bitbucket into git_host_webhook_event_selectors
---- event_id : 3 - Tag creation for github, 4 - Tag creation for bitbucket
INSERT INTO git_host_webhook_event_selectors (event_id, name, selector, to_show, is_active, possible_values, created_on)
VALUES (4, 'repository url', 'repository.links.html.href', 'f', 't', NULL, NOW()),
       (4, 'author', 'actor.display_name', 't', 't', NULL, NOW()),
       (4, 'date', 'push.changes.0.new.date', 't', 't', NULL, NOW()),
       (4, 'tag creation identifier', 'push.changes.0.new.type', 'f', 't', NULL, NOW()),
       (4, 'tag name', 'push.changes.0.new.name', 'f', 't', NULL, NOW()),
       (4, 'target checkout', 'push.changes.0.new.name', 't', 't', NULL, NOW());



--- Create index on git_host_id in git_host_webhook_event
CREATE INDEX git_host_webhook_event_ghid_IX ON public.git_host_webhook_event (git_host_id);

--- Create index on event_id in git_host_webhook_event_selectors
CREATE INDEX git_host_webhook_event_selectors_eventId_IX ON public.git_host_webhook_event_selectors (event_id);


--- Create index on event_id and unique_id
CREATE INDEX webhook_event_parsed_data_eventId_uid_IX ON public.webhook_event_parsed_data (event_id, unique_id);


--- Create index on ci_pipeline_material_webhook_data_mapping.ci_pipeline_material_id
CREATE INDEX ci_pipeline_material_webhook_data_mapping_IX ON public.ci_pipeline_material_webhook_data_mapping (ci_pipeline_material_id);


--- Create index on ci_pipeline_material_webhook_data_mapping.ci_pipeline_material_id and ci_pipeline_material_webhook_data_mapping.webhook_data_id
CREATE INDEX ci_pipeline_material_webhook_data_mapping_IX2 ON public.ci_pipeline_material_webhook_data_mapping (ci_pipeline_material_id, webhook_data_id);


--- Create index on git_material.url
CREATE INDEX git_material_url_IX ON public.git_material (url);