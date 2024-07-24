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
ALTER TABLE git_host_webhook_event ADD COLUMN IF NOT EXISTS git_host_name varchar(250);

INSERT INTO git_host_webhook_event (git_host_name, name, event_types_csv,
                                    action_type, is_active, created_on)
VALUES ('Gitlab_Devtron', 'Pull Request', 'Merge Request Hook', 'non-merged','t', NOW()),
       ('Gitlab_Devtron', 'Tag Creation', 'Tag Push Hook', 'merged','t', NOW());

INSERT INTO git_host_webhook_event_selectors
(event_id, name, selector, to_show, to_show_in_ci_filter, is_active, possible_values, created_on)
VALUES (5, 'unique id', 'object_attributes.id', 'f', 'f', 't', NULL, NOW()),
       (5, 'repository url', 'project.http_url', 'f', 'f', 't', NULL, NOW()),
       (5, 'title', 'object_attributes.title', 't', 't', 't', NULL, NOW()),
       (5, 'git url', 'object_attributes.url', 't', 'f', 't', NULL, NOW()),
       (5, 'author', 'user.username', 't', 't', 't', NULL, NOW()),
       (5, 'date', 'object_attributes.updated_at', 't', 'f', 't', NULL, NOW()),
       (5, 'source checkout', 'object_attributes.last_commit.id', 't', 'f', 't', NULL, NOW()),
       (5, 'target branch name', 'object_attributes.target_branch', 't', 't', 't', NULL, NOW()),
       (5, 'source branch name', 'object_attributes.source_branch', 't', 't', 't', NULL, NOW()),
       (5, 'repository ssh url', 'project.ssh_url', 't', 't', 't', NULL, NOW()),
       (5, 'description', 'object_attributes.description', 't', 't', 't', NULL, NOW()),
       (5, 'state', 'object_attributes.state', 't', 't', 't', 'opened', NOW());


INSERT INTO git_host_webhook_event_selectors
(event_id, name, selector, to_show, to_show_in_ci_filter, is_active, possible_values, created_on)
VALUES
    (6, 'repository url', 'project.web_url', 'f', 'f', 't', NULL, NOW()),
    (6, 'author', 'user_username', 't', 't', 't', NULL, NOW()),
    (6, 'date', 'object_attributes.updated_at', 't', 'f', 't', NULL, NOW()),
    (6, 'tag name', 'ref', 't', 'f', 't', NULL, NOW()),
    (6, 'target checkout', 'checkout_sha', 't', 'f', 't', NULL, NOW()),
    (6, 'tag creation identifier', 'checkout_sha', 't', 't', 't', NULL, NOW()),
    (6, 'repository ssh url', 'project.ssh_url', 't', 't', 't', NULL, NOW());


--- set fix_value for gitlab pull_request state
update git_host_webhook_event_selectors
set fix_value = '^opened$'
where event_id = 5
  and name = 'state';

--- set fix_value for gitlab tag creation identifier ref_type
update git_host_webhook_event_selectors
set fix_value = '\b[0-9a-f]{5,40}\b'
where event_id=6 and name='tag creation identifier';