package cmd

import (
	"errors"

	"github.com/pvarganov/yuno-cli/internal/api"
	"github.com/pvarganov/yuno-cli/internal/config"
	"github.com/pvarganov/yuno-cli/internal/confirm"
)

// Process exit codes, shared with the sibling CLIs: 0 success, 1 API error,
// 2 confirmation declined, 3 configuration or input error.
const (
	ExitOK          = 0
	ExitAPIError    = 1
	ExitDeclined    = 2
	ExitConfigError = 3
)

// apiSentinels are the errors that mean "the API answered, and it said no".
var apiSentinels = []error{
	api.ErrUnauthorized,
	api.ErrForbidden,
	api.ErrNotFound,
	api.ErrRateLimited,
	api.ErrValidation,
	api.ErrServer,
}

// configSentinels are the errors that mean "fix your config or your flags".
var configSentinels = []error{
	config.ErrProfileNotFound,
	config.ErrMissingCredentials,
	config.ErrUnknownEnvironment,
}

// ExitCode maps an error returned by the root command onto a process exit code.
func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}

	if errors.Is(err, confirm.ErrDeclined) || errors.Is(err, confirm.ErrNotATerminal) {
		return ExitDeclined
	}

	if matchesAny(err, configSentinels) {
		return ExitConfigError
	}

	var apiErr *api.Error
	if errors.As(err, &apiErr) || matchesAny(err, apiSentinels) {
		return ExitAPIError
	}

	return ExitConfigError
}

// matchesAny reports whether err wraps any of the given sentinels.
func matchesAny(err error, sentinels []error) bool {
	for _, sentinel := range sentinels {
		if errors.Is(err, sentinel) {
			return true
		}
	}

	return false
}
