package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"

	"github.com/mantis-exchange/mantis-account/internal/model"
)

type AuthService struct {
	users     *model.UserRepo
	jwtSecret []byte
	jwtExpiry time.Duration
}

func NewAuthService(users *model.UserRepo, jwtSecret string, jwtExpiry time.Duration) *AuthService {
	return &AuthService{
		users:     users,
		jwtSecret: []byte(jwtSecret),
		jwtExpiry: jwtExpiry,
	}
}

type RegisterRequest struct {
	Email    string
	Password string
}

type LoginResponse struct {
	Token  string    `json:"token"`
	UserID uuid.UUID `json:"user_id"`
}

func (s *AuthService) Register(ctx context.Context, req RegisterRequest) (*model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now()
	user := &model.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: string(hash),
		IsVerified:   false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	token, err := s.generateJWT(user.ID)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{Token: token, UserID: user.ID}, nil
}

func (s *AuthService) ValidateToken(tokenStr string) (uuid.UUID, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return uuid.Nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return uuid.Nil, fmt.Errorf("invalid token")
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return uuid.Nil, fmt.Errorf("invalid token claims")
	}

	return uuid.Parse(sub)
}

func (s *AuthService) GenerateAPIKeys(ctx context.Context, userID uuid.UUID) (string, string, error) {
	apiKey := generateRandomHex(20)
	apiSecret := generateRandomHex(32)

	if err := s.users.UpdateAPIKeys(ctx, userID, apiKey, apiSecret); err != nil {
		return "", "", err
	}

	return apiKey, apiSecret, nil
}

func (s *AuthService) LookupByAPIKey(ctx context.Context, apiKey string) (*model.User, error) {
	return s.users.GetByAPIKey(ctx, apiKey)
}

func (s *AuthService) GetUser(ctx context.Context, userID uuid.UUID) (*model.User, error) {
	return s.users.GetByID(ctx, userID)
}

func (s *AuthService) ListUsers(ctx context.Context) ([]model.User, error) {
	return s.users.ListAll(ctx)
}

// EnableTOTP generates a new TOTP secret for the user.
func (s *AuthService) EnableTOTP(ctx context.Context, userID uuid.UUID) (string, string, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return "", "", fmt.Errorf("user not found")
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Mantis Exchange",
		AccountName: user.Email,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to generate TOTP: %w", err)
	}

	if err := s.users.UpdateTOTPSecret(ctx, userID, key.Secret()); err != nil {
		return "", "", err
	}

	return key.Secret(), key.URL(), nil
}

// VerifyTOTP validates a TOTP code. Returns true if valid.
func (s *AuthService) VerifyTOTP(ctx context.Context, userID uuid.UUID, code string) (bool, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}
	if user.TOTPSecret == nil || *user.TOTPSecret == "" {
		return false, fmt.Errorf("TOTP not enabled")
	}
	return totp.Validate(code, *user.TOTPSecret), nil
}

// DisableTOTP removes TOTP from the user account.
func (s *AuthService) DisableTOTP(ctx context.Context, userID uuid.UUID, code string) error {
	valid, err := s.VerifyTOTP(ctx, userID, code)
	if err != nil {
		return err
	}
	if !valid {
		return fmt.Errorf("invalid TOTP code")
	}
	return s.users.UpdateTOTPSecret(ctx, userID, "")
}

// HasTOTP checks if a user has TOTP enabled.
func (s *AuthService) HasTOTP(ctx context.Context, userID uuid.UUID) (bool, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}
	return user.TOTPSecret != nil && *user.TOTPSecret != "", nil
}

func (s *AuthService) generateJWT(userID uuid.UUID) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID.String(),
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(s.jwtExpiry).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func generateRandomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
