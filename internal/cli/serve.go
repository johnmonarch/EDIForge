package cli

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"

	"github.com/johnmonarch/ediforge/internal/api"
	"github.com/johnmonarch/ediforge/internal/config"
	"github.com/johnmonarch/ediforge/internal/web"
)

func runServe(ctx context.Context, args []string) error {
	loaded, err := loadConfig()
	if err != nil {
		return err
	}
	cfg := loaded.Server
	var unsafeNoToken bool
	_, _, err = parseFlagSet("serve", args, map[string]bool{"require-token": true, "unsafe-no-token": true, "no-web": true, "open": true}, func(fs *flag.FlagSet) {
		fs.StringVar(&cfg.Host, "host", cfg.Host, "host to bind")
		fs.IntVar(&cfg.Port, "port", cfg.Port, "port to bind")
		fs.StringVar(&cfg.Token, "token", "", "API token")
		fs.BoolVar(&cfg.RequireToken, "require-token", false, "require API token")
		fs.BoolVar(&unsafeNoToken, "unsafe-no-token", false, "allow non-localhost bind without token")
		fs.Int64Var(&cfg.MaxBodyMB, "max-body-mb", cfg.MaxBodyMB, "maximum request body size")
		fs.StringVar(&cfg.CORSOrigin, "cors-origin", "", "explicit CORS origin")
		fs.Bool("no-web", false, "accepted; embedded web UI remains available in this MVP")
		fs.Bool("open", false, "accepted; print URL instead of launching a browser")
	})
	if err != nil {
		return err
	}
	if !config.IsLocalHost(cfg.Host) && cfg.Token == "" && cfg.RequireTokenOutsideLocalhost && !unsafeNoToken {
		return ExitError{Code: 6, Err: fmt.Errorf("binding to %s requires --token or explicit --unsafe-no-token", cfg.Host)}
	}
	if cfg.Token != "" {
		cfg.RequireToken = true
	}
	server := api.NewServer(newServiceWithConfig(loaded), cfg, web.Handler())
	addr := net.JoinHostPort(cfg.Host, fmt.Sprint(cfg.Port))
	httpServer := &http.Server{Addr: addr, Handler: server.Handler()}
	errCh := make(chan error, 1)
	go func() {
		fmt.Printf("EDIForge listening at http://%s\n", addr)
		errCh <- httpServer.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		_ = httpServer.Shutdown(context.Background())
		return ctx.Err()
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return ExitError{Code: 5, Err: err}
	}
}
