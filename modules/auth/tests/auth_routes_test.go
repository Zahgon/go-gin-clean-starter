package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
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

const notFoundBody = "404 page not found"

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

func doRequest(t *testing.T, method, target, contentType, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Del("Content-Type")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	rec := httptest.NewRecorder()
	newTestServer(t).ServeHTTP(rec, req)

	return rec
}

func TestAuthRoutes_LoginRejectsMalformedJSON(t *testing.T) {
	rec := doRequest(t, http.MethodPost, "/api/auth/login", "application/json", `{"email":`)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Equal(t, `{"status":false,"message":"failed get data from body","error":"unexpected EOF"}`, rec.Body.String())
}

func TestAuthRoutes_LoginRejectsEmptyJSONBody(t *testing.T) {
	rec := doRequest(t, http.MethodPost, "/api/auth/login", "application/json", "")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, `{"status":false,"message":"failed get data from body","error":"EOF"}`, rec.Body.String())
}

func TestAuthRoutes_LoginWithoutContentTypeFallsBackToFormBinding(t *testing.T) {
	rec := doRequest(t, http.MethodPost, "/api/auth/login", "", "")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, `{"status":false,"message":"failed get data from body","error":"Key: 'UserLoginRequest.Email' Error:Field validation for 'Email' failed on the 'required' tag\nKey: 'UserLoginRequest.Password' Error:Field validation for 'Password' failed on the 'required' tag"}`, rec.Body.String())
}

func TestAuthRoutes_LoginAcceptsFormEncodedBody(t *testing.T) {
	rec := doRequest(t, http.MethodPost, "/api/auth/login", "application/x-www-form-urlencoded", "email=nobody@example.com&password=secret123")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, `{"status":false,"message":"failed login","error":"email not found"}`, rec.Body.String())
}

func TestAuthRoutes_RegisterRejectsWrongScalarType(t *testing.T) {
	rec := doRequest(t, http.MethodPost, "/api/auth/register", "application/json", `{"name":123}`)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, `{"status":false,"message":"failed get data from body","error":"json: cannot unmarshal number into Go struct field UserCreateRequest.name of type string"}`, rec.Body.String())
}

func TestAuthRoutes_RefreshTokenRejectsMalformedJSON(t *testing.T) {
	rec := doRequest(t, http.MethodPost, "/api/auth/refresh", "application/json", `{`)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, `{"status":false,"message":"failed get data from body","error":"unexpected EOF"}`, rec.Body.String())
}

func TestAuthRoutes_RefreshTokenFailureUsesUnauthorized(t *testing.T) {
	rec := doRequest(t, http.MethodPost, "/api/auth/refresh", "application/json", `{"refresh_token":"nope"}`)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, `{"status":false,"message":"failed refresh token","error":"refresh token not found"}`, rec.Body.String())
}

func TestAuthRoutes_LogoutWithoutGuardPanicsIntoInternalServerError(t *testing.T) {
	rec := doRequest(t, http.MethodPost, "/api/auth/logout", "application/json", `{}`)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Empty(t, rec.Header().Get("Content-Type"))
	assert.Empty(t, rec.Body.String())
}

func TestAuthRoutes_GetOnLoginIsNotFound(t *testing.T) {
	rec := doRequest(t, http.MethodGet, "/api/auth/login", "", "")

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "text/plain", rec.Header().Get("Content-Type"))
	assert.Equal(t, notFoundBody, rec.Body.String())
}

func TestAuthRoutes_EveryEndpointIsRegisteredForPost(t *testing.T) {
	cases := []struct {
		path string
		code int
		body string
	}{
		{"/api/auth/register", http.StatusBadRequest, `{"status":false,"message":"failed get data from body","error":"Key: 'UserCreateRequest.Name' Error:Field validation for 'Name' failed on the 'required' tag\nKey: 'UserCreateRequest.Email' Error:Field validation for 'Email' failed on the 'required' tag\nKey: 'UserCreateRequest.Password' Error:Field validation for 'Password' failed on the 'required' tag"}`},
		{"/api/auth/login", http.StatusBadRequest, `{"status":false,"message":"failed get data from body","error":"Key: 'UserLoginRequest.Email' Error:Field validation for 'Email' failed on the 'required' tag\nKey: 'UserLoginRequest.Password' Error:Field validation for 'Password' failed on the 'required' tag"}`},
		{"/api/auth/refresh", http.StatusBadRequest, `{"status":false,"message":"failed get data from body","error":"Key: 'RefreshTokenRequest.RefreshToken' Error:Field validation for 'RefreshToken' failed on the 'required' tag"}`},
		{"/api/auth/logout", http.StatusInternalServerError, ``},
		{"/api/auth/send-verification-email", http.StatusBadRequest, `{"status":false,"message":"failed get data from body","error":"Key: 'SendVerificationEmailRequest.Email' Error:Field validation for 'Email' failed on the 'required' tag"}`},
		{"/api/auth/verify-email", http.StatusBadRequest, `{"status":false,"message":"failed get data from body","error":"Key: 'VerifyEmailRequest.Token' Error:Field validation for 'Token' failed on the 'required' tag"}`},
		{"/api/auth/send-password-reset", http.StatusBadRequest, `{"status":false,"message":"failed get data from body","error":"Key: 'SendPasswordResetRequest.Email' Error:Field validation for 'Email' failed on the 'required' tag"}`},
		{"/api/auth/reset-password", http.StatusBadRequest, `{"status":false,"message":"failed get data from body","error":"Key: 'ResetPasswordRequest.Token' Error:Field validation for 'Token' failed on the 'required' tag\nKey: 'ResetPasswordRequest.NewPassword' Error:Field validation for 'NewPassword' failed on the 'required' tag"}`},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			rec := doRequest(t, http.MethodPost, tc.path, "application/json", `{}`)

			assert.Equal(t, tc.code, rec.Code)
			assert.Equal(t, tc.body, rec.Body.String())
		})
	}
}

func TestAuthRoutes_JSONBodiesCarryNoTrailingNewline(t *testing.T) {
	rec := doRequest(t, http.MethodPost, "/api/auth/refresh", "application/json", `{"refresh_token":"nope"}`)

	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.False(t, strings.HasSuffix(rec.Body.String(), "\n"))
	assert.True(t, strings.HasSuffix(rec.Body.String(), "}"))
}

func TestAuthRoutes_UnknownAuthPathIsNotFound(t *testing.T) {
	rec := doRequest(t, http.MethodPost, "/api/auth/unknown", "application/json", `{}`)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, notFoundBody, rec.Body.String())
}
