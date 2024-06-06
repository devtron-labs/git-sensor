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

---- ALTER TABLE git_provider - modify type
ALTER TABLE git_provider
ALTER COLUMN ssh_private_key TYPE varchar(250);

---- ALTER TABLE git_provider - rename column
ALTER TABLE git_provider
RENAME COLUMN ssh_private_key TO ssh_key;

---- ALTER TABLE git_material - drop column
ALTER TABLE git_material
DROP COLUMN IF EXISTS fetch_submodules