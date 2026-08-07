package middleware

import (
	"luna-backend/types"

	"github.com/gin-gonic/gin"
)

func Prepend(first gin.HandlerFunc, rest []gin.HandlerFunc) []gin.HandlerFunc {
	return append([]gin.HandlerFunc{first}, rest...)
}

func RequirePermissionAndBody[T any](perm types.Permission, handler func(c *gin.Context, body *T)) []gin.HandlerFunc {
	return Prepend(RequirePermissions(perm), WithBody(handler))
}

func RequirePermissionAndQuery[T any](perm types.Permission, handler func(c *gin.Context, query *T)) []gin.HandlerFunc {
	return Prepend(RequirePermissions(perm), WithQuery(handler))
}

func RequirePermissionAndBodyAndQuery[T any, S any](perm types.Permission, handler func(c *gin.Context, body *T, query *S)) []gin.HandlerFunc {
	return Prepend(RequirePermissions(perm), WithBodyAndQuery(handler))
}
