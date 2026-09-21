// Package httpx holds the response envelope, error shape, and pagination
// helpers shared by every feature's Handler layer, per the conventions in
// api-spec.md section 3.
package httpx

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Meta is the pagination block attached to list responses.
type Meta struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	TotalItems int `json:"totalItems"`
	TotalPages int `json:"totalPages"`
}

// List wraps a paginated list response: { "data": [...], "meta": {...} }.
func List(c *gin.Context, data any, meta Meta) {
	c.JSON(http.StatusOK, gin.H{"data": data, "meta": meta})
}

// Item wraps a single-resource response: the resource returned directly.
func Item(c *gin.Context, status int, data any) {
	c.JSON(status, data)
}

// FieldError is one entry in a validation_error's "fields" array.
type FieldError struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

// Error writes the standard { "code", "message" } error envelope.
func Error(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"code": code, "message": message})
}

// ValidationError writes { "code": "validation_error", "message", "fields" }.
func ValidationError(c *gin.Context, message string, fields []FieldError) {
	c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
		"code":    "validation_error",
		"message": message,
		"fields":  fields,
	})
}

// Pagination reads page/pageSize query params with the defaults and cap
// from api-spec.md: page default 1, pageSize default 20, max 100.
func Pagination(c *gin.Context) (page, pageSize int) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err = strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if err != nil || pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

// BuildMeta computes the Meta block for a slice of length totalItems given
// the page/pageSize that were actually applied.
func BuildMeta(page, pageSize, totalItems int) Meta {
	totalPages := totalItems / pageSize
	if totalItems%pageSize != 0 {
		totalPages++
	}
	if totalPages == 0 {
		totalPages = 1
	}
	return Meta{Page: page, PageSize: pageSize, TotalItems: totalItems, TotalPages: totalPages}
}
