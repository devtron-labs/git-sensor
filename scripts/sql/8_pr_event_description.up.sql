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

---- insert PR - message for github into git_host_webhook_event_selectors
---- event_id : 1 - PR for github
INSERT INTO git_host_webhook_event_selectors (event_id, name, selector, to_show, to_show_in_ci_filter, is_active, possible_values, created_on)
VALUES (1, 'description', 'pull_request.body', 'f', 't', 't', NULL, NOW());

---- 2 - PR for bitbucket
INSERT INTO git_host_webhook_event_selectors (event_id, name, selector, to_show, to_show_in_ci_filter, is_active, possible_values, created_on)
VALUES (2, 'description', 'pullrequest.description', 'f', 't', 't', NULL, NOW());

---- add matched_groups column if condition matched
ALTER TABLE ci_pipeline_material_webhook_data_mapping_filter_result ADD COLUMN IF NOT EXISTS matched_groups JSON;