package ports

//go:generate mockery --all --output=../app/mocks --outpkg=mocks

import (
	"context"
	"merchant-transactions-api/internal/domain"
)

type TransactionRepository interface {
	SaveTransaction(ctx context.Context, transaction domain.Transaction) error
	GetTransaction(ctx context.Context, id string) (*domain.Transaction, error)
	ListTransactions(ctx context.Context) ([]domain.Transaction, error)
	DeleteTransaction(ctx context.Context, id string) error
}

type ReceivableRepository interface {
	SaveReceivable(ctx context.Context, receivable domain.Receivable) error
	GetReceivable(ctx context.Context, id string) (*domain.Receivable, error)
	ListReceivables(ctx context.Context) ([]domain.Receivable, error)
	DeleteReceivable(ctx context.Context, id string) error
}

type Numerator interface {
	NextID(ctx context.Context) (string, error)
}
