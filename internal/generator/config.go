package generator

// AppConfig holds the configuration for generating an HTTP application
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

// RPCConfig holds the configuration for generating a gRPC application
type RPCConfig struct {
	// Project info (multi-app workspace)
	ProjectName string // e.g., "my-project" (workspace root directory)
	OrgName     string // e.g., "github.com/myorg" (organization prefix)

	// App info
	AppName     string // e.g., "payment-rpc"
	ModuleName  string // e.g., "github.com/myorg/my-project/apps/payment-rpc"
	Description string // e.g., "Payment gRPC service"

	// Service info
	ServiceName string // e.g., "Payment" (proto service name, PascalCase)

	// Output
	OutputPath string // e.g., "." (where to create project)

	// Framework reference
	UseLocalFramework bool   // true = use replace directive
	FrameworkPath     string // local path to framework (relative to apps/<app>)

	// Server config
	GRPCPort int // e.g., 9000
}

// NewDefaultRPCConfig returns a config with sensible defaults
func NewDefaultRPCConfig() *RPCConfig {
	return &RPCConfig{
		GRPCPort:          9000,
		UseLocalFramework: true,
		FrameworkPath:     "../../../go-yogan-framework", // relative to apps/<app>/
	}
}

// Validate checks if the config is valid
func (c *RPCConfig) Validate() error {
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
	if c.ServiceName == "" {
		return ErrServiceNameRequired
	}
	return nil
}
