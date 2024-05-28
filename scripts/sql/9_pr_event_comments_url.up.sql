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

---- add to_use_in_ci_env_variable column
ALTER TABLE git_host_webhook_event_selectors ADD COLUMN IF NOT EXISTS to_use_in_ci_env_variable bool;

---- insert PR - comments_url for github into git_host_webhook_event_selectors
---- event_id : 1 - PR for github
INSERT INTO git_host_webhook_event_selectors (event_id, name, selector, to_show, to_show_in_ci_filter, to_use_in_ci_env_variable, is_active, possible_values, created_on)
VALUES (1, 'comments url', 'pull_request.comments_url', 'f', 'f', 't', 't', NULL, NOW());

---- add ci_env_variable_data column
ALTER TABLE webhook_event_parsed_data ADD COLUMN IF NOT EXISTS ci_env_variable_data JSON;
