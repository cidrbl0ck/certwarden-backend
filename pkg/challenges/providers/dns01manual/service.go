package dns01manual

import (
	"certwarden-backend/pkg/acme"
	"certwarden-backend/pkg/datatypes/environment"
	"errors"
	"os/exec"

	"go.uber.org/zap"
)

var (
	errServiceComponent = errors.New("necessary dns-01 manual script component is missing")
)

// App interface is for connecting to the main app
type App interface {
	GetLogger() *zap.SugaredLogger
}

// provider Service struct
type Service struct {
	logger            *zap.SugaredLogger
	shellPath         string
	environmentParams *environment.Params
	createScriptPath  string
	deleteScriptPath  string
}

// ChallengeType returns the ACME Challenge Type this provider uses, which is dns-01
func (service *Service) AcmeChallengeType() acme.ChallengeType {
	return acme.ChallengeTypeDns01
}

// Stop is used for any actions needed prior to deleting this provider. If no actions
// are needed, it is just a no-op.
func (service *Service) Stop() error { return nil }

// Configuration options
type Config struct {
	Environment  []string `yaml:"environment" json:"environment"`
	CreateScript string   `yaml:"create_script" json:"create_script"`
	DeleteScript string   `yaml:"delete_script" json:"delete_script"`
}

// NewService creates a new service
func NewService(app App, cfg *Config) (*Service, error) {
	// if no config, error
	if cfg == nil {
		return nil, errServiceComponent
	}
	service := new(Service)

	// logger
	service.logger = app.GetLogger()
	if service.logger == nil {
		return nil, errServiceComponent
	}

	// determine shell (os dependent)
	// powershell
	var err error
	service.shellPath, err = exec.LookPath("powershell.exe")
	if err != nil {
		service.logger.Debugf("unable to find powershell (%s)", err)
		// then try bash
		service.shellPath, err = exec.LookPath("bash")
		if err != nil {
			service.logger.Debugf("unable to find bash (%s)", err)
			// then try zshell
			service.shellPath, err = exec.LookPath("zsh")
			if err != nil {
				service.logger.Debugf("unable to find zshell (%s)", err)
				// then try sh
				service.shellPath, err = exec.LookPath("sh")
				if err != nil {
					service.logger.Debugf("unable to find sh (%s)", err)
					// failed
					return nil, errors.New("unable to find suitable shell")
				}
			}
		}
	}

	// environment vars
	var invalidParams []string
	service.environmentParams, invalidParams = environment.NewParams(cfg.Environment)
	if len(invalidParams) > 0 {
		service.logger.Errorf("dns-01 manual: some environment param(s) invalid and won't be used (%s)", invalidParams)
	}

	// set script locations
	service.createScriptPath = cfg.CreateScript
	service.deleteScriptPath = cfg.DeleteScript

	return service, nil
}

// Update Service updates the Service to use the new config
func (service *Service) UpdateService(app App, cfg *Config) error {
	// if no config, error
	if cfg == nil {
		return errServiceComponent
	}

	// don't need to do anything with "old" Service, just set a new one
	newServ, err := NewService(app, cfg)
	if err != nil {
		return err
	}

	// set content of old pointer so anything with the pointer calls the
	// updated service
	*service = *newServ

	return nil
}
