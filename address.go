package blockwatch

import (
	"context"
	"errors"
)

var ErrAddressAlreadySaved = errors.New("address already saved")

type AddressStorage interface {
	SaveAddress(ctx context.Context, address string) error
	ListAddresses(ctx context.Context) ([]string, error)
}
