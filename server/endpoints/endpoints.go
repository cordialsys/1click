package endpoints

import (
	"fmt"
	"log/slog"

	"filippo.io/age"
	"github.com/cordialsys/panel/pkg/admin"
	"github.com/cordialsys/panel/pkg/s3client"
	"github.com/cordialsys/panel/pkg/secret"
	"github.com/cordialsys/panel/server/panel"
	"github.com/cordialsys/panel/server/servererrors"
)

type Endpoints struct {
	panel    *panel.Panel
	identity *age.X25519Identity
	s3Client *s3client.BackupS3Client
}

func NewEndpoints(panel *panel.Panel, identity *age.X25519Identity) *Endpoints {
	if panel.ApiKey != "" {
		valid, err := validateAPIKey(panel.ApiKey)
		if err != nil {
			slog.Error("invalid API key", "error", err)
		} else {
			panel.ApiKey = valid
		}
	}
	cli, err := s3client.NewBackupS3Client(s3client.BackupS3ClientOptions{
		Endpoint: DefaultBackupUrl,
		Treasury: panel.TreasuryId,
		Node:     fmt.Sprint(panel.NodeId),
		ApiKey:   secret.NewRawSecret(panel.ApiKey),

		// TODO
		S3Token: "",
		Bucket:  "",
		Region:  "",
		Debug:   false,
	})
	if err != nil {
		panic(err)
	}
	return &Endpoints{
		panel,
		identity,
		cli,
	}
}

func (endpoints *Endpoints) AdminClient() (*admin.Client, error) {
	if !endpoints.panel.HasNodeSet() {
		return nil, servererrors.BadRequestf("the API key has not yet been activated")
	}
	client := admin.NewClient(endpoints.panel.ApiKey)
	return client, nil
}
