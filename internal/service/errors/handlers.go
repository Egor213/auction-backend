package serverrs

import (
	repoerrs "auction-platform/internal/repo/errors"
	"errors"
)

func HandleRepoNotFound(err error, notFoundErr, genericErr error) error {
	if errors.Is(err, repoerrs.ErrNotFound) {
		return notFoundErr
	}
	if err != nil {
		return genericErr
	}
	return nil
}
