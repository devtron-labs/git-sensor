/*
 * Copyright (c) 2020-2024. Devtron Inc.
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

package git

import "time"

// Hash of commit
type Hash struct {
	Long  string
	Short string
}

// Tree hash of commit
type Tree struct {
	Long  string
	Short string
}

// Author of commit
type Author struct {
	Name  string
	Email string
	Date  time.Time
}

// Tag of commit
type Tag struct {
	Name string
	Date time.Time
}

// Committer of commit
type Committer struct {
	Name  string
	Email string
	Date  time.Time
}

// Commit data
type Commit struct {
	Hash      *Hash
	Tree      *Tree
	Author    *Author
	Committer *Committer
	Tag       *Tag
	Subject   string
	Body      string
}
