package generator

import "errors"

var (
	ErrAppNameRequired    = errors.New("app name is required")
	ErrModuleNameRequired = errors.New("module name is required")
	ErrOutputPathRequired = errors.New("output path is required")
	ErrPathExists         = errors.New("path already exists")
	ErrTemplateRender     = errors.New("failed to render template")
)
