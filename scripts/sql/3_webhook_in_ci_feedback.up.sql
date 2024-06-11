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

--- add column fix_value in git_host_webhook_event_selectors table
alter table git_host_webhook_event_selectors
add column fix_value character varying(1000);

--- set fix_value for github pull_request state
update git_host_webhook_event_selectors
set fix_value = '^open$'
where event_id = 1
and name = 'state';


--- set fix_value for bitbucket pull_request state
update git_host_webhook_event_selectors
set fix_value = '^OPEN$'
where event_id = 2
and name = 'state';


--- set fix_value for github tag creation identifier ref_type
update git_host_webhook_event_selectors
set fix_value = '^tag$'
where event_id = 3
and name = 'tag creation identifier';


--- set fix_value for bitbucket tag creation identifier ref_type
update git_host_webhook_event_selectors
set fix_value = '^tag$'
where event_id = 4
and name = 'tag creation identifier';


--- set selector created_at instead of updated_at for date for PR github
update git_host_webhook_event_selectors
set selector = 'pull_request.created_at'
where event_id = 1
and name = 'date';


--- set selector created_at instead of updated_at for date for PR bitbucket
update git_host_webhook_event_selectors
set selector = 'pullrequest.created_on'
where event_id = 2
and name = 'date';

---- drop table webhook_event_data as moved to orchestrator
DROP TABLE IF EXISTS public.webhook_event_data;

---- DROP sequence as moved to orchestrator
DROP SEQUENCE IF EXISTS public.webhook_event_data_id_seq;