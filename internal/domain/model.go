package domain

import "strings"

type PaymentMethod string

const (
	PaymentMethodDebitCard  PaymentMethod = "debit_card"
	PaymentMethodCreditCard PaymentMethod = "credit_card"
)

type Transaction struct {
	ID                 string        `json:"id"`
	Value              string        `json:"value"`
	Description        string        `json:"description"`
	Method             PaymentMethod `json:"method"`
	CardNumber         string        `json:"cardNumber"`
	CardHolderName     string        `json:"cardHolderName"`
	CardExpirationDate string        `json:"cardExpirationDate"`
	CardCvv            string        `json:"-"`
}

type Receivable struct {
	ID            string `json:"id"`
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
	CreateDate    string `json:"create_date"`
	PaymentDate   string `json:"payment_date"`
	Subtotal      string `json:"subtotal"`
	Discount      string `json:"discount"`
	Total         string `json:"total"`
}

type CreateTransactionCommand struct {
	Value              string `json:"value"`
	Description        string `json:"description"`
	Method             string `json:"method"`
	CardNumber         string `json:"cardNumber"`
	CardHolderName     string `json:"cardHolderName"`
	CardExpirationDate string `json:"cardExpirationDate"`
	CardCvv            string `json:"cardCvv"`
}

func Last4CardDigits(cardNumber string) string {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, cardNumber)
	if len(digits) <= 4 {
		return digits
	}
	return digits[len(digits)-4:]
}
