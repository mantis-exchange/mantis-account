package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/mantis-exchange/mantis-account/internal/service"
)

type Handler struct {
	auth    *service.AuthService
	balance *service.BalanceService
	notify  *service.NotificationService
}

func New(auth *service.AuthService, balance *service.BalanceService, notify *service.NotificationService) *Handler {
	return &Handler{auth: auth, balance: balance, notify: notify}
}

type registerReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

func (h *Handler) Register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.auth.Register(c.Request.Context(), service.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	h.notify.SendWelcome(req.Email)

	c.JSON(http.StatusCreated, gin.H{
		"user_id": user.ID,
		"email":   user.Email,
	})
}

type loginReq struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	TOTPCode string `json:"totp_code"`
}

func (h *Handler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Check if TOTP is enabled
	hasTOTP, _ := h.auth.HasTOTP(c.Request.Context(), resp.UserID)
	if hasTOTP {
		if req.TOTPCode == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "totp_required", "totp_required": true})
			return
		}
		valid, _ := h.auth.VerifyTOTP(c.Request.Context(), resp.UserID, req.TOTPCode)
		if !valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid TOTP code"})
			return
		}
	}

	h.notify.SendLoginAlert(req.Email, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"token":   resp.Token,
		"user_id": resp.UserID,
	})
}

func (h *Handler) GetBalances(c *gin.Context) {
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	balances, err := h.balance.GetBalances(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"balances": balances})
}

func (h *Handler) GetProfile(c *gin.Context) {
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	user, err := h.auth.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	hasTOTP, _ := h.auth.HasTOTP(c.Request.Context(), userID)

	c.JSON(http.StatusOK, gin.H{
		"user_id":     user.ID,
		"email":       user.Email,
		"is_verified": user.IsVerified,
		"has_totp":    hasTOTP,
		"has_api_key": user.APIKey != nil,
		"created_at":  user.CreatedAt,
	})
}

func (h *Handler) GenerateAPIKeys(c *gin.Context) {
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	apiKey, apiSecret, err := h.auth.GenerateAPIKeys(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"api_key":    apiKey,
		"api_secret": apiSecret,
		"warning":    "Store the api_secret securely. It will not be shown again.",
	})
}

func (h *Handler) LookupAPIKey(c *gin.Context) {
	apiKey := c.Query("api_key")
	if apiKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "api_key required"})
		return
	}

	user, err := h.auth.LookupByAPIKey(c.Request.Context(), apiKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "invalid api key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":    user.ID,
		"api_secret": user.APISecret,
	})
}

type balanceOpReq struct {
	UserID string `json:"user_id" binding:"required"`
	Asset  string `json:"asset" binding:"required"`
	Amount string `json:"amount" binding:"required"`
}

func (h *Handler) FreezeBalance(c *gin.Context) {
	var req balanceOpReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	if err := h.balance.Freeze(c.Request.Context(), userID, req.Asset, req.Amount); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handler) UnfreezeBalance(c *gin.Context) {
	var req balanceOpReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	if err := h.balance.Unfreeze(c.Request.Context(), userID, req.Asset, req.Amount); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handler) CreditBalance(c *gin.Context) {
	var req balanceOpReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	if err := h.balance.Credit(c.Request.Context(), userID, req.Asset, req.Amount); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handler) ListUsers(c *gin.Context) {
	users, err := h.auth.ListUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}

type totpVerifyReq struct {
	Code string `json:"code" binding:"required"`
}

func (h *Handler) EnableTOTP(c *gin.Context) {
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	secret, url, err := h.auth.EnableTOTP(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"secret":  secret,
		"otpauth": url,
		"message": "Scan the QR code or enter the secret in your authenticator app, then verify with /account/totp/verify",
	})
}

func (h *Handler) VerifyTOTP(c *gin.Context) {
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	var req totpVerifyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	valid, err := h.auth.VerifyTOTP(c.Request.Context(), userID, req.Code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid TOTP code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"verified": true})
}

func (h *Handler) DisableTOTP(c *gin.Context) {
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	var req totpVerifyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.auth.DisableTOTP(c.Request.Context(), userID, req.Code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"disabled": true})
}

func (h *Handler) Faucet(c *gin.Context) {
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	// Credit testnet funds
	assets := map[string]string{
		"USDT": "10000",
		"BTC":  "0.5",
		"ETH":  "5",
	}

	for asset, amount := range assets {
		_ = h.balance.Credit(c.Request.Context(), userID, asset, amount)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Testnet funds credited",
		"assets":  assets,
	})
}

func (h *Handler) DeductFrozenBalance(c *gin.Context) {
	var req balanceOpReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	if err := h.balance.DeductFrozen(c.Request.Context(), userID, req.Asset, req.Amount); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
