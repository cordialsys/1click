package secret_test

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/cordialsys/panel/pkg/secret"
	vault "github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/require"
)

var GetSecret = secret.GetSecret

func TestGetSecretEnv(t *testing.T) {
	os.Setenv("XCTEST", "mysecret")
	secret, err := GetSecret("env:XCTEST")
	os.Unsetenv("XCTEST")
	require.Equal(t, "mysecret", secret)
	require.NoError(t, err)
}

func TestGetSecretFile(t *testing.T) {
	tmp, _ := os.CreateTemp("", "test-secret")
	_, _ = tmp.WriteString("Apache License")
	_ = tmp.Close()

	secret, err := GetSecret("file:" + tmp.Name())
	require.Contains(t, secret, "Apache License")
	require.NoError(t, err)
}

func TestGetSecretFileHomeErrFileNotFound(t *testing.T) {
	secret, err := GetSecret("file:~/config-in-home")
	require.Equal(t, "", secret)
	require.Error(t, err)
}

func TestGetSecretErrFileNotFound(t *testing.T) {
	secret, err := GetSecret("file:../LICENSEinvalid")
	require.Equal(t, "", secret)
	require.Error(t, err)
	// Should be detectable as not existing
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestGetSecretErrNoColon(t *testing.T) {
	secret, err := GetSecret("invalid")
	require.Equal(t, "", secret)
	require.Error(t, err)
}

func TestGetSecretErrInvalidType(t *testing.T) {
	secret, err := GetSecret("invalid:value")
	require.Equal(t, "", secret)
	require.Error(t, err)
}

type MockedVaultLoaded struct {
	data map[string]interface{}
}

var _ secret.VaultLoader = &MockedVaultLoaded{}

func (l *MockedVaultLoaded) LoadSecretData(path string) (*vault.Secret, error) {
	data, ok := l.data[path]
	if !ok {
		return &vault.Secret{}, errors.New("path not found")
	}
	return &vault.Secret{
		Data: data.(map[string]interface{}),
	}, nil
}

func TestGetSecretVault(t *testing.T) {
	secret.NewVaultClient = func(cfg *vault.Config) (secret.VaultLoader, error) {
		vaultRes := `{
			"path1/to": {
				"data": {
					"secret": "mysecret"
				}
			},
			"path2/to": {
				"data": {
					"secret2": "mysecret2"
				}
			}
		}`
		data := make(map[string]interface{})
		err := json.Unmarshal([]byte(vaultRes), &data)
		require.NoError(t, err)

		return &MockedVaultLoaded{
			data: data,
		}, nil
	}

	_, err := GetSecret("vault:wrong_args")
	require.ErrorContains(t, err, "vault secret has 2 comma separated arguments")
	_, err = GetSecret("vault:wrong_args,aaa,bbb")
	require.ErrorContains(t, err, "vault secret has 2 comma separated arguments")

	_, err = GetSecret("vault:url,aaa")
	require.ErrorContains(t, err, "malformed vault secret")

	_, err = GetSecret("vault:url,aaa/secret")
	require.EqualError(t, err, "path not found")

	secret, err := GetSecret("vault:https://example.com,path1/to/secret")
	require.NoError(t, err)
	require.Equal(t, "mysecret", secret)

	secret, err = GetSecret("vault:https://example.com,path2/to/secret2")
	require.NoError(t, err)
	require.Equal(t, "mysecret2", secret)

	secret, err = GetSecret("vault:https://example.com,path2/to/secret_none")
	require.NoError(t, err)
	require.Equal(t, "", secret)
}

func TestGetSecretFileTrimmed(t *testing.T) {
	dir := os.TempDir()
	file, err := os.CreateTemp(dir, "config-test")
	require.NoError(t, err)
	defer file.Close()
	file.Write([]byte("MYSECRET"))
	file.Sync()

	sec, err := GetSecret("file:" + file.Name())
	require.NoError(t, err)
	require.Equal(t, "MYSECRET", sec)

	file2, err := os.CreateTemp(dir, "config-test")
	require.NoError(t, err)
	defer file2.Close()
	// add whitespace
	file2.Write([]byte(" MY SECRET \n"))
	file2.Sync()

	sec, err = GetSecret("file:" + file2.Name())
	require.NoError(t, err)
	require.Equal(t, "MY SECRET", sec)
}
