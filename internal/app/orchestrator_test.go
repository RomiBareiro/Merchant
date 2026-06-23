package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"merchant-transactions-api/internal/app/mocks"
	"merchant-transactions-api/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateTransactionUseCases(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name             string
		scenario         string
		command          domain.CreateTransactionCommand
		nextIDs          []string
		expectedCard     string
		expectedStatus   string
		expectedDiscount string
		expectedTotal    string
		expectedPayDelay int
	}{
		{
			name:     "Given a debit card transaction when creating a transaction then the receivable is paid",
			scenario: "debit",
			command: domain.CreateTransactionCommand{
				Value:              "100",
				Description:        "T-Shirt Black M",
				Method:             string(domain.PaymentMethodDebitCard),
				CardNumber:         "1234 5678 9012 3456",
				CardHolderName:     "Juan Pérez",
				CardExpirationDate: "04/28",
				CardCvv:            "123",
			},
			nextIDs:          []string{"101", "102"},
			expectedCard:     "3456",
			expectedStatus:   "paid",
			expectedDiscount: "2.00",
			expectedTotal:    "98.00",
			expectedPayDelay: 0,
		},
		{
			name:     "Given a credit card transaction when creating a transaction then the receivable is waiting funds",
			scenario: "credit",
			command: domain.CreateTransactionCommand{
				Value:              "250.00",
				Description:        "Remera Negra M",
				Method:             string(domain.PaymentMethodCreditCard),
				CardNumber:         "2222333344445555",
				CardHolderName:     "Simplenube Store",
				CardExpirationDate: "10/25",
				CardCvv:            "222",
			},
			nextIDs:          []string{"201", "202"},
			expectedStatus:   "waiting_funds",
			expectedDiscount: "10.00",
			expectedTotal:    "240.00",
			expectedPayDelay: 30,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given
			txRepo := &mocks.TransactionRepository{}
			rcRepo := &mocks.ReceivableRepository{}
			numerator := &mocks.Numerator{}

			for _, id := range tc.nextIDs {
				numerator.On("NextID", mock.Anything).Return(id, nil).Once()
			}

			txRepo.On("SaveTransaction", mock.Anything, mock.Anything).Return(nil)
			rcRepo.On("SaveReceivable", mock.Anything, mock.Anything).Return(nil)

			orchestrator := NewOrchestrator(txRepo, rcRepo, numerator)

			// When
			transaction, receivable, err := orchestrator.CreateTransaction(context.Background(), tc.command)

			// Then
			require.NoError(t, err)

			if tc.scenario == "debit" {
				assert.Equal(t, tc.expectedCard, transaction.CardNumber)
			}
			assert.Equal(t, tc.expectedStatus, receivable.Status)
			assert.Equal(t, tc.expectedDiscount, receivable.Discount)
			assert.Equal(t, tc.expectedTotal, receivable.Total)

			createDate, err := time.Parse(time.RFC3339, receivable.CreateDate)
			require.NoError(t, err)
			payDate, err := time.Parse(time.RFC3339, receivable.PaymentDate)
			require.NoError(t, err)
			assert.Equal(t, createDate.AddDate(0, 0, tc.expectedPayDelay), payDate)

			txRepo.AssertExpectations(t)
			rcRepo.AssertExpectations(t)
			numerator.AssertExpectations(t)
		})
	}
}

func TestCreateTransactionRollsBackPersistedTransactionWhenReceivableCreationFails(t *testing.T) {
	t.Parallel()

	txRepo := &mocks.TransactionRepository{}
	rcRepo := &mocks.ReceivableRepository{}
	numerator := &mocks.Numerator{}
	orchestrator := NewOrchestrator(txRepo, rcRepo, numerator)
	command := domain.CreateTransactionCommand{
		Value:              "100",
		Description:        "Rollback case",
		Method:             string(domain.PaymentMethodDebitCard),
		CardNumber:         "1234 5678 9012 3456",
		CardHolderName:     "Test",
		CardExpirationDate: "04/28",
		CardCvv:            "123",
	}
	saveErr := errors.New("json-server failed")

	numerator.On("NextID", mock.Anything).Return("101", nil).Once()
	txRepo.On("SaveTransaction", mock.Anything, mock.MatchedBy(func(tx domain.Transaction) bool {
		return tx.ID == "101"
	})).Return(nil).Once()
	numerator.On("NextID", mock.Anything).Return("102", nil).Once()
	rcRepo.On("SaveReceivable", mock.Anything, mock.MatchedBy(func(receivable domain.Receivable) bool {
		return receivable.ID == "102" && receivable.TransactionID == "101"
	})).Return(saveErr).Once()
	txRepo.On("DeleteTransaction", mock.Anything, "101").Return(nil).Once()

	transaction, receivable, err := orchestrator.CreateTransaction(context.Background(), command)

	require.Error(t, err)
	require.Nil(t, transaction)
	require.Nil(t, receivable)
	require.Contains(t, err.Error(), "failed saving receivable")
	txRepo.AssertExpectations(t)
	rcRepo.AssertExpectations(t)
	numerator.AssertExpectations(t)
}
