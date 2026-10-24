package services

import "github.com/freak/guitarists/internal/platform"

// SystemService exposes read-only application information to the frontend.
type SystemService struct {
	version string
}

// NewSystemService creates the system service.
func NewSystemService(version string) *SystemService {
	return &SystemService{version: version}
}

// ServiceName implements application.ServiceName.
func (s *SystemService) ServiceName() string { return "SystemService" }

// Version returns the application version.
func (s *SystemService) Version() string {
	return s.version
}

// TabsDirectory returns the directory used to store tab files.
func (s *SystemService) TabsDirectory() (string, error) {
	return platform.TabsDir()
}
