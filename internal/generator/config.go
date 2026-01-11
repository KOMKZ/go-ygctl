package generator

// AppConfig holds the configuration for generating an application
type AppConfig struct {
	// Basic info
	AppName     string // e.g., "user-api"
	ModuleName  string // e.g., "github.com/myorg/user-api"
	Description string // e.g., "User management API"

	// Output
	OutputPath string // e.g., "./apps" or "/path/to/apps"

	// Framework reference
	UseLocalFramework bool   // true = use replace directive, false = use remote
	FrameworkPath     string // local path to framework (if UseLocalFramework)

	// Features
	EnableDatabase bool
	EnableRedis    bool
	SkipDemo       bool // If true, generate empty structure without demo code

	// Server config
	ServerPort int
}

// NewDefaultConfig returns a config with sensible defaults
func NewDefaultConfig() *AppConfig {
	return &AppConfig{
		ServerPort:        8080,
		UseLocalFramework: true,
		FrameworkPath:     "../../go-yogan-framework",
		EnableDatabase:    false,
		EnableRedis:       false,
		SkipDemo:          false,
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
