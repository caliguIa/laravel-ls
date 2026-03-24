package project

import (
	"errors"
	"fmt"
	"path"

	"github.com/laravel-ls/laravel-ls/runtime"
	"github.com/laravel-ls/laravel-ls/utils"
	"github.com/laravel-ls/laravel-ls/utils/repository"
)

var ErrNotAnLaravelProject = errors.New("not a laravel project")

// Options configure optional behaviour when creating a Project.
type Options struct {
	// DBHost, when non-empty, overrides the DB_HOST environment variable
	// passed to the PHP introspection process. Useful when the database is
	// exposed on a non-default loopback address (e.g. Docker bind mounts).
	DBHost string
}

// Project encapsulates the runtime details needed to
// execute PHP code and retrieve information from a Laravel project.
type Project struct {
	rootPath string
	process  runtime.Process
}

// New initializes a new project by
// analyzing the project in rootPath and finding a suitable php process.
func New(rootPath string, opts Options) (*Project, error) {
	if !utils.FileExists(path.Join(rootPath, "bootstrap", "app.php")) {
		return nil, ErrNotAnLaravelProject
	}

	process, err := runtime.FindPHPProcess(rootPath)
	if err != nil {
		return nil, err
	}

	// Inject env overrides into the PHP process when requested.
	if opts.DBHost != "" {
		process.Env = append(process.Env, fmt.Sprintf("DB_HOST=%s", opts.DBHost))
	}

	return &Project{
		rootPath: rootPath,
		process:  process,
	}, nil
}

// RootPath returns the root path for the project
func (project Project) RootPath() string {
	return project.rootPath
}

// Process returns the php process for the project
func (project Project) Process() runtime.Process {
	return project.process
}

// Configs retrieves the configuration repository in the project
func (project Project) Configs() (repository.ConfigRepository, error) {
	return runtime.CallScript(project.process, project.rootPath, configScript, repository.ConfigRepository{})
}

// Routes retrieves the routes repository in the project
func (project Project) Routes() (repository.RouteRepository, error) {
	return runtime.CallScript(project.process, project.rootPath, routeScript, repository.RouteRepository{})
}

// AppBindings retrieves the application bindings repository in the project
func (project Project) AppBindings() (repository.AppRepository, error) {
	return runtime.CallScript(project.process, project.rootPath, appScript, repository.AppRepository{})
}

// Models retrieves the Eloquent model repository in the project
func (project Project) Models() (repository.ModelRepository, error) {
	return runtime.CallScript(project.process, project.rootPath, modelScript, repository.ModelRepository{})
}
