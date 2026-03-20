package service

import (
	"log"
)

// NotificationService handles sending notifications to users.
// Currently logs to stdout. In production, replace with SMTP or email API.
type NotificationService struct{}

func NewNotificationService() *NotificationService {
	return &NotificationService{}
}

// SendWelcome sends a welcome email after registration.
func (s *NotificationService) SendWelcome(email string) {
	log.Printf("[NOTIFICATION] Welcome email to %s: Welcome to Mantis Exchange! Your account has been created.", email)
}

// SendLoginAlert sends a login notification.
func (s *NotificationService) SendLoginAlert(email, ip string) {
	log.Printf("[NOTIFICATION] Login alert to %s: New login from IP %s", email, ip)
}

// SendWithdrawalRequest notifies about a withdrawal request.
func (s *NotificationService) SendWithdrawalRequest(email, asset, amount string) {
	log.Printf("[NOTIFICATION] Withdrawal request to %s: %s %s withdrawal requested. Please check your dashboard.", email, amount, asset)
}

// SendTradeConfirmation notifies about a completed trade.
func (s *NotificationService) SendTradeConfirmation(email, symbol, side, quantity, price string) {
	log.Printf("[NOTIFICATION] Trade confirmation to %s: %s %s %s @ %s executed", email, side, quantity, symbol, price)
}

// SendTOTPEnabled notifies when 2FA is enabled.
func (s *NotificationService) SendTOTPEnabled(email string) {
	log.Printf("[NOTIFICATION] Security alert to %s: Two-factor authentication has been enabled on your account.", email)
}

// SendLargeTradeAlert notifies admin about a large trade.
func (s *NotificationService) SendLargeTradeAlert(symbol, quantity, price, userEmail string) {
	log.Printf("[NOTIFICATION] ADMIN ALERT: Large trade by %s: %s %s @ %s", userEmail, quantity, symbol, price)
}
