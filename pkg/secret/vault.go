package secret

import (
	vault "github.com/hashicorp/vault/api"
)

func newVaultClient(cfg *vault.Config) (VaultLoader, error) {
	cli, err := vault.NewClient(cfg)
	if err != nil {
		return &DefaultVaultLoader{}, err
	}
	return &DefaultVaultLoader{Client: cli}, nil
}

var NewVaultClient = newVaultClient

type DefaultVaultLoader struct {
	*vault.Client
}

var _ VaultLoader = &DefaultVaultLoader{}

func (v *DefaultVaultLoader) LoadSecretData(vaultPath string) (*vault.Secret, error) {
	secret, err := v.Logical().Read(vaultPath)
	if err != nil || secret == nil { // yes, secret can be nil
		return &vault.Secret{}, err
	}
	return secret, nil
}

type VaultLoader interface {
	LoadSecretData(path string) (*vault.Secret, error)
}
