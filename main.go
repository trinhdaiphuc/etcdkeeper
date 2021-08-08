package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/trinhdaiphuc/etcdkeeper/config"
	"github.com/trinhdaiphuc/etcdkeeper/pkg/routers"
)

//go:embed web
var embededFiles embed.FS

func init() {
	config.Load()
}

func getFileSystem(dirName string) http.FileSystem {
	log.Print("using embed mode")
	fsys, err := fs.Sub(embededFiles, "web"+dirName)
	if err != nil {
		panic(err)
	}

	return http.FS(fsys)
}

func main() {
	// Setup
	cfg := config.GetConfig()
	e := echo.New()

	e.Logger.SetLevel(log.INFO)
	e.Use(middleware.Logger())

	// Set up routers
	routers.SetRoutes(e)

	// Set up static files
	assetHandler := func(dirName string) http.Handler {
		return http.FileServer(getFileSystem(dirName))
	}
	e.GET("/", echo.WrapHandler(assetHandler("")))
	e.GET("/static/*", echo.WrapHandler(http.StripPrefix("/static/", assetHandler("/static"))))

	// Start server
	go func() {
		if err := e.Start(fmt.Sprintf("%v:%d", cfg.Host, cfg.Port)); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("shutting down the server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
	// Use a buffered channel to avoid missing signals as recommended for signal.Notify
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}
