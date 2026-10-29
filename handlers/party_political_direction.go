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

type PartyPoliticalDirection struct {
	service *services.PartyPoliticalDirection
}

func NewPartyPoliticalDirection(service *services.PartyPoliticalDirection) *PartyPoliticalDirection {
	return &PartyPoliticalDirection{service: service}
}

func (h *PartyPoliticalDirection) CreateMany() gin.HandlerFunc {
	return func(c *gin.Context) {
		var inputs []services.SavePartyPoliticalDirectionInput
		if err := c.ShouldBindJSON(&inputs); err != nil || len(inputs) == 0 {
			commonsutils.SendErrorResponse(c, commonserrors.NewApiError(http.StatusBadRequest, commonserrors.InvalidDecodeJsonError("request body must be a non-empty JSON array")))
			return
		}
		results := h.service.CreateMany(inputs)
		commonsutils.SendResponse(c, commonsmodels.ResponseList[services.BulkResult]{Items: results, Total: len(results)}, http.StatusMultiStatus)
	}
}
func (h *PartyPoliticalDirection) List() gin.HandlerFunc {
	return func(c *gin.Context) {
		if direction := c.Query("direction"); direction != "" {
			items, err := h.service.ListActiveByDirectionAndIdeologies(direction, c.QueryArray("ideology"))
			h.respondList(c, items, err)
			return
		}
		items, err := h.service.List()
		h.respondList(c, items, err)
	}
}
func (h *PartyPoliticalDirection) Get() gin.HandlerFunc {
	return func(c *gin.Context) {
		item, err := h.service.FindByID(c.Param("id"))
		h.respond(c, item, err, http.StatusOK)
	}
}
func (h *PartyPoliticalDirection) UpdateMany() gin.HandlerFunc {
	return func(c *gin.Context) {
		var inputs []services.UpdatePartyPoliticalDirectionInput
		if err := c.ShouldBindJSON(&inputs); err != nil || len(inputs) == 0 {
			commonsutils.SendErrorResponse(c, commonserrors.NewApiError(http.StatusBadRequest, commonserrors.InvalidDecodeJsonError("request body must be a non-empty JSON array")))
			return
		}
		results := h.service.UpdateMany(inputs)
		commonsutils.SendResponse(c, commonsmodels.ResponseList[services.BulkResult]{Items: results, Total: len(results)}, http.StatusMultiStatus)
	}
}
func (h *PartyPoliticalDirection) Delete() gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := h.service.Delete(c.Param("id")); err != nil {
			h.respond(c, nil, err, http.StatusNoContent)
			return
		}
		commonsutils.SendResponse(c, commonsmodels.NewResponseSuccess(), http.StatusOK)
	}
}

func (h *PartyPoliticalDirection) respond(c *gin.Context, payload any, err error, successStatus int) {
	if err == nil {
		commonsutils.SendResponse(c, payload, successStatus)
		return
	}
	if errors.Is(err, repositories.ErrNotFound) {
		commonsutils.SendErrorResponse(c, commonserrors.NewApiError(http.StatusNotFound, commonserrors.NotFoundError("party political direction not found")))
		return
	}
	if errors.Is(err, services.ErrInvalidInput) {
		commonsutils.SendErrorResponse(c, commonserrors.NewApiError(http.StatusBadRequest, commonserrors.BadRequestError(err.Error())))
		return
	}
	commonsutils.SendErrorResponse(c, commonserrors.NewApiError(http.StatusInternalServerError, commonserrors.InternalServerError(err.Error())))
}

func (h *PartyPoliticalDirection) respondList(c *gin.Context, items []models.PartyPoliticalDirection, err error) {
	if err != nil {
		h.respond(c, nil, err, http.StatusOK)
		return
	}
	commonsutils.SendResponse(c, commonsmodels.ResponseList[models.PartyPoliticalDirection]{Items: items, Total: len(items)}, http.StatusOK)
}
