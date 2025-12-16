package entity

import (
	"crypto/rand"
	"errors"
	"io"
	"math/big"
	"time"

	domain_errors "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/errors"
)

const (
	OTPLength     = 6
	OTPExpiration = 10 * time.Minute
	MaxOTPAttempts = 3
)


// OTP entity with business logic
type OTP struct {
	Code      string    `json:"code"`
	Email     string    `json:"email"`
	UserId    string    `json:"userId"`
	ExpiresAt time.Time `json:"expiresAt"`
	Attempts  int       `json:"attempts"`
}

// NewOTP creates a new OTP with proper defaults
func NewOTP(userId, email string) (*OTP, error) {
	code, err := GenerateDefaultOTP()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	return &OTP{
		Code:      code,
		Email:     email,
		UserId:    userId,
		ExpiresAt: now.Add(OTPExpiration),
		Attempts:  0,
	}, nil
}



func (o OTP) IsExpired(now time.Time) bool {
	return now.After(o.ExpiresAt)
}


// IsValid checks if OTP is still valid
func (o *OTP) IsValid() bool {
	return time.Now().UTC().Before(o.ExpiresAt)
}

// IncrementAttempts increases attempt counter
func (o *OTP) IncrementAttempts() {
	o.Attempts++
}

// CanAttempt checks if more attempts are allowed
func (o *OTP) CanAttempt() bool {
	return o.Attempts < MaxOTPAttempts
}

// Verify checks if provided code matches
func (o *OTP) Verify(code string) error {

	if o.Code != code {
		return domain_errors.ErrInvalidOTP
	}
	if !o.IsValid() {
		return domain_errors.ErrOTPExpired
	}
	if !o.CanAttempt() {
		return errors.New("maximum OTP attempts exceeded")
	}
	
	o.IncrementAttempts()
	
	if o.Code != code {
		return domain_errors.ErrInvalidOTP
	}
	
	return nil
}

// GenerateOTP generates a 6-digit numeric OTP.
// randReader is injectable for testability (can pass rand.Reader in prod).
func GenerateOTP(randReader io.Reader) (string, error) {
	const otpLength = 6
	const digits = "0123456789"

	code := make([]byte, otpLength)

	for i := 0; i < otpLength; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}

		// Make sure first code digit is not less than 5
		if i == 0 {
			if num.Int64() < 5 {
				code[i] = digits[num.Int64()+5]
			} else {
				code[i] = digits[num.Int64()]
			}
		} else {
			code[i] = digits[num.Int64()]
		}
		// num, err := rand.Int(randReader, big.NewInt(int64(len(digits))))
		
		// if err != nil {
		// 	return "", err
		// }

		// code[i] = digits[num.Int64()]
	}
	return string(code), nil
}

// Convenience wrapper for production.
func GenerateDefaultOTP() (string, error) {
	return GenerateOTP(rand.Reader)
}
