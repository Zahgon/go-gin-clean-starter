package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Caknoooo/go-gin-clean-starter/config"
	"github.com/Caknoooo/go-gin-clean-starter/middlewares"
	"github.com/Caknoooo/go-gin-clean-starter/modules/auth"
	authController "github.com/Caknoooo/go-gin-clean-starter/modules/auth/controller"
	authRepository "github.com/Caknoooo/go-gin-clean-starter/modules/auth/repository"
	authService "github.com/Caknoooo/go-gin-clean-starter/modules/auth/service"
	"github.com/Caknoooo/go-gin-clean-starter/modules/user"
	userController "github.com/Caknoooo/go-gin-clean-starter/modules/user/controller"
	userRepository "github.com/Caknoooo/go-gin-clean-starter/modules/user/repository"
	userService "github.com/Caknoooo/go-gin-clean-starter/modules/user/service"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/constants"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/httpx"
	"github.com/labstack/echo/v4"
	"github.com/samber/do"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

const (
	corsAllowOrigin      = "*"
	corsAllowCredentials = "true"
	corsAllowHeaders     = "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With"
	corsAllowMethods     = "POST, HEAD, PATCH, OPTIONS, GET, PUT, DELETE"
	notFoundBody         = "404 page not found"
	sampleUserID         = "1a6c0acd-a8e1-465a-b4f5-c75026b64e31"
)

func newTestServer(t *testing.T) *echo.Echo {
	t.Helper()

	db := config.SetUpInMemoryDatabase()

	injector := do.New()
	do.ProvideNamed(injector, constants.DB, func(i *do.Injector) (*gorm.DB, error) {
		return db, nil
	})
	do.ProvideNamed(injector, constants.JWTService, func(i *do.Injector) (authService.JWTService, error) {
		return authService.NewJWTService(), nil
	})

	jwtService := do.MustInvokeNamed[authService.JWTService](injector, constants.JWTService)
	userRepo := userRepository.NewUserRepository(db)
	refreshTokenRepo := authRepository.NewRefreshTokenRepository(db)

	do.Provide(injector, func(i *do.Injector) (userController.UserController, error) {
		return userController.NewUserController(i, userService.NewUserService(userRepo, db)), nil
	})
	do.Provide(injector, func(i *do.Injector) (authController.AuthController, error) {
		return authController.NewAuthController(i, authService.NewAuthService(userRepo, refreshTokenRepo, jwtService, db)), nil
	})

	server := httpx.NewServer()
	server.Use(middlewares.CORSMiddleware())
	server.Use(httpx.SingleSegmentParams())

	user.RegisterRoutes(server, injector)
	auth.RegisterRoutes(server, injector)

	return server
}

func doRequest(t *testing.T, method, target string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, target, nil)
	for key, values := range header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	rec := httptest.NewRecorder()
	newTestServer(t).ServeHTTP(rec, req)

	return rec
}

func TestUserRoutes_MeRequiresAuthentication(t *testing.T) {
	rec := doRequest(t, http.MethodGet, "/api/user/me", nil)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Equal(t, `{"status":false,"message":"failed proses request","error":"token not found"}`, rec.Body.String())
}

func TestUserRoutes_MeRejectsMalformedBearerToken(t *testing.T) {
	rec := doRequest(t, http.MethodGet, "/api/user/me", http.Header{"Authorization": []string{"Bearer garbage.token.here"}})

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, `{"status":false,"message":"failed proses request","error":"token not valid"}`, rec.Body.String())
}

func TestUserRoutes_MeRejectsAuthorizationWithoutBearerPrefix(t *testing.T) {
	rec := doRequest(t, http.MethodGet, "/api/user/me", http.Header{"Authorization": []string{"abcdef"}})

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, `{"status":false,"message":"failed proses request","error":"token not valid"}`, rec.Body.String())
}

func TestUserRoutes_UpdateRequiresAuthentication(t *testing.T) {
	rec := doRequest(t, http.MethodPut, "/api/user/"+sampleUserID, nil)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, `{"status":false,"message":"failed proses request","error":"token not found"}`, rec.Body.String())
}

func TestUserRoutes_DeleteRequiresAuthentication(t *testing.T) {
	rec := doRequest(t, http.MethodDelete, "/api/user/"+sampleUserID, nil)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, `{"status":false,"message":"failed proses request","error":"token not found"}`, rec.Body.String())
}

func TestUserRoutes_UnsupportedMethodOnCollectionIsNotFound(t *testing.T) {
	rec := doRequest(t, http.MethodPost, "/api/user", nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "text/plain", rec.Header().Get("Content-Type"))
	assert.Equal(t, notFoundBody, rec.Body.String())
}

func TestUserRoutes_UnsupportedMethodOnMeIsNotFound(t *testing.T) {
	rec := doRequest(t, http.MethodPatch, "/api/user/me", nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, notFoundBody, rec.Body.String())
}

func TestUserRoutes_UnknownPathIsNotFound(t *testing.T) {
	rec := doRequest(t, http.MethodGet, "/nope", nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "text/plain", rec.Header().Get("Content-Type"))
	assert.Equal(t, notFoundBody, rec.Body.String())
}

func TestUserRoutes_TrailingSlashRedirectsWithoutCORSHeaders(t *testing.T) {
	rec := doRequest(t, http.MethodGet, "/api/user/", nil)

	assert.Equal(t, http.StatusMovedPermanently, rec.Code)
	assert.Equal(t, "/api/user", rec.Header().Get("Location"))
	assert.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestUserRoutes_CORSHeadersArePresentOnEveryResponse(t *testing.T) {
	cases := []struct {
		name   string
		method string
		target string
		code   int
	}{
		{"unauthorized", http.MethodGet, "/api/user/me", http.StatusUnauthorized},
		{"not found", http.MethodGet, "/nope", http.StatusNotFound},
		{"wrong method", http.MethodPost, "/api/user", http.StatusNotFound},
		{"preflight", http.MethodOptions, "/api/user", http.StatusNoContent},
		{"panic", http.MethodPost, "/api/auth/logout", http.StatusInternalServerError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doRequest(t, tc.method, tc.target, nil)

			assert.Equal(t, tc.code, rec.Code)
			assert.Equal(t, corsAllowOrigin, rec.Header().Get("Access-Control-Allow-Origin"))
			assert.Equal(t, corsAllowCredentials, rec.Header().Get("Access-Control-Allow-Credentials"))
			assert.Equal(t, corsAllowHeaders, rec.Header().Get("Access-Control-Allow-Headers"))
			assert.Equal(t, corsAllowMethods, rec.Header().Get("Access-Control-Allow-Methods"))
		})
	}
}

func TestUserRoutes_PreflightReturnsNoContent(t *testing.T) {
	rec := doRequest(t, http.MethodOptions, "/api/user", nil)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Body.String())
	assert.Equal(t, corsAllowOrigin, rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestUserRoutes_PathParameterNeverSpansSegments(t *testing.T) {
	rec := doRequest(t, http.MethodPut, "/api/user/a/b", nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, notFoundBody, rec.Body.String())
}
