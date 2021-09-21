package cleanup

import (
	iusers "github.com/k0sproject/k0s/internal/pkg/users"
	"github.com/k0sproject/k0s/pkg/config"
	"github.com/k0sproject/k0s/pkg/install"
	"github.com/sirupsen/logrus"
)

type users struct {
	Config *Config
}

// Name returns the name of the step
func (u *users) Name() string {
	return "remove k0s users step:"
}

// NeedsToRun detects controller users
func (u *users) NeedsToRun() bool {
	cfg, err := config.GetNodeConfig(u.Config.cfgFile, u.Config.k0sVars)
	if err != nil {
		return false
	}

	users := install.GetControllerUsers(cfg)
	for _, user := range users {
		if exists, _ := iusers.CheckIfUserExists(user); exists {
			return true
		}
	}
	return false
}

// Run removes all controller users that are present on the host
func (u *users) Run() error {
	logger := logrus.New()
	cfg, err := config.GetNodeConfig(u.Config.cfgFile, u.Config.k0sVars)
	if err != nil {
		logger.Errorf("failed to get cluster setup: %v", err)
	}
	if err := install.DeleteControllerUsers(cfg); err != nil {
		// don't fail, just notify on delete error
		logger.Infof("failed to delete controller users: %v", err)
	}
	return nil
}
