ALTER TABLE git_provider ADD CONSTRAINT git_provider_name_key UNIQUE (name);

ALTER TABLE git_provider ADD CONSTRAINT git_provider_url_key UNIQUE (url);