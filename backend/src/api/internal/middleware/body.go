package middleware

import (
	"encoding/json"
	"luna-backend/api/internal/util"
	"luna-backend/errors"
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"
)

func parseBody[T any](c *gin.Context) {
	u := util.GetUtil(c)

	// Create object consumed by the handler
	obj := new(T)

	// Parse the body
	objType := reflect.TypeFor[T]()
	unmarshalIntoMap := objType.Kind() == reflect.Map && objType.Key().Kind() == reflect.String && objType.Elem().Kind() == reflect.Slice && objType.Elem().Elem().Kind() == reflect.Uint8

	var tr *errors.ErrorTrace
	if unmarshalIntoMap {
		// Unmarshal into map
		var mapped map[string]json.RawMessage
		mapped, tr = util.ParseIntoMap(c)
		if casted, ok := any(&mapped).(*T); ok {
			obj = casted
		} else {
			tr = errors.New().Status(http.StatusInternalServerError).
				Append(errors.LvlDebug, "Could not cast map to generic").
				AltStr(errors.LvlPlain, "Malformed request")
		}
	} else {
		// Unmarshal into struct
		tr = util.ParseIntoObject(c, obj)
	}
	if tr != nil {
		u.Error(tr)
		c.Abort()
		return
	}

	// Pass the object to the handler
	c.Set("body", obj)

	c.Next()
}

func parseQuery[T any](c *gin.Context) {
	u := util.GetUtil(c)

	// Create object consumed by the handler
	obj := new(T)

	// Parse the body
	err := c.ShouldBindQuery(obj)

	if err != nil {
		u.Error(
			errors.New().Status(http.StatusBadRequest).
				AddErr(errors.LvlDebug, err).
				Append(errors.LvlWordy, "Could not unmarshal body").
				Append(errors.LvlPlain, "Malformed request"),
		)
		c.Abort()
	}

	// Pass the object to the handler
	c.Set("query", obj)

	c.Next()
}

func WithBody[T any](handler func(c *gin.Context, obj *T)) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		parseBody[T],
		func(c *gin.Context) {
			// This is type-safe because it is set in the immediately previous middleware
			handler(c, c.MustGet("body").(*T))
		},
	}
}

func WithQuery[T any](handler func(c *gin.Context, obj *T)) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		parseQuery[T],
		func(c *gin.Context) {
			// This is type-safe because it is set in the immediately previous middleware
			handler(c, c.MustGet("query").(*T))
		},
	}
}

func WithBodyAndQuery[T any, S any](handler func(c *gin.Context, obj *T, obs *S)) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		parseBody[T],
		parseQuery[S],
		func(c *gin.Context) {
			// This is type-safe because it is set in the immediately previous middleware
			handler(c, c.MustGet("body").(*T), c.MustGet("query").(*S))
		},
	}
}
