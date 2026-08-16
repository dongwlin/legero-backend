//go:build wireinject

package app

import "github.com/google/wire"

// InitializeApplication builds the all-in-one HTTP application.
func InitializeApplication() (*Application, func(), error) {
	wire.Build(ServerProviderSet)
	return nil, nil, nil
}

// InitializeUserCreator builds the create-user CLI bootstrap.
func InitializeUserCreator() (*UserCreator, func(), error) {
	wire.Build(UserCreatorProviderSet)
	return nil, nil, nil
}
