package blockproc

type Service interface {
}

type service struct {
	idempotencyGuard IdempotencyGuard

	walletStorage       WalletStorage
	transactionNotifier TransactionNotifier

	blockProcessedNotifier         BlockProcessedNotifier
	blockProcessingFailureNotifier BlockProcessingFailureNotifier
}

var _ Service = (*service)(nil)

func New() *service {
	return &service{}
}
