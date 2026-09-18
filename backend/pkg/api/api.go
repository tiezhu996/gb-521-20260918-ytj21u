package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AppError struct {
	Status  int         `json:"-"`
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
	Cause   error       `json:"-"`
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Cause }

type Envelope struct {
	Data      interface{} `json:"data,omitempty"`
	Error     *AppError   `json:"error,omitempty"`
	RequestID string      `json:"request_id"`
	Meta      interface{} `json:"meta,omitempty"`
}

type PageMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

func NewError(status int, code, message string, cause error) *AppError {
	return &AppError{Status: status, Code: code, Message: message, Cause: cause}
}

func BadRequest(code, message string, details interface{}) *AppError {
	return &AppError{Status: http.StatusBadRequest, Code: code, Message: message, Details: details}
}

func Unauthorized(message string) *AppError {
	return &AppError{Status: http.StatusUnauthorized, Code: "AUTH_REQUIRED", Message: message}
}

func Forbidden(message string) *AppError {
	return &AppError{Status: http.StatusForbidden, Code: "ACCESS_DENIED", Message: message}
}

func NotFound(entity string) *AppError {
	return &AppError{Status: http.StatusNotFound, Code: "NOT_FOUND", Message: entity + "不存在"}
}

func Conflict(code, message string) *AppError {
	return &AppError{Status: http.StatusConflict, Code: code, Message: message}
}

func Unprocessable(code, message string, details interface{}) *AppError {
	return &AppError{Status: http.StatusUnprocessableEntity, Code: code, Message: message, Details: details}
}

func Internal(cause error) *AppError {
	return NewError(http.StatusInternalServerError, "INTERNAL_ERROR", "服务暂时无法完成请求", cause)
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Envelope{Data: data, RequestID: RequestID(c)})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Envelope{Data: data, RequestID: RequestID(c)})
}

func Page(c *gin.Context, data interface{}, page, pageSize int, total int64) {
	totalPages := 0
	if pageSize > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}
	c.JSON(http.StatusOK, Envelope{
		Data: data, RequestID: RequestID(c),
		Meta: PageMeta{Page: page, PageSize: pageSize, Total: total, TotalPages: totalPages},
	})
}

func NoContent(c *gin.Context) { c.Status(http.StatusNoContent) }

func Fail(c *gin.Context, err error) {
	var appErr *AppError
	if !errors.As(err, &appErr) {
		appErr = Internal(err)
	}
	c.AbortWithStatusJSON(appErr.Status, Envelope{Error: appErr, RequestID: RequestID(c)})
}

func BindError(c *gin.Context, err error) {
	Fail(c, BadRequest("INVALID_REQUEST", "请求字段不符合要求，请检查格式与范围", err.Error()))
}

func RequestID(c *gin.Context) string {
	if value, ok := c.Get("request_id"); ok {
		if id, valid := value.(string); valid {
			return id
		}
	}
	return ""
}
