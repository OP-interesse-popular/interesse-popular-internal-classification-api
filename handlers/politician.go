package handlers

import (
	"net/http"

	"github.com/OP-interesse-popular/interesse-popular-internal-classification/models"
	"github.com/OP-interesse-popular/interesse-popular-internal-classification/repositories"
	"github.com/TiB-Software/go-commons/commonsmodels"
	commonserrors "github.com/TiB-Software/go-commons/errors"
	commonsutils "github.com/TiB-Software/go-commons/utils"
	"github.com/gin-gonic/gin"
)

type Politicians struct {
	repository repositories.PoliticianRepository
}

func NewPoliticians(repository repositories.PoliticianRepository) *Politicians {
	return &Politicians{repository: repository}
}
func (h *Politicians) List() gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := h.repository.List()
		if err != nil {
			commonsutils.SendErrorResponse(c, commonserrors.NewApiError(http.StatusInternalServerError, commonserrors.InternalServerError(err.Error())))
			return
		}
		commonsutils.SendResponse(c, commonsmodels.ResponseList[models.Politician]{Items: items, Total: len(items)}, http.StatusOK)
	}
}
func (h *Politicians) Get() gin.HandlerFunc {
	return func(c *gin.Context) {
		item, err := h.repository.FindByID(c.Param("id"))
		if err != nil {
			commonsutils.SendErrorResponse(c, commonserrors.NewApiError(http.StatusNotFound, commonserrors.NotFoundError("politician not found")))
			return
		}
		commonsutils.SendResponse(c, item, http.StatusOK)
	}
}
