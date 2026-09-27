package handlers

import (
	"net/http"

	"github.com/bus-booking/BE-bus-booking-platform/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// AdminHandler handles admin-related HTTP requests
type AdminHandler struct {
	DB *sqlx.DB
}

// NewAdminHandler creates a new AdminHandler with the given database connection
func NewAdminHandler(db *sqlx.DB) *AdminHandler {
	return &AdminHandler{DB: db}
}

// GetExchangeRate godoc
// @Summary Récupère le taux de change actuel USD→CDF
// @description Retourne le taux de change configuré par le Super Admin
// @id get-exchange-rate
// @tags admin
// @produce json
// @success 200 {object} map[string]float64
// @failure 500 {object} map[string]string
// @router /admin/exchange-rate [get]
func (h *AdminHandler) GetExchangeRate(c *gin.Context) {
	rate, err := models.GetCurrent(h.DB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"usd_to_cdf": rate.USDToCDF,
		"cdf_to_usd": 1 / rate.USDToCDF,
	})
}

// UpdateExchangeRate godoc
// @Summary Met à jour le taux de change USD→CDF
// @description Permet au Super Admin de modifier le taux de change
// @id update-exchange-rate
// @tags admin
// @accept json
// @produce json
// @param rate body map[string]float64 true "Nouveau taux de change"
// @success 200 {object} map[string]string
// @failure 400 {object} map[string]string
// @failure 500 {object} map[string]string
// @router /admin/exchange-rate [put]
func (h *AdminHandler) UpdateExchangeRate(c *gin.Context) {
	var payload struct {
		USDToCDF float64 `json:"usd_to_cdf" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload invalide: " + err.Error()})
		return
	}

	// TODO: Récupérer l'utilisateur depuis le contexte (à adapter selon votre système d'auth)
	// Pour l'instant, nous utilisons un placeholder. En production, vous devriez extraire l'ID utilisateur du JWT ou du contexte.
	userID := "super_admin" // À remplacer par l'ID réel de l'utilisateur authentifié

	current, err := models.GetCurrent(h.DB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := current.Update(h.DB, payload.USDToCDF, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Taux de change mis à jour avec succès"})
}