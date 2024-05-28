package git

import (
	"context"
	"time"
)

type GitContext struct {
	context.Context // Embedding original Go context
	Username        string
	Password        string
	CloningMode     string
}

func (gitCtx GitContext) WithCredentials(Username string, Password string) GitContext {
	gitCtx.Username = Username
	gitCtx.Password = Password
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
