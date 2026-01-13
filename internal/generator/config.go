package generator

// AppConfig holds the configuration for generating an application
type AppConfig struct {
	// Project info (multi-app workspace)
	ProjectName string // e.g., "my-project" (workspace root directory)
	OrgName     string // e.g., "github.com/myorg" (organization prefix)

	// App info
	AppName     string // e.g., "user-api"
	ModuleName  string // e.g., "github.com/myorg/my-project/apps/user-api"
	Description string // e.g., "User management API"

	// Output
	OutputPath string // e.g., "." (where to create project)

	// Framework reference
	UseLocalFramework bool   // true = use replace directive
	FrameworkPath     string // local path to framework (relative to apps/<app>)

	// Server config
	ServerPort int

	// Optional features
	GenerateProto bool // generate proto example (DemoPaymentService)
}

// NewDefaultConfig returns a config with sensible defaults
func NewDefaultConfig() *AppConfig {
	return &AppConfig{
		ServerPort:        8080,
		UseLocalFramework: true,
		FrameworkPath:     "../../../go-yogan-framework", // relative to apps/<app>/
	}
}

// Validate checks if the config is valid
func (c *AppConfig) Validate() error {
	if c.ProjectName == "" {
		return ErrProjectNameRequired
	}
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
