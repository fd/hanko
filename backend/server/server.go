package server

import (
	"net/http"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/teamhanko/hanko/backend/v2/config"
	"github.com/teamhanko/hanko/backend/v2/handler"
	"github.com/teamhanko/hanko/backend/v2/mapper"
	"github.com/teamhanko/hanko/backend/v2/persistence"
)

func StartPublic(cfg *config.Config, wg *sync.WaitGroup, persister persistence.Persister, prometheus echo.MiddlewareFunc, authenticatorMetadata mapper.AuthenticatorMetadata) {
	defer wg.Done()
	address := cfg.Server.Public.Address
	router := handler.NewPublicRouter(cfg, persister, prometheus, authenticatorMetadata)

	if fileDescriptorName, cut := strings.CutPrefix(address, "sd-socket/"); cut {
		sockets, err := getSystemdSockets()
		if err != nil {
			router.Logger.Fatal(err)
		}

		if len(sockets[fileDescriptorName]) != 1 {
			router.Logger.Fatalf("no systemd socket with name %s found")
		}

		router.Listener = sockets[fileDescriptorName][0]
		router.Logger.Fatal(router.StartServer(&http.Server{}))
	} else {
		router.Logger.Fatal(router.Start(address))
	}
}

func StartAdmin(cfg *config.Config, wg *sync.WaitGroup, persister persistence.Persister, prometheus echo.MiddlewareFunc) {
	defer wg.Done()
	address := cfg.Server.Admin.Address
	router := handler.NewAdminRouter(cfg, persister, prometheus)

	if fileDescriptorName, cut := strings.CutPrefix(address, "sd-socket/"); cut {
		sockets, err := getSystemdSockets()
		if err != nil {
			router.Logger.Fatal(err)
		}

		if len(sockets[fileDescriptorName]) != 1 {
			router.Logger.Fatalf("no systemd socket with name %s found")
		}

		router.Listener = sockets[fileDescriptorName][0]
		router.Logger.Fatal(router.StartServer(&http.Server{}))
	} else {
		router.Logger.Fatal(router.Start(address))
	}
}
