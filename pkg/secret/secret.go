package secret

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	awssecretmanager "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	vault "github.com/hashicorp/vault/api"
	"google.golang.org/api/iterator"
)

type Secret string

func (s Secret) Load() (string, error) {
	return GetSecret(string(s))
}

func (s Secret) IsType(t SecretType) bool {
	return strings.HasPrefix(string(s), string(t)+":")
}
func (s Secret) Type() (SecretType, bool) {
	splits := strings.Split(string(s), ":")
	if len(splits) == 1 {
		return "", false
	}
	return SecretType(splits[0]), SecretType(splits[0]).Name() != ""
}

type SecretType string

const (
	Env              SecretType = "env"
	Vault            SecretType = "vault"
	File             SecretType = "file"
	Keyring          SecretType = "keyring"
	GcpSecretManager SecretType = "gcp"
	AwsSecretManager SecretType = "aws"
	Raw              SecretType = "raw"
)

func (t SecretType) Name() string {
	switch t {
	case Env:
		return "Environment variable"
	case Vault:
		return "Hashicorp Vault"
	case File:
		return "File"
	case GcpSecretManager, "gsm":
		return "GCP Secret Manager"
	case AwsSecretManager:
		return "AWS Secret Manager"
	case Keyring:
		return "Treasury Keyring"
	case Raw:
		return "Raw"
	}
	return ""
}
func (t SecretType) Usage() string {
	switch t {
	case Env:
		return "<name>"
	case Vault:
		return "<server-url>,<path>"
	case File:
		return "<path>"
	case GcpSecretManager, "gsm":
		return "<project>,<name>[,version]"
	case AwsSecretManager:
		return "<name[:key]>[,region][,version]"
	case Keyring:
		return "<id-or-path>"
	}
	return ""
}

var Types = []SecretType{
	Env, Vault, File, GcpSecretManager, AwsSecretManager, Keyring,
}

func replaceTilda(path string) string {
	if len(path) > 1 && path[0] == '~' {
		userHome, _ := os.UserHomeDir()
		path = strings.Replace(path, "~", userHome, 1)
	}
	return path
}

// GetSecret returns a secret, e.g. from env variable. Extend as needed.
func GetSecret(uri string) (secret string, err error) {
	value := uri

	splits := strings.Split(value, ":")
	if len(splits) < 2 {
		return "", fmt.Errorf(
			"could not load secret, missing prefix; secret should be in '<prefix>:<path>' format, where prefix is one of %v",
			Types,
		)
	}

	secretType := strings.ToLower(splits[0])
	args := strings.Split(strings.Join(splits[1:], ":"), ",")
	switch SecretType(secretType) {
	case Env:
		path := args[0]
		return strings.TrimSpace(os.Getenv(path)), nil
	case File:
		path := replaceTilda(args[0])
		_, err := os.Stat(path)
		if err != nil {
			return "", err
		}
		file, err := os.Open(path)
		defer file.Close()
		if err != nil {
			return "", err
		}
		result, err := io.ReadAll(file)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(result)), nil
	case Vault:
		if len(args) != 2 {
			return "", errors.New("vault secret has 2 comma separated arguments (url,path)")
		}
		vaultUrl := args[0]
		vaultFullPath := args[1]

		cfg := &vault.Config{Address: vaultUrl}
		// just check the error
		_, err := vault.NewClient(cfg)
		if err != nil {
			return "", err
		}
		client, err := NewVaultClient(cfg)
		if err != nil {
			return "", err
		}

		idx := strings.LastIndex(vaultFullPath, "/")
		if idx == -1 || idx == len(vaultFullPath) { // idx shouldn't be the last char
			return "", errors.New("malformed vault secret in config file")
		}
		vaultKey := vaultFullPath[idx+1:]
		vaultPath := vaultFullPath[:idx]

		secret, err := client.LoadSecretData(vaultPath)
		if err != nil {
			return "", err
		}
		data, _ := secret.Data["data"].(map[string]interface{})
		result, _ := data[vaultKey].(string)
		return strings.TrimSpace(result), nil
	case GcpSecretManager, "gsm":
		// google secret manager
		if !slices.Contains([]int{2, 3}, len(args)) {
			return "", fmt.Errorf("%s secret has 2-3 comma separated arguments: %s", GcpSecretManager, GcpSecretManager.Usage())
		}
		project := args[0]
		if len(strings.Split(project, "/")) == 1 {
			// should have /projects/ prefix
			project = filepath.Join("projects", project)
		}
		name := args[1]
		version := "latest"
		if len(args) > 2 {
			version = args[2]
			version = strings.TrimPrefix(version, "versions/")
		}

		client, err := secretmanager.NewClient(context.Background())
		if err != nil {
			return "", err
		}

		it := client.ListSecrets(context.Background(), &secretmanagerpb.ListSecretsRequest{
			Parent: project,
			Filter: name,
		})
		for {
			resp, err := it.Next()
			if err == iterator.Done {
				break
			}

			if err != nil {
				return "", err
			}

			_, secretName := filepath.Split(resp.Name)
			if secretName == name {
				// access the specific version
				latest := filepath.Join(resp.Name, "versions/"+version)
				latestSecret, err := client.AccessSecretVersion(context.Background(), &secretmanagerpb.AccessSecretVersionRequest{
					Name: latest,
				})
				if err != nil {
					return "", err
				}
				return string(latestSecret.Payload.Data), nil
			}
		}
		return "", fmt.Errorf("could not find a gsm secret by name %s", name)

	case AwsSecretManager:
		if !slices.Contains([]int{1, 2, 3}, len(args)) {
			return "", fmt.Errorf("%s secret has 1-3 comma separated arguments: %s", AwsSecretManager, AwsSecretManager.Usage())
		}
		secretName := args[0]
		keyName := ""
		nameParts := strings.Split(secretName, ":")
		if len(nameParts) > 1 {
			secretName = nameParts[0]
			keyName = strings.Join(nameParts[1:], ":")
		}
		region := ""
		version := "AWSCURRENT"
		if len(args) > 1 {
			region = args[1]
		}
		if len(args) > 2 {
			version = args[2]
		}
		awsAwgs := []func(*config.LoadOptions) error{}
		if region != "" {
			awsAwgs = append(awsAwgs, config.WithRegion(region))
		}

		config, err := config.LoadDefaultConfig(context.Background(), awsAwgs...)
		if err != nil {
			return "", err
		}
		svc := awssecretmanager.NewFromConfig(config)
		input := &awssecretmanager.GetSecretValueInput{
			SecretId:     aws.String(secretName),
			VersionStage: aws.String(version),
		}
		result, err := svc.GetSecretValue(context.Background(), input)
		if err != nil {
			// https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_GetSecretValue.html
			return "", err
		}
		secretBz := *result.SecretString
		if keyName != "" {
			// Another layer of nesting to unwrap, this time JSON
			secretData := map[string]interface{}{}
			err = json.Unmarshal([]byte(secretBz), &secretData)
			if err != nil {
				// do not omit internal error to guard from leaking anything sensitive
				return "", fmt.Errorf("could not retrieve %s key because %s has invalid JSON", keyName, secretName)
			}
			secretValue, ok := secretData[keyName]
			if !ok {
				return "", fmt.Errorf("could not find %s key in %s JSON", keyName, secretName)
			}
			return fmt.Sprint(secretValue), nil
		}
		// return secretBz literally as the secret, no JSON nesting
		return secretBz, nil
	case Raw:
		return strings.Join(splits[1:], ":"), nil
	}
	return "", errors.New("invalid secret source for: ***")
}

func NewRawSecret(value string) Secret {
	return Secret(fmt.Sprintf("%s:%s", Raw, value))
}
