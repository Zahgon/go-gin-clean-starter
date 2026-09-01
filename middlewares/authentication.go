package middlewares

import (
	"net/http"
	"strings"

	"github.com/Caknoooo/go-gin-clean-starter/modules/auth/service"
	"github.com/Caknoooo/go-gin-clean-starter/modules/user/dto"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/labstack/echo/v4"
)

func Authenticate(jwtService service.JWTService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			authHeader := ctx.Request().Header.Get("Authorization")

			if authHeader == "" {
				response := utils.BuildResponseFailed(dto.MESSAGE_FAILED_PROSES_REQUEST, dto.MESSAGE_FAILED_TOKEN_NOT_FOUND, nil)
				return ctx.JSON(http.StatusUnauthorized, response)
			}

			if !strings.Contains(authHeader, "Bearer ") {
				response := utils.BuildResponseFailed(dto.MESSAGE_FAILED_PROSES_REQUEST, dto.MESSAGE_FAILED_TOKEN_NOT_VALID, nil)
				return ctx.JSON(http.StatusUnauthorized, response)
			}

			authHeader = strings.Replace(authHeader, "Bearer ", "", -1)
			token, err := jwtService.ValidateToken(authHeader)
			if err != nil {
				response := utils.BuildResponseFailed(dto.MESSAGE_FAILED_PROSES_REQUEST, dto.MESSAGE_FAILED_TOKEN_NOT_VALID, nil)
				return ctx.JSON(http.StatusUnauthorized, response)
			}

			if !token.Valid {
				response := utils.BuildResponseFailed(dto.MESSAGE_FAILED_PROSES_REQUEST, dto.MESSAGE_FAILED_DENIED_ACCESS, nil)
				return ctx.JSON(http.StatusUnauthorized, response)
			}

			userId, err := jwtService.GetUserIDByToken(authHeader)
			if err != nil {
				response := utils.BuildResponseFailed(dto.MESSAGE_FAILED_PROSES_REQUEST, err.Error(), nil)
				return ctx.JSON(http.StatusUnauthorized, response)
			}

			ctx.Set("token", authHeader)
			ctx.Set("user_id", userId)
			return next(ctx)
		}
	}
}
