package api

import (
	"embed"
	"fmt"
	"html/template"
	"strings"

	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/license"
	"github.com/gemfast/server/internal/middleware"
	"github.com/gemfast/server/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

//go:embed templates/*
var efs embed.FS

const adminAPIPath = "/admin/api/v1"

func checkLicense(l *license.License) {
	if !l.Validated {
		config.Cfg.Auth = &config.AuthConfig{
			Type: "none",
		}
	}
}

func Run(l *license.License) error {
	checkLicense(l)
	router := initRouter()
	port := fmt.Sprintf(":%d", config.Cfg.Port)
	log.Info().Str("detail", port).Msg("gemfast server listening on port")
	if config.Cfg.Mirrors[0].Enabled {
		log.Info().Str("detail", config.Cfg.Mirrors[0].Upstream).Msg("mirroring upstream gem server")
	}
	return router.Run(port)
}

func initRouter() (r *gin.Engine) {
	gin.SetMode(gin.ReleaseMode)
	r = gin.Default()
	tmpl := template.Must(template.New("").ParseFS(efs, "templates/github/*.tmpl"))
	r.SetHTMLTemplate(tmpl)
	r.Use(gin.Recovery())
	r.GET("/up", health)
	authMode := config.Cfg.Auth.Type
	log.Info().Str("detail", authMode).Msg("configuring auth strategy")
	switch strings.ToLower(authMode) {
	case "github":
		configureGitHubAuth(r)
	case "local":
		configureLocalAuth(r)
	case "none":
		configureNoneAuth(r)
	}
	return r
}

func configureGitHubAuth(r *gin.Engine) {
	adminGitHubAuth := r.Group(adminAPIPath)
	adminGitHubAuth.POST("/login", middleware.GitHubLoginHandler)
	slash := r.Group("/")
	slash.GET("/github/callback", middleware.GitHubCallbackHandler)
	adminGitHubAuth.Use(middleware.GitHubMiddlewareFunc())
	{
		configureAdmin(adminGitHubAuth)
	}
	configurePrivate(r)
}

func configureLocalAuth(r *gin.Engine) {
	err := models.CreateAdminUserIfNotExists()
	if err != nil {
		panic(err)
	}
	err = models.CreateLocalUsers()
	if err != nil {
		panic(err)
	}
	jwtMiddleware, err := middleware.NewJwtMiddleware()
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize auth middleware")
	}
	adminLocalAuth := r.Group(adminAPIPath)
	adminLocalAuth.POST("/login", jwtMiddleware.LoginHandler)
	adminLocalAuth.GET("/refresh-token", jwtMiddleware.RefreshHandler)
	adminLocalAuth.Use(jwtMiddleware.MiddlewareFunc())
	{
		configureAdmin(adminLocalAuth)
	}
	configurePrivate(r)
}

func configureNoneAuth(r *gin.Engine) {
	if config.Cfg.Mirrors[0].Enabled {
		mirror := r.Group("/")
		configureMirror(mirror)
	}
	private := r.Group(config.Cfg.PrivateGemURL)
	configurePrivateRead(private)
	configurePrivateWrite(private)
	admin := r.Group(adminAPIPath)
	configureAdmin(admin)
}

// /
func configureMirror(mirror *gin.RouterGroup) {
	mirror.GET("/specs.4.8.gz", mirroredIndexHandler)
	mirror.GET("/latest_specs.4.8.gz", mirroredIndexHandler)
	mirror.GET("/prerelease_specs.4.8.gz", mirroredIndexHandler)
	mirror.GET("/quick/Marshal.4.8/:gemspec.rz", mirroredGemspecRzHandler)
	mirror.GET("/gems/:gem", mirroredGemHandler)
	mirror.GET("/api/v1/dependencies", mirroredDependenciesHandler)
	mirror.GET("/api/v1/dependencies.json", mirroredDependenciesJSONHandler)
	mirror.GET("/info/*gem", mirroredInfoHandler)
	mirror.GET("/versions", mirroredVersionsHandler)
}

// /private
func configurePrivate(r *gin.Engine) {
	privateTokenAuth := r.Group(config.Cfg.PrivateGemURL)
	privateTokenAuth.Use(middleware.NewTokenMiddleware())
	{
		if !config.Cfg.Auth.AllowAnonymousRead {
			configurePrivateRead(privateTokenAuth)
		}
		configurePrivateWrite(privateTokenAuth)
	}
	if config.Cfg.Mirrors[0].Enabled {
		mirror := r.Group("/")
		configureMirror(mirror)
	}
	if config.Cfg.Auth.AllowAnonymousRead {
		private := r.Group(config.Cfg.PrivateGemURL)
		configurePrivateRead(private)
	}
	middleware.InitACL()
}

// /private
func configurePrivateRead(private *gin.RouterGroup) {
	private.GET("/specs.4.8.gz", localIndexHandler)
	private.GET("/latest_specs.4.8.gz", localIndexHandler)
	private.GET("/prerelease_specs.4.8.gz", localIndexHandler)
	private.GET("/quick/Marshal.4.8/:gemspec.rz", localGemspecRzHandler)
	private.GET("/gems/:gem", localGemHandler)
	private.GET("/api/v1/dependencies", localDependenciesHandler)
	private.GET("/api/v1/dependencies.json", localDependenciesJSONHandler)
	private.GET("/versions", localVersionsHandler)
	private.GET("/info/:gem", localInfoHandler)
	private.GET("/names", localNamesHandler)
}

// /private
func configurePrivateWrite(private *gin.RouterGroup) {
	private.POST("/api/v1/gems", localUploadGemHandler)
	private.DELETE("/api/v1/gems/yank", localYankHandler)
	private.POST("/upload", geminaboxUploadGem)
}

// /admin
func configureAdmin(admin *gin.RouterGroup) {
	admin.GET("/auth", authMode)
	admin.POST("/token", middleware.CreateTokenHandler)
	admin.GET("/gems", listGems)
	admin.GET("/gems/:gem", getGem)
	admin.GET("/users", listUsers)
	admin.GET("/users/:username", getUser)
	admin.DELETE("/users/:username", deleteUser)
	admin.PUT("/users/:username/role/:role", setUserRole)
}
