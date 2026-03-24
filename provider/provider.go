package provider

import (
	"github.com/laravel-ls/laravel-ls/cache"
	"github.com/laravel-ls/laravel-ls/config"
	"github.com/laravel-ls/laravel-ls/project"

	log "github.com/sirupsen/logrus"
)

type InitContext struct {
	Logger    *log.Entry
	RootPath  string
	FileCache *cache.FileCache
	Project   *project.Project
	Config    config.Config
}

type Provider interface {
	Register(manager *Manager)
	Init(ctx InitContext)
}
