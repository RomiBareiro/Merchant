package app

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	"merchant-transactions-api/internal/domain"
	"merchant-transactions-api/internal/ports"
)

const defaultOperationTimeout = 10 * time.Second

type Orchestrator struct {
	transactionRepo  ports.TransactionRepository
	receivableRepo   ports.ReceivableRepository
	numeratorService ports.Numerator
}

func NewOrchestrator(transactionRepo ports.TransactionRepository, receivableRepo ports.ReceivableRepository, numeratorService ports.Numerator) *Orchestrator {
	return &Orchestrator{
		transactionRepo:  transactionRepo,
		receivableRepo:   receivableRepo,
		numeratorService: numeratorService,
	}
}

func (o *Orchestrator) CreateTransaction(ctx context.Context, command domain.CreateTransactionCommand) (*domain.Transaction, *domain.Receivable, error) {
	if err := command.Validate(); err != nil {
		return nil, nil, err
	}

	// Ensure context has a timeout for external API calls
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultOperationTimeout)
		defer cancel()
	}

	totalAmount, _ := domain.ParseAmount(command.Value)
	formattedValue := domain.FormatAmount(totalAmount)

	transactionID, err := o.numeratorService.NextID(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed generating transaction id: %w", err)
	}

	transaction := domain.Transaction{
		ID:                 transactionID,
		Value:              formattedValue,
		Description:        command.Description,
		Method:             domain.PaymentMethod(command.Method),
		CardNumber:         domain.Last4CardDigits(command.CardNumber),
		CardHolderName:     command.CardHolderName,
		CardExpirationDate: command.CardExpirationDate,
	}

	if err := o.transactionRepo.SaveTransaction(ctx, transaction); err != nil {
		return nil, nil, fmt.Errorf("failed saving transaction: %w", err)
	}

	receivableID, err := o.numeratorService.NextID(ctx)
	if err != nil {
		if delErr := o.rollbackTransaction(ctx, transaction.ID, err); delErr != nil {
			return nil, nil, fmt.Errorf("failed generating receivable id: %w (rollback also failed: %v)", err, delErr)
		}
		return nil, nil, fmt.Errorf("failed generating receivable id: %w", err)
	}

	createdAt := time.Now()
	status, discount, receivableTotal := computeReceivable(totalAmount, transaction.Method)
	receivable := domain.Receivable{
		ID:            receivableID,
		TransactionID: transactionID,
		Status:        status,
		CreateDate:    createdAt.Format(time.RFC3339),
		PaymentDate:   paymentDate(createdAt, transaction.Method).Format(time.RFC3339),
		Subtotal:      formattedValue,
		Discount:      discount,
		Total:         receivableTotal,
	}

	if err := o.receivableRepo.SaveReceivable(ctx, receivable); err != nil {
		if delErr := o.rollbackTransaction(ctx, transaction.ID, err); delErr != nil {
			return nil, nil, fmt.Errorf("failed saving receivable: %w (rollback also failed: %v)", err, delErr)
		}
		return nil, nil, fmt.Errorf("failed saving receivable: %w", err)
	}

	return &transaction, &receivable, nil
}

func computeReceivable(amount *big.Rat, method domain.PaymentMethod) (status, discount, total string) {
	var feePercent int64
	if method == domain.PaymentMethodDebitCard {
		status = "paid"
		feePercent = 2
	} else {
		status = "waiting_funds"
		feePercent = 4
	}

	feeRatio := new(big.Rat).SetFrac(big.NewInt(feePercent), big.NewInt(100))
	discountAmount := new(big.Rat).Mul(amount, feeRatio)
	totalAmount := new(big.Rat).Sub(amount, discountAmount)

	return status, domain.FormatAmount(discountAmount), domain.FormatAmount(totalAmount)
}

func paymentDate(createdAt time.Time, method domain.PaymentMethod) time.Time {
	if method == domain.PaymentMethodCreditCard {
		return createdAt.AddDate(0, 0, 30)
	}
	return createdAt
}

func (o *Orchestrator) rollbackTransaction(ctx context.Context, transactionID string, originalErr error) error {
	if delErr := o.transactionRepo.DeleteTransaction(ctx, transactionID); delErr != nil {
		log.Printf("orchestrator: compensating delete failed for tx %s: %v (original error: %v)", transactionID, delErr, originalErr)
		return delErr
	}
	return nil
}
