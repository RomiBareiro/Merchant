package domain

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

var (
	ErrInvalidTransactionValue       = errors.New("invalid transaction value")
	ErrInvalidPaymentMethod          = errors.New("invalid payment method")
	ErrInvalidCardNumber             = errors.New("invalid card number")
	ErrInvalidCardExpirationDate     = errors.New("invalid card expiration date")
	ErrInvalidCardCvv                = errors.New("invalid card cvv")
	ErrInvalidTransactionDescription = errors.New("invalid transaction description")
)

var expiryPattern = regexp.MustCompile(`^(0[1-9]|1[0-2])/(\d{2})$`)

func (c *CreateTransactionCommand) Validate() error {
	if strings.TrimSpace(c.Description) == "" {
		return ErrInvalidTransactionDescription
	}

	if _, err := ParseAmount(c.Value); err != nil {
		return ErrInvalidTransactionValue
	}

	switch PaymentMethod(c.Method) {
	case PaymentMethodDebitCard, PaymentMethodCreditCard:
	default:
		return ErrInvalidPaymentMethod
	}

	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, c.CardNumber)
	if len(digits) < 4 {
		return ErrInvalidCardNumber
	}

	if !expiryPattern.MatchString(c.CardExpirationDate) {
		return ErrInvalidCardExpirationDate
	}

	cvvDigits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, c.CardCvv)
	if len(cvvDigits) < 3 || len(cvvDigits) > 4 {
		return ErrInvalidCardCvv
	}

	return nil
}

func ParseAmount(value string) (*big.Rat, error) {
	sanitized := strings.TrimSpace(value)
	if sanitized == "" {
		return nil, fmt.Errorf("empty amount")
	}
	rat := new(big.Rat)
	if _, ok := rat.SetString(sanitized); !ok {
		return nil, fmt.Errorf("invalid amount format")
	}
	if rat.Sign() <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}
	return rat, nil
}

func FormatAmount(value *big.Rat) string {
	return value.FloatString(2)
}
