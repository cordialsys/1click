package s3client

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/cordialsys/panel/pkg/secret"
	"github.com/sirupsen/logrus"
)

type treasuryS3Transport struct {
	treasury string
	node     string
	apiKey   string
}

func (t *treasuryS3Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.treasury != "" {
		req.Header.Set("treasury", t.treasury)
	}
	if t.node != "" {
		req.Header.Set("node", t.node)
	}
	if t.apiKey != "" {
		asB64 := ""
		if strings.Contains(t.apiKey, ":") {
			asB64 = base64.StdEncoding.EncodeToString([]byte(t.apiKey))
		}
		req.Header.Set("authorization", fmt.Sprintf("Basic %s", asB64))
	}
	return http.DefaultTransport.RoundTrip(req)
}

type BackupS3Client struct {
	svc        *s3.Client
	opts       BackupS3ClientOptions
	bucketName *string
}

type BackupS3ClientOptions struct {
	Endpoint string
	Treasury string
	Node     string

	// Set only if you want to override the bucket name from the treasury ID
	Bucket string

	// Optional:
	Region  string
	ApiKey  secret.Secret
	S3Token secret.Secret
	Debug   bool
}

func (opts BackupS3ClientOptions) BucketName() string {
	if opts.Bucket != "" {
		return strings.TrimSpace(opts.Bucket)
	}
	bucket := strings.TrimSpace(opts.Treasury)
	// bucket names must be lowercase
	bucket = strings.ToLower(bucket)
	bucket = strings.TrimPrefix(bucket, "treasuries/")

	return bucket
}

func NewBackupS3Client(opts BackupS3ClientOptions) (*BackupS3Client, error) {
	var err error
	region := "unknown"
	if opts.Region != "" {
		region = opts.Region
	}

	// Start with a basic config
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	var creds aws.CredentialsProvider = aws.AnonymousCredentials{}

	// Check for environment credentials first
	envCreds := cfg.Credentials
	if _, err := envCreds.Retrieve(context.TODO()); err == nil {
		creds = envCreds
	}

	if opts.S3Token != "" {
		s3Token, err := opts.S3Token.Load()
		if err != nil {
			logrus.WithError(err).Error("error loading s3 token")
			return nil, err
		}
		if s3Token == "" {
			logrus.Warn("s3 token is empty")
		} else {
			parts := strings.Split(s3Token, ":")
			if len(parts) == 0 {
				logrus.Warn("unable to split s3 token by ':'")
			}
			id := parts[0]
			secret := ""
			token := ""
			if len(parts) > 1 {
				secret = parts[1]
			}
			if len(parts) > 2 {
				token = parts[2]
			}
			creds = credentials.NewStaticCredentialsProvider(id, secret, token)
		}
	}

	var apiKey string
	if opts.ApiKey != "" {
		// Check if we're using non-anonymous credentials
		_, err := creds.Retrieve(context.TODO())
		isAnonymous := err != nil

		if !isAnonymous {
			logrus.Warn("ignoring TREASURY_API_KEY since s3-token is set or inferred from environment")
		} else {
			apiKey, err = opts.ApiKey.Load()
			if err != nil {
				logrus.WithError(err).Error("error loading treasury api key")
				return nil, err
			}
			if apiKey == "" {
				logrus.Warn("treasury api key is empty")
			}
		}
	}

	// Configure the S3 client
	s3Options := []func(*s3.Options){
		func(o *s3.Options) {
			o.Credentials = creds
			o.Region = region
			if opts.Endpoint != "" {
				o.BaseEndpoint = aws.String(opts.Endpoint)
				o.UsePathStyle = true
			}
			o.HTTPClient = &http.Client{
				Transport: &treasuryS3Transport{
					treasury: opts.Treasury,
					node:     opts.Node,
					apiKey:   apiKey,
				},
			}
		},
	}

	svc := s3.NewFromConfig(cfg, s3Options...)

	return &BackupS3Client{
		svc:        svc,
		opts:       opts,
		bucketName: aws.String(opts.BucketName()),
	}, nil
}

func (c *BackupS3Client) GetNodeId() string {
	return c.opts.Node
}

type ListObjectsOptions struct {
	Prefix string
	Marker string
}

func (c *BackupS3Client) ListObjects(ctx context.Context, opts ListObjectsOptions) (*s3.ListObjectsOutput, error) {
	var prefix *string
	if opts.Prefix != "" {
		prefix = aws.String(opts.Prefix)
	}
	var marker *string
	if opts.Marker != "" {
		marker = aws.String(opts.Marker)
	}
	return c.svc.ListObjects(ctx, &s3.ListObjectsInput{
		// Treasury is always the bucket name
		Bucket: c.bucketName,
		Prefix: prefix,
		Marker: marker,
	})
}

func (c *BackupS3Client) HeadObject(ctx context.Context, key string) (*s3.HeadObjectOutput, error) {
	return c.svc.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: c.bucketName,
		Key:    aws.String(key),
	})
}

func (c *BackupS3Client) GetObject(ctx context.Context, key string) (*s3.GetObjectOutput, error) {
	return c.svc.GetObject(ctx, &s3.GetObjectInput{
		Bucket: c.bucketName,
		Key:    aws.String(key),
	})
}

func trimPathPrefix(path string) string {
	path = strings.TrimPrefix(path, "backups/")
	path = strings.TrimPrefix(path, "snapshots/")
	path = strings.TrimPrefix(path, "keys/")
	return path
}

func (c *BackupS3Client) PutKey(ctx context.Context, keyPath string, body io.ReadSeeker) (*s3.PutObjectOutput, error) {
	keyPath = trimPathPrefix(keyPath)
	return c.svc.PutObject(ctx, &s3.PutObjectInput{
		Bucket: c.bucketName,
		Key:    aws.String(filepath.Join("nodes", c.opts.Node, "keys", keyPath)),
		Body:   body,
	})
}

func (c *BackupS3Client) PutSnapshot(ctx context.Context, snapshotPath string, body io.ReadSeeker) (*s3.PutObjectOutput, error) {
	snapshotPath = trimPathPrefix(snapshotPath)
	return c.svc.PutObject(ctx, &s3.PutObjectInput{
		Bucket: c.bucketName,
		Key:    aws.String(filepath.Join("nodes", c.opts.Node, "snapshots", snapshotPath)),
		Body:   body,
	})
}

func (s3Client *BackupS3Client) CreateBucketIfErrIsMissing(ctx context.Context, err error) bool {
	// try to create bucket if it doesn't exist
	var noSuchBucket *types.NoSuchBucket
	var invalidBucketName *types.InvalidObjectState

	if errors.As(err, &noSuchBucket) || errors.As(err, &invalidBucketName) {
		log := logrus.WithField("name", s3Client.opts.BucketName())
		log.Info("bucket does not exist, will try to create it")
		_, err = s3Client.svc.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(s3Client.opts.BucketName()),
		})
		if err != nil {
			var bucketAlreadyExists *types.BucketAlreadyExists
			var bucketAlreadyOwnedByYou *types.BucketAlreadyOwnedByYou

			if errors.As(err, &bucketAlreadyExists) || errors.As(err, &bucketAlreadyOwnedByYou) {
				log.Info("bucket is already created")
			} else {
				log.WithError(err).Error("error creating bucket")
			}
		} else {
			return true
		}
	}
	return false
}

// Return an iterator for the files under the given prefix.
// Note that files will be returned in lexicographic order, and without the prefix.
func (s3Client *BackupS3Client) IterateFiles(ctx context.Context, prefix string) (<-chan string, error) {
	files := make(chan string)
	// err := make(chan error)
	go func() {
		defer close(files)
		// defer close(err)
		var nextMarker string
		for {
			output, err := s3Client.ListObjects(ctx, ListObjectsOptions{
				Prefix: prefix,
				Marker: nextMarker,
			})
			if err != nil {
				logrus.WithError(err).Error("error listing objects")
				return
			}
			for _, obj := range output.Contents {
				path := strings.TrimPrefix(*obj.Key, prefix)
				path = strings.TrimPrefix(path, "/")
				files <- path
			}
			if len(output.Contents) == 0 {
				break
			}
			if output.NextMarker == nil || *output.NextMarker == "" {
				break
			}
			nextMarker = *output.NextMarker
		}
	}()

	return files, nil
}
