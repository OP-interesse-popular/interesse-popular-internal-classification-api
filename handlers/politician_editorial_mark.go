package handlers

import (
	"errors"
	"net/http"

	"github.com/OP-interesse-popular/interesse-popular-internal-classification/models"
	"github.com/OP-interesse-popular/interesse-popular-internal-classification/repositories"
	"github.com/OP-interesse-popular/interesse-popular-internal-classification/services"
	"github.com/TiB-Software/go-commons/commonsmodels"
	commonserrors "github.com/TiB-Software/go-commons/errors"
	commonsutils "github.com/TiB-Software/go-commons/utils"
	"github.com/gin-gonic/gin"
)

type PoliticianEditorialMarks struct {
	service *services.PoliticianEditorialMarks
}

func NewPoliticianEditorialMarks(service *services.PoliticianEditorialMarks) *PoliticianEditorialMarks {
	return &PoliticianEditorialMarks{service: service}
}

func (h *PoliticianEditorialMarks) CreateMany() gin.HandlerFunc {
	return func(c *gin.Context) {
		var inputs []services.SavePoliticianEditorialMarkInput
		if err := c.ShouldBindJSON(&inputs); err != nil || len(inputs) == 0 {
			sendMarkError(c, commonserrors.NewApiError(http.StatusBadRequest, commonserrors.InvalidDecodeJsonError("request body must be a non-empty JSON array")))
			return
		}
		results := h.service.CreateMany(inputs)
		commonsutils.SendResponse(c, commonsmodels.ResponseList[services.PoliticianEditorialMarkBulkResult]{Items: results, Total: len(results)}, http.StatusMultiStatus)
	}
}
func (h *PoliticianEditorialMarks) List() gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := h.service.List()
		if err != nil {
			h.respond(c, nil, err)
			return
		}
		commonsutils.SendResponse(c, commonsmodels.ResponseList[models.PoliticianEditorialMark]{Items: items, Total: len(items)}, http.StatusOK)
	}
}
func (h *PoliticianEditorialMarks) UpdateMany() gin.HandlerFunc {
	return func(c *gin.Context) {
		var inputs []services.UpdatePoliticianEditorialMarkInput
		if err := c.ShouldBindJSON(&inputs); err != nil || len(inputs) == 0 {
			sendMarkError(c, commonserrors.NewApiError(http.StatusBadRequest, commonserrors.InvalidDecodeJsonError("request body must be a non-empty JSON array")))
			return
		}
		results := h.service.UpdateMany(inputs)
		commonsutils.SendResponse(c, commonsmodels.ResponseList[services.PoliticianEditorialMarkBulkResult]{Items: results, Total: len(results)}, http.StatusMultiStatus)
	}
}
func (h *PoliticianEditorialMarks) Delete() gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := h.service.Delete(c.Param("id")); err != nil {
			h.respond(c, nil, err)
			return
		}
		commonsutils.SendResponse(c, commonsmodels.NewResponseSuccess(), http.StatusOK)
	}
}
func (h *PoliticianEditorialMarks) respond(c *gin.Context, payload any, err error) {
	if err == nil {
		commonsutils.SendResponse(c, payload, http.StatusOK)
		return
	}
	if errors.Is(err, repositories.ErrNotFound) {
		sendMarkError(c, commonserrors.NewApiError(http.StatusNotFound, commonserrors.NotFoundError("politician or editorial mark not found")))
		return
	}
	if errors.Is(err, services.ErrInvalidInput) {
		sendMarkError(c, commonserrors.NewApiError(http.StatusBadRequest, commonserrors.BadRequestError(err.Error())))
		return
	}
	sendMarkError(c, commonserrors.NewApiError(http.StatusInternalServerError, commonserrors.InternalServerError(err.Error())))
}
func sendMarkError(c *gin.Context, err commonserrors.ApiError) {
	commonsutils.SendErrorResponse(c, err)
}
