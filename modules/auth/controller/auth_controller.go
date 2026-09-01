package controller

import (
	"net/http"

	"github.com/Caknoooo/go-gin-clean-starter/modules/auth/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/auth/service"
	"github.com/Caknoooo/go-gin-clean-starter/modules/auth/validation"
	userDto "github.com/Caknoooo/go-gin-clean-starter/modules/user/dto"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/constants"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/labstack/echo/v4"
	"github.com/samber/do"
	"gorm.io/gorm"
)

type (
	AuthController interface {
		Register(ctx echo.Context) error
		Login(ctx echo.Context) error
		RefreshToken(ctx echo.Context) error
		Logout(ctx echo.Context) error
		SendVerificationEmail(ctx echo.Context) error
		VerifyEmail(ctx echo.Context) error
		SendPasswordReset(ctx echo.Context) error
		ResetPassword(ctx echo.Context) error
	}

	authController struct {
		authService    service.AuthService
		authValidation *validation.AuthValidation
		db             *gorm.DB
	}
)

func NewAuthController(injector *do.Injector, as service.AuthService) AuthController {
	db := do.MustInvokeNamed[*gorm.DB](injector, constants.DB)
	authValidation := validation.NewAuthValidation()
	return &authController{
		authService:    as,
		authValidation: authValidation,
		db:             db,
	}
}

func (c *authController) Register(ctx echo.Context) error {
	var req userDto.UserCreateRequest
	if err := ctx.Bind(&req); err != nil {
		res := utils.BuildResponseFailed(userDto.MESSAGE_FAILED_GET_DATA_FROM_BODY, err.Error(), nil)
		return ctx.JSON(http.StatusBadRequest, res)
	}

	// Validate request
	if err := c.authValidation.ValidateRegisterRequest(req); err != nil {
		res := utils.BuildResponseFailed("Validation failed", err.Error(), nil)
		return ctx.JSON(http.StatusBadRequest, res)
	}

	result, err := c.authService.Register(ctx.Request().Context(), req)
	if err != nil {
		res := utils.BuildResponseFailed(userDto.MESSAGE_FAILED_REGISTER_USER, err.Error(), nil)
		return ctx.JSON(http.StatusBadRequest, res)
	}

	res := utils.BuildResponseSuccess(userDto.MESSAGE_SUCCESS_REGISTER_USER, result)
	return ctx.JSON(http.StatusOK, res)
}

func (c *authController) Login(ctx echo.Context) error {
	var req userDto.UserLoginRequest
	if err := ctx.Bind(&req); err != nil {
		response := utils.BuildResponseFailed(userDto.MESSAGE_FAILED_GET_DATA_FROM_BODY, err.Error(), nil)
		return ctx.JSON(http.StatusBadRequest, response)
	}

	// Validate request
	if err := c.authValidation.ValidateLoginRequest(req); err != nil {
		res := utils.BuildResponseFailed("Validation failed", err.Error(), nil)
		return ctx.JSON(http.StatusBadRequest, res)
	}

	result, err := c.authService.Login(ctx.Request().Context(), req)
	if err != nil {
		res := utils.BuildResponseFailed(userDto.MESSAGE_FAILED_LOGIN, err.Error(), nil)
		return ctx.JSON(http.StatusBadRequest, res)
	}

	res := utils.BuildResponseSuccess(userDto.MESSAGE_SUCCESS_LOGIN, result)
	return ctx.JSON(http.StatusOK, res)
}

func (c *authController) RefreshToken(ctx echo.Context) error {
	var req dto.RefreshTokenRequest
	if err := ctx.Bind(&req); err != nil {
		res := utils.BuildResponseFailed(userDto.MESSAGE_FAILED_GET_DATA_FROM_BODY, err.Error(), nil)
		return ctx.JSON(http.StatusBadRequest, res)
	}

	result, err := c.authService.RefreshToken(ctx.Request().Context(), req)
	if err != nil {
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_REFRESH_TOKEN, err.Error(), nil)
		return ctx.JSON(http.StatusUnauthorized, res)
	}

	res := utils.BuildResponseSuccess(dto.MESSAGE_SUCCESS_REFRESH_TOKEN, result)
	return ctx.JSON(http.StatusOK, res)
}

func (c *authController) Logout(ctx echo.Context) error {
	userId := ctx.Get("user_id").(string)

	err := c.authService.Logout(ctx.Request().Context(), userId)
	if err != nil {
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_LOGOUT, err.Error(), nil)
		return ctx.JSON(http.StatusBadRequest, res)
	}

	res := utils.BuildResponseSuccess(dto.MESSAGE_SUCCESS_LOGOUT, nil)
	return ctx.JSON(http.StatusOK, res)
}

func (c *authController) SendVerificationEmail(ctx echo.Context) error {
	var req userDto.SendVerificationEmailRequest
	if err := ctx.Bind(&req); err != nil {
		res := utils.BuildResponseFailed(userDto.MESSAGE_FAILED_GET_DATA_FROM_BODY, err.Error(), nil)
		return ctx.JSON(http.StatusBadRequest, res)
	}

	err := c.authService.SendVerificationEmail(ctx.Request().Context(), req)
	if err != nil {
		res := utils.BuildResponseFailed(userDto.MESSAGE_FAILED_PROSES_REQUEST, err.Error(), nil)
		return ctx.JSON(http.StatusBadRequest, res)
	}

	res := utils.BuildResponseSuccess(userDto.MESSAGE_SEND_VERIFICATION_EMAIL_SUCCESS, nil)
	return ctx.JSON(http.StatusOK, res)
}

func (c *authController) VerifyEmail(ctx echo.Context) error {
	var req userDto.VerifyEmailRequest
	if err := ctx.Bind(&req); err != nil {
		res := utils.BuildResponseFailed(userDto.MESSAGE_FAILED_GET_DATA_FROM_BODY, err.Error(), nil)
		return ctx.JSON(http.StatusBadRequest, res)
	}

	result, err := c.authService.VerifyEmail(ctx.Request().Context(), req)
	if err != nil {
		res := utils.BuildResponseFailed(userDto.MESSAGE_FAILED_VERIFY_EMAIL, err.Error(), nil)
		return ctx.JSON(http.StatusBadRequest, res)
	}

	res := utils.BuildResponseSuccess(userDto.MESSAGE_SUCCESS_VERIFY_EMAIL, result)
	return ctx.JSON(http.StatusOK, res)
}

func (c *authController) SendPasswordReset(ctx echo.Context) error {
	var req dto.SendPasswordResetRequest
	if err := ctx.Bind(&req); err != nil {
		res := utils.BuildResponseFailed(userDto.MESSAGE_FAILED_GET_DATA_FROM_BODY, err.Error(), nil)
		return ctx.JSON(http.StatusBadRequest, res)
	}

	err := c.authService.SendPasswordReset(ctx.Request().Context(), req)
	if err != nil {
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_SEND_PASSWORD_RESET, err.Error(), nil)
		return ctx.JSON(http.StatusBadRequest, res)
	}

	res := utils.BuildResponseSuccess(dto.MESSAGE_SUCCESS_SEND_PASSWORD_RESET, nil)
	return ctx.JSON(http.StatusOK, res)
}

func (c *authController) ResetPassword(ctx echo.Context) error {
	var req dto.ResetPasswordRequest
	if err := ctx.Bind(&req); err != nil {
		res := utils.BuildResponseFailed(userDto.MESSAGE_FAILED_GET_DATA_FROM_BODY, err.Error(), nil)
		return ctx.JSON(http.StatusBadRequest, res)
	}

	err := c.authService.ResetPassword(ctx.Request().Context(), req)
	if err != nil {
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_RESET_PASSWORD, err.Error(), nil)
		return ctx.JSON(http.StatusBadRequest, res)
	}

	res := utils.BuildResponseSuccess(dto.MESSAGE_SUCCESS_RESET_PASSWORD, nil)
	return ctx.JSON(http.StatusOK, res)
}
