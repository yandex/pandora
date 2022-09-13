package confutil

import (
	"errors"
	"fmt"
	"os"
)

var ErrEnvVariableNotProvided error = errors.New("env variable not set")

// Resolve custom tag token with env variable value
var EnvTagResolver TagResolver = envTokenResolver

func envTokenResolver(in string) (string, error) {
	// TODO: windows os is case-insensitive for env variables,
	// so it may requre to load all vars and lookup for env var manually

	val, ok := os.LookupEnv(in)
	if !ok {
		return "", fmt.Errorf("%s: %w", in, ErrEnvVariableNotProvided)
	}

	return val, nil
}
