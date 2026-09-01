package user

import (
	"github.com/Caknoooo/go-gin-clean-starter/middlewares"
	"github.com/Caknoooo/go-gin-clean-starter/modules/auth/service"
	"github.com/Caknoooo/go-gin-clean-starter/modules/user/controller"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/constants"
	"github.com/labstack/echo/v4"
	"github.com/samber/do"
)

func RegisterRoutes(server *echo.Echo, injector *do.Injector) {
	userController := do.MustInvoke[controller.UserController](injector)
	jwtService := do.MustInvokeNamed[service.JWTService](injector, constants.JWTService)

	userRoutes := server.Group("/api/user")
	{
		userRoutes.GET("", userController.GetAllUser)
		userRoutes.GET("/me", userController.Me, middlewares.Authenticate(jwtService))
		userRoutes.PUT("/:id", userController.Update, middlewares.Authenticate(jwtService))
		userRoutes.DELETE("/:id", userController.Delete, middlewares.Authenticate(jwtService))
	}
}
