package download

import (
	"certwarden-backend/pkg/domain/certificates"
	"certwarden-backend/pkg/domain/orders"
	"certwarden-backend/pkg/domain/private_keys"
	"certwarden-backend/pkg/output"
	"errors"

	"go.uber.org/zap"
)

var errServiceComponent = errors.New("necessary download service component is missing")

// App interface is for connecting to the main app
type App interface {
	GetLogger() *zap.SugaredLogger
	GetOutputter() *output.Service
	GetDownloadStorage() Storage
}

// Storage interface for storage functions
type Storage interface {
	GetOneKeyByName(name string) (private_keys.Key, error)

	GetOneCertByName(name string) (cert certificates.Certificate, err error)

	GetCertNewestValidOrderByName(certName string) (order orders.Order, err error)
}

// Keys service struct
type Service struct {
	logger  *zap.SugaredLogger
	output  *output.Service
	storage Storage
}

// NewService creates a new private_key service
func NewService(app App) (*Service, error) {
	service := new(Service)

	// logger
	service.logger = app.GetLogger()
	if service.logger == nil {
		return nil, errServiceComponent
	}

	// output service
	service.output = app.GetOutputter()
	if service.output == nil {
		return nil, errServiceComponent
	}

	// storage
	service.storage = app.GetDownloadStorage()
	if service.storage == nil {
		return nil, errServiceComponent
	}

	return service, nil
}
