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

DELETE FROM git_host_webhook_event_selectors where event_id in (select id from git_host_webhook_event where git_host_name='Gitlab_Devtron' and name in 'Pull Request', 'Tag Creation')
DELETE FROM git_host_webhook_event where git_host_name='Gitlab_Devtron';
ALTER TABLE git_host_webhook_event  DROP COLUMN git_host_name;
ALTER TABLE git_host_webhook_event ALTER COLUMN git_host_id SET NOT NULL;