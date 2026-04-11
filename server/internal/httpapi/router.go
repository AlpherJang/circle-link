package httpapi

import (
	"net/http"

	"github.com/circle-link/circle-link/server/internal/service/auth"
	"github.com/circle-link/circle-link/server/internal/service/device"
)

type Server struct {
	authService   auth.Service
	deviceService device.Service
}

func NewRouter() http.Handler {
	return NewRouterWithServices(auth.NewMemoryService(), device.NewMemoryService())
}

func NewRouterWithServices(authService auth.Service, deviceService device.Service) http.Handler {
	server := &Server{
		authService:   authService,
		deviceService: deviceService,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("POST /v1/auth/signup", server.handleSignUp)
	mux.HandleFunc("POST /v1/auth/verify-email", server.handleVerifyEmail)
	mux.HandleFunc("POST /v1/auth/login", server.handleLogin)
	mux.HandleFunc("POST /v1/auth/refresh", server.handleRefresh)
	mux.HandleFunc("POST /v1/auth/logout", server.handleLogout)
	mux.HandleFunc("POST /v1/auth/change-password", server.handleChangePassword)
	mux.HandleFunc("GET /v1/devices", server.handleListDevices)
	mux.HandleFunc("POST /v1/devices", server.handleRegisterDevice)
	mux.HandleFunc("DELETE /v1/devices/{deviceId}", server.handleRevokeDevice)
	mux.HandleFunc("/v1/users/", notImplemented("prekey bundle"))
	mux.HandleFunc("/v1/contacts", notImplemented("contacts"))
	mux.HandleFunc("/v1/contacts/invite", notImplemented("contacts invite"))

	return mux
}
