package generator

import "errors"

var (
	ErrProjectNameRequired  = errors.New("project name is required")
	ErrAppNameRequired      = errors.New("app name is required")
	ErrModuleNameRequired   = errors.New("module name is required")
	ErrOutputPathRequired   = errors.New("output path is required")
	ErrServiceNameRequired  = errors.New("service name is required")
	ErrPathExists           = errors.New("path already exists")
	ErrTemplateRender       = errors.New("failed to render template")
)
