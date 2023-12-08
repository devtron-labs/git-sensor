package git

import (
	"github.com/devtron-labs/git-sensor/internal"
)

func GetGitManager(conf *internal.Configuration, cliManager CliGitManager, goGitManager GoGitManager) GitManager {
	if conf.UseCli {
		return cliManager
	}
	return goGitManager
}
