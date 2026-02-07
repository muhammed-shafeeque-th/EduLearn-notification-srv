package utils

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"github.com/google/uuid"
	"strings"
	"time"
)

// GenerateID returns a new UUID string.
func GenerateID() string {
	return uuid.New().String()
}

// GenerateRandomOTP generates a random numeric OTP of the given length.
func GenerateRandomOTP(length int) (string, error) {
	if length < 4 || length > 12 {
		return "", errors.New("OTP length must be between 4-12")
	}
	const digits = "0123456789"
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	for i := 0; i < length; i++ {
		b[i] = digits[int(b[i])%10]
	}
	return string(b), nil
}

// IsEmail checks if the given string is a valid email address (simple check).
func IsEmail(email string) bool {
	// Not RFC compliant, just covers trivial invalid cases
	at := strings.Index(email, "@")
	dot := strings.LastIndex(email, ".")
	return at > 0 && dot > at+1 && dot < len(email)-1
}


// GetCurrentTimestamp returns the current UTC Unix timestamp.
func GetCurrentTimestamp() int64 {
	return time.Now().UTC().Unix()
}

// RandomToken generates a URL-safe random string with the provided byte length.
func RandomToken(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// StringSliceContains returns true if s is in the slice.
func StringSliceContains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// GenerateResetToken generates a secure reset token
func GenerateResetToken(length int) (string, error) {
	if length < 16 {
		length = 32 // default
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// MaskEmail masks an email address for privacy
func MaskEmail(email string) string {
	if len(email) < 5 {
		return "***@***"
	}
	
	atIndex := -1
	for i, c := range email {
		if c == '@' {
			atIndex = i
			break
		}
	}
	
	if atIndex < 3 {
		return email[:1] + "***" + email[atIndex:]
	}
	
	return email[:2] + "***" + email[atIndex:]
}

// MaskPhoneNumber masks a phone number for privacy
func MaskPhoneNumber(phone string) string {
	if len(phone) < 4 {
		return "***"
	}
	
	return phone[:3] + "****" + phone[len(phone)-2:]
}

// Contains checks if a string contains a substring (case-insensitive)
func Contains(s, substr string) bool {
	return containsIgnoreCase(s, substr)
}

func containsIgnoreCase(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

// Truncate truncates a string to a maximum length
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ValidateEmail performs basic email validation
func ValidateEmail(email string) bool {
	if len(email) < 3 {
		return false
	}
	
	atCount := 0
	dotAfterAt := false
	atIndex := -1
	
	for i, c := range email {
		if c == '@' {
			atCount++
			atIndex = i
		}
		if atIndex != -1 && c == '.' && i > atIndex {
			dotAfterAt = true
		}
	}
	
	return atCount == 1 && dotAfterAt && atIndex > 0 && atIndex < len(email)-2
}

// ValidatePhoneNumber performs basic phone number validation
func ValidatePhoneNumber(phone string) bool {
	if len(phone) < 10 || len(phone) > 15 {
		return false
	}
	
	// Check if it contains only digits, +, -, (, )
	for _, c := range phone {
		if !(c >= '0' && c <= '9') && c != '+' && c != '-' && c != '(' && c != ')' && c != ' ' {
			return false
		}
	}
	
	return true
}