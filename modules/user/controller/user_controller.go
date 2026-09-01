package controller

import (
	"net/http"

	"github.com/Caknoooo/go-gin-clean-starter/modules/user/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/user/query"
	"github.com/Caknoooo/go-gin-clean-starter/modules/user/service"
	"github.com/Caknoooo/go-gin-clean-starter/modules/user/validation"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/binding"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/constants"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/pagination"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/labstack/echo/v4"
	"github.com/samber/do"
	"gorm.io/gorm"
)

type (
	UserController interface {
		Me(ctx echo.Context) error
		GetAllUser(ctx echo.Context) error
		Update(ctx echo.Context) error
		Delete(ctx echo.Context) error
	}

	userController struct {
		userService    service.UserService
		userValidation *validation.UserValidation
		db             *gorm.DB
	}
)

func NewUserController(injector *do.Injector, us service.UserService) UserController {
	db := do.MustInvokeNamed[*gorm.DB](injector, constants.DB)
	userValidation := validation.NewUserValidation()
	return &userController{
		userService:    us,
		userValidation: userValidation,
		db:             db,
	}
}

func (c *userController) GetAllUser(ctx echo.Context) error {
	var filter = &query.UserFilter{}
	filter.BindPagination(ctx)

	binding.BindQuery(ctx, filter)

	users, total, err := pagination.PaginatedQueryWithIncludable[query.User](c.db, filter)
	if err != nil {
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_GET_USER, err.Error(), nil)
		return ctx.JSON(http.StatusBadRequest, res)
	}

	paginationResponse := pagination.CalculatePagination(filter.Pagination, total)
	response := pagination.NewPaginatedResponse(http.StatusOK, dto.MESSAGE_SUCCESS_GET_LIST_USER, users, paginationResponse)
	return ctx.JSON(http.StatusOK, response)
}

func (c *userController) Me(ctx echo.Context) error {
	userId := ctx.Get("user_id").(string)

	result, err := c.userService.GetUserById(ctx.Request().Context(), userId)
	if err != nil {
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_GET_USER, err.Error(), nil)
		return ctx.JSON(http.StatusBadRequest, res)
	}

	res := utils.BuildResponseSuccess(dto.MESSAGE_SUCCESS_GET_USER, result)
	return ctx.JSON(http.StatusOK, res)
}

func (c *userController) Update(ctx echo.Context) error {
	var req dto.UserUpdateRequest
	if err := ctx.Bind(&req); err != nil {
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_GET_DATA_FROM_BODY, err.Error(), nil)
		return ctx.JSON(http.StatusBadRequest, res)
	}

	if err := c.userValidation.ValidateUserUpdateRequest(req); err != nil {
		res := utils.BuildResponseFailed("Validation failed", err.Error(), nil)
		return ctx.JSON(http.StatusBadRequest, res)
	}

	userId := ctx.Get("user_id").(string)
	result, err := c.userService.Update(ctx.Request().Context(), req, userId)
	if err != nil {
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_UPDATE_USER, err.Error(), nil)
		return ctx.JSON(http.StatusBadRequest, res)
	}

	res := utils.BuildResponseSuccess(dto.MESSAGE_SUCCESS_UPDATE_USER, result)
	return ctx.JSON(http.StatusOK, res)
}

func (c *userController) Delete(ctx echo.Context) error {
	userId := ctx.Get("user_id").(string)

	if err := c.userService.Delete(ctx.Request().Context(), userId); err != nil {
		res := utils.BuildResponseFailed(dto.MESSAGE_FAILED_DELETE_USER, err.Error(), nil)
		return ctx.JSON(http.StatusBadRequest, res)
	}

	res := utils.BuildResponseSuccess(dto.MESSAGE_SUCCESS_DELETE_USER, nil)
	return ctx.JSON(http.StatusOK, res)
}
