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

package git

import (
	"context"
	"time"
)

type GitContext struct {
	context.Context        // Embedding original Go context
	Username               string
	Password               string
	CloningMode            string
	CACert                 string
	TLSKey                 string
	TLSCertificate         string
	TLSVerificationEnabled bool
}

func (gitCtx GitContext) WithCredentials(Username string, Password string) GitContext {
	gitCtx.Username = Username
	gitCtx.Password = Password
	return gitCtx
}

func (gitCtx GitContext) WithTLSData(caData string, tlsKey string, tlsCertificate string, tlsVerificationEnabled bool) GitContext {
	gitCtx.CACert = caData
	gitCtx.TLSKey = tlsKey
	gitCtx.TLSCertificate = tlsCertificate
	gitCtx.TLSVerificationEnabled = tlsVerificationEnabled
	return gitCtx
}

func BuildGitContext(ctx context.Context) GitContext {
	return GitContext{
		Context: ctx,
	}
}

func (gitCtx GitContext) WithTimeout(timeoutSeconds int) (GitContext, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(gitCtx.Context, time.Duration(timeoutSeconds)*time.Second)
	gitCtx.Context = ctx
	return gitCtx, cancel
}

func (gitCtx GitContext) WithCloningMode(CloningMode string) GitContext {
	if CloningMode == "" {
		CloningMode = CloningModeFull
	}
	gitCtx.CloningMode = CloningMode
	return gitCtx
}

func RunWithTimeout[T any](ctx context.Context, f func() ([]*T, error)) ([]*T, error) {
	resultCh := make(chan []*T)
	errCh := make(chan error)

	go func() {
		result, err := f()
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()

	select {
	case result := <-resultCh:
		return result, nil
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
