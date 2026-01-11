package generator

// AppConfig holds the configuration for generating an application
type AppConfig struct {
	// Basic info
	AppName     string // e.g., "user-api"
	ModuleName  string // e.g., "github.com/myorg/user-api"
	Description string // e.g., "User management API"

	// Output
	OutputPath string // e.g., "./apps"

	// Framework reference
	UseLocalFramework bool   // true = use replace directive
	FrameworkPath     string // local path to framework

	// Server config
	ServerPort int
}

// NewDefaultConfig returns a config with sensible defaults
func NewDefaultConfig() *AppConfig {
	return &AppConfig{
		ServerPort:        8080,
		UseLocalFramework: true,
		FrameworkPath:     "../../go-yogan-framework",
	}
}

// Validate checks if the config is valid
func (c *AppConfig) Validate() error {
	if c.AppName == "" {
		return ErrAppNameRequired
	}
	if c.ModuleName == "" {
		return ErrModuleNameRequired
	}
	if c.OutputPath == "" {
		return ErrOutputPathRequired
	}
	return nil
}
