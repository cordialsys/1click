package endpoints

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"filippo.io/age"
	"github.com/cordialsys/panel/pkg/bak"
	"github.com/cordialsys/panel/pkg/resource"
	"github.com/cordialsys/panel/pkg/s3client"
	"github.com/cordialsys/panel/pkg/snapshot"
	"github.com/cordialsys/panel/server/servererrors"
	"github.com/gofiber/fiber/v2"
)

const DefaultBackupUrl = "https://backup.cordialapis.com"
const ENV_SIGNER_BAK_PHRASE = "SIGNER_BAK_PHRASE"

const ENV_SIGNER_NEW_EAR_PHRASE = "SIGNER_NEW_EAR_PHRASE"
const ENV_SIGNER_EAR_PHRASE = "SIGNER_EAR_PHRASE"

func NodeSpecificSignerBakPhrase(nodeId int) string {
	return fmt.Sprintf("SIGNER_%d_BAK_PHRASE", nodeId)
}

func DecodeEncryptedSecretPhrase(identity *age.X25519Identity, encryptedSecretPhrase string) (string, error) {
	encryptedBytes, err := base64.StdEncoding.DecodeString(encryptedSecretPhrase)
	if err != nil {
		return "", servererrors.InternalErrorf("failed to decode encrypted secret phrase as base64: %v", err)
	}
	reader, err := age.Decrypt(bytes.NewReader(encryptedBytes), identity)
	if err != nil {
		return "", servererrors.InternalErrorf("failed to decrypt encrypted secret phrase: %v", err)
	}
	mnemonicBz, err := io.ReadAll(reader)
	if err != nil {
		return "", servererrors.InternalErrorf("failed to read secret phrase: %v", err)
	}
	mnemonic := string(mnemonicBz)
	mnemonic = FormatMnemonic(mnemonic)
	if len(mnemonic) == 0 {
		return "", servererrors.InternalErrorf("secret phrase is too short")
	}
	return mnemonic, nil
}

func FormatMnemonic(mnemonic string) string {
	mnemonic = strings.TrimSpace(mnemonic)
	parts := strings.Split(mnemonic, " ")
	mnemonic = ""
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		mnemonic += part + " "
	}
	mnemonic = strings.TrimSpace(mnemonic)
	return mnemonic
}

func BakShortId(bak string) string {
	// use first 32 in backups/s3
	if len(bak) > 32 {
		return bak[:32]
	}
	return bak
}

func (endpoints *Endpoints) ListObjects(c *fiber.Ctx) error {
	if endpoints.panel.ApiKey == "" {
		return servererrors.FailedPreconditionf("not activated")
	}
	ctx := c.Context()
	prefix := c.Query("prefix")
	marker := c.Query("marker")

	resp, err := endpoints.s3Client.ListObjects(ctx, s3client.ListObjectsOptions{
		Prefix: prefix,
		Marker: marker,
	})
	if err != nil {
		return servererrors.InternalErrorf("failed to list objects: %v", err)
	}

	return c.JSON(resp)
}

func (endpoints *Endpoints) DownloadObject(c *fiber.Ctx) error {
	if endpoints.panel.ApiKey == "" {
		return servererrors.FailedPreconditionf("not activated")
	}
	ctx := c.Context()
	fileKey := c.Query("key")
	if fileKey == "" {
		return servererrors.BadRequestf("missing key query param")
	}

	resp, err := endpoints.s3Client.GetObject(ctx, fileKey)
	if err != nil {
		return servererrors.InternalErrorf("failed to get object: %v", err)
	}

	c.Response().Header.Set(
		"Content-Disposition",
		fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(fileKey)),
	)

	if resp.ContentLength != nil {
		return c.SendStream(resp.Body, int(*resp.ContentLength))
	} else {
		return c.SendStream(resp.Body)
	}
}

func (endpoints *Endpoints) TakeSnapshot(c *fiber.Ctx) error {
	if endpoints.panel.ApiKey == "" {
		return servererrors.FailedPreconditionf("not activated")
	}
	snapshotId, err := url.PathUnescape(c.Params("id"))
	if err != nil {
		return servererrors.BadRequestf("invalid snapshot id: %v", err)
	}
	if snapshotId == "" {
		snapshotId = fmt.Sprintf("manual-snapshot-%d", time.Now().Unix())
	}
	snapshotId = resource.NormalizeId(snapshotId)
	snapshotId = strings.Trim(snapshotId, "-")

	_, download := c.Queries()["download"]
	bak := c.Query("bak")
	if download {
		if bak == "" {
			return servererrors.BadRequestf("must select a backup key to download, or you may download separately")
		}
	}
	if bak != "" {
		matches := false
		for _, b := range endpoints.panel.Baks {
			if b.Key == bak {
				matches = true
				break
			}
		}
		if !matches {
			return servererrors.BadRequestf("bak does not match any existing backup key")
		}
	}

	args := []string{
		"backup",
		"snapshot",
		"--output-dir", endpoints.panel.BackupDir,
	}
	if bak != "" {
		args = append(args, "--bak", bak)
	}
	if snapshotId != "" {
		args = append(args, "--id", snapshotId)
	}

	err = endpoints.execCordWithHome(args, IncludeEar)
	if err != nil {
		return servererrors.InternalErrorf("failed to take snapshot: %v", err)
	}

	if download {
		type snapshotInfo struct {
			Path string
			Info os.FileInfo
		}
		mostRecentSnapshot := snapshotInfo{}
		err = filepath.WalkDir(filepath.Join(endpoints.panel.BackupDir, "snapshots"), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".tar") {
				return nil
			}
			if !strings.Contains(path, "age1") {
				return nil
			}
			info, err := os.Stat(path)
			if err != nil {
				slog.Error("failed to stat snapshot", "path", path, "error", err)
				return nil
			}
			fmt.Println(path)
			if mostRecentSnapshot.Path == "" {
				mostRecentSnapshot = snapshotInfo{
					Path: path,
					Info: info,
				}
				return nil
			}
			if info.ModTime().Unix() > mostRecentSnapshot.Info.ModTime().Unix() {
				mostRecentSnapshot = snapshotInfo{
					Path: path,
					Info: info,
				}
			}

			return nil
		})
		if err != nil {
			return servererrors.InternalErrorf("failed to walk snapshots: %v", err)
		}
		if mostRecentSnapshot.Path == "" {
			return servererrors.InternalErrorf("unable to locate snapshot, please check the uploaded backups")
		}

		return c.SendFile(mostRecentSnapshot.Path)
	}

	return c.JSON(nil)
}

func (endpoints *Endpoints) UploadSnapshot(c *fiber.Ctx) error {
	if endpoints.panel.ApiKey == "" {
		return servererrors.FailedPreconditionf("not activated")
	}
	snapshotId, err := url.PathUnescape(c.Params("id"))
	if err != nil {
		return servererrors.BadRequestf("invalid snapshot id: %v", err)
	}
	snapshotId = resource.NormalizeId(snapshotId)
	snapshotId = strings.Trim(snapshotId, "-")
	if snapshotId == "" {
		return servererrors.BadRequestf("snapshot id is required")
	}
	tmpdir, err := os.MkdirTemp("", "snapshot-")
	if err != nil {
		return servererrors.InternalErrorf("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tmpdir)

	tmpSnapshotPath := filepath.Join(tmpdir, "tmp.tar")
	tmpSnapshotFile, err := os.Create(tmpSnapshotPath)
	if err != nil {
		return servererrors.InternalErrorf("failed to create temporary snapshot file: %v", err)
	}
	defer tmpSnapshotFile.Close()
	bodyStream := c.Request().BodyStream()
	if bodyStream == nil {
		bodyStream = bytes.NewReader(c.Request().Body())
	}

	_, err = io.Copy(tmpSnapshotFile, bodyStream)
	if err != nil {
		return servererrors.InternalErrorf("failed to copy snapshot to temporary file: %v", err)
	}
	tmpSnapshotFile.Close()

	err = exec.Command("tar", "-xvf", tmpSnapshotPath, "-C", tmpdir).Run()
	if err != nil {
		return servererrors.BadRequestf("failed to parse snapshot: %v", err)
	}

	infoBz, err := os.ReadFile(filepath.Join(tmpdir, "info.json"))
	if err != nil {
		return servererrors.BadRequestf("failed to read info in snapshot: %v", err)
	}

	var info snapshot.Info
	if err := json.Unmarshal(infoBz, &info); err != nil {
		return servererrors.BadRequestf("failed to parse snapshot info: %v", err)
	}

	if info.Participant != snapshot.Int(endpoints.panel.NodeId) {
		return servererrors.BadRequestf("snapshot is for node %d, but this node is %d", info.Participant, endpoints.panel.NodeId)
	}
	if info.Bak == "" {
		return servererrors.BadRequestf("snapshot does not have an associated bak")
	}

	ageShortId := BakShortId(info.Bak)

	relativePath := fmt.Sprintf("%s/%s.tar", ageShortId, snapshotId)

	tmpSnapshotFile, err = os.Open(tmpSnapshotPath)
	if err != nil {
		return servererrors.InternalErrorf("failed to open temporary snapshot file: %v", err)
	}
	defer tmpSnapshotFile.Close()

	_, err = endpoints.s3Client.PutSnapshot(c.Context(), relativePath, tmpSnapshotFile)
	if err != nil {
		return servererrors.InternalErrorf("failed to upload snapshot: %v", err)
	}

	return c.JSON(nil)
}

type RestoreSnapshotRequest struct {
	// Age encrypted mnemonic phrase
	EncryptedSecretPhrase string `json:"encrypted_secret_phrase"`

	// S3 File key
	S3Key string `json:"s3_key"`
}

// Restore from snapshot
// - Stop treasury
// - Download the snapshot
// - Restore the snapshot
func (endpoints *Endpoints) RestoreFromSnapshot(c *fiber.Ctx) error {
	if endpoints.panel.ApiKey == "" {
		return servererrors.FailedPreconditionf("not activated")
	}
	ctx := c.Context()

	req := RestoreSnapshotRequest{}
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return servererrors.BadRequestf("failed to parse request: %v", err)
	}
	if req.EncryptedSecretPhrase == "" {
		return servererrors.BadRequestf("missing encrypted_mnemonic_phrase")
	}
	if req.S3Key == "" {
		return servererrors.BadRequestf("missing s3_key")
	}
	if !strings.Contains(req.S3Key, "/snapshots/") {
		return servererrors.BadRequestf("file does not appear to be a snapshot")
	}

	mnemonic, err := DecodeEncryptedSecretPhrase(endpoints.identity, req.EncryptedSecretPhrase)
	if err != nil {
		return err
	}

	tmpdir, err := os.MkdirTemp("", "snapshot-")
	if err != nil {
		return servererrors.InternalErrorf("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tmpdir)

	baseName := filepath.Base(req.S3Key)
	snapshotPath := filepath.Join(tmpdir, baseName)

	outputFile, err := os.Create(snapshotPath)
	if err != nil {
		return servererrors.InternalErrorf("failed to create output file: %v", err)
	}
	defer outputFile.Close()

	slog.Info("getting object", "s3_key", req.S3Key)

	object, err := endpoints.s3Client.GetObject(ctx, req.S3Key)
	if err != nil {
		return servererrors.InternalErrorf("failed to get object: %v", err)
	}
	defer object.Body.Close()

	_, err = io.Copy(outputFile, object.Body)
	if err != nil {
		return servererrors.InternalErrorf("failed to download object to file: %v", err)
	}
	_ = object.Body.Close()
	_ = outputFile.Close()

	// stop treasury
	_, err = stopSystemdServiceAndWait(ctx, ServiceTreasury)
	if err != nil {
		return err
	}

	err = endpoints.execCordWithHome([]string{
		"backup",
		"restore",
		"--snapshot", snapshotPath,
	}, IncludeEar, fmt.Sprintf("%s=%s", ENV_SIGNER_BAK_PHRASE, mnemonic))
	if err != nil {
		return servererrors.InternalErrorf("failed to apply snapshot: %v", err)
	}

	// // start treasury
	// _, err = updateSystemdService(ctx, ServiceTreasury, "start")
	// if err != nil {
	// 	return servererrors.InternalErrorf("failed to start treasury: %v", err)
	// }

	return c.JSON(nil)
}

type RestoreMissingKeysRequest struct {
	// Age encrypted mnemonic phrase
	EncryptedSecretPhrase string `json:"encrypted_secret_phrase"`
}

type RestoreMissingKeysResponse struct {
	// Keys currently active in signer
	ActiveKeys int `json:"active_keys"`
	// Keys currently backed up (by the input bak) in s3
	BackedUpKeys int `json:"backed_up_keys"`
	// Keys imported
	ImportedKeys int `json:"imported_keys"`
}

// Restore Missing Keys is the for case of a interruption where keys were not included in the latest snapshot.
// - Stop treasury
// - Scan all key names using `signer list-keys`
// - Scan all of keys in s3 bucket encrypted with the input bak
// - Download the keys that are missing and import them
func (endpoints *Endpoints) RestoreMissingKeys(c *fiber.Ctx) error {
	ctx := c.Context()
	var err error
	req := RestoreMissingKeysRequest{}
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return servererrors.BadRequestf("failed to parse request: %v", err)
	}
	if req.EncryptedSecretPhrase == "" {
		return servererrors.BadRequestf("missing encrypted_mnemonic_phrase")
	}

	// Validate the mnemonic phrase decrypts
	mnemonic, err := DecodeEncryptedSecretPhrase(endpoints.identity, req.EncryptedSecretPhrase)
	if err != nil {
		return err
	}

	bakKey, err := bak.NewEncryptionKey(strings.Split(mnemonic, " "))
	if err != nil {
		return servererrors.InternalErrorf("failed to derive decryption key: %v", err)
	}
	bakRecipient := bakKey.Recipient()
	bak := bakRecipient.String()

	// stop treasury
	_, err = stopSystemdServiceAndWait(ctx, ServiceTreasury)
	if err != nil {
		return err
	}

	signerBin := filepath.Join(endpoints.panel.BinaryDir, "signer")
	execList := []string{
		"list-keys",
		"--db", endpoints.panel.TreasuryHome.SignerDb(),
	}
	execCmd := exec.Command(signerBin, execList...)
	err = endpoints.attachEarSecretToCmd(execCmd)
	if err != nil {
		return err
	}
	signerOut, err := execCmd.StdoutPipe()
	if err != nil {
		return servererrors.InternalErrorf("failed to create stdout pipe: %v", err)
	}
	err = execCmd.Start()
	if err != nil {
		return servererrors.InternalErrorf("failed to start signer: %v", err)
	}

	// Set of key names
	existingKeys := map[resource.KeyName]struct{}{}
	backedUpKeys := 0
	activeKeys := 0
	importedKeys := 0

	// read key infos from signer output
	scanner := bufio.NewScanner(signerOut)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
		var key resource.Key
		if err := json.Unmarshal([]byte(line), &key); err != nil {
			slog.Warn("failed to unmarshal key", "line", line, "error", err)
			continue
		}
		existingKeys[key.Name] = struct{}{}
		activeKeys++
	}
	if err := scanner.Err(); err != nil {
		return servererrors.InternalErrorf("failed to read signer output: %v", err)
	}
	err = execCmd.Wait()
	if err != nil {
		return servererrors.InternalErrorf("failed to wait for signer: %v", err)
	}

	// now scan all of the keys in the s3 bucket
	nodeId := fmt.Sprintf("%d", endpoints.panel.NodeId)
	prefix := fmt.Sprintf("nodes/%s/keys/nodes/%s/%s", nodeId, nodeId, BakShortId(bak))
	fmt.Println("prefix", prefix)

	s3FileIter, err := endpoints.s3Client.IterateFiles(ctx, prefix)
	if err != nil {
		return servererrors.InternalErrorf("failed to iterate s3 files: %v", err)
	}
	tmpdir, err := os.MkdirTemp("", "missing-keys-")
	if err != nil {
		return servererrors.InternalErrorf("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tmpdir)

	type missingKey struct {
		name    resource.KeyName
		fileKey string
	}

	missingKeys := make(chan missingKey, 50)
	completedDownload := make(chan bool, 1)
	completedScan := make(chan bool, 1)
	ctxDownload := context.Background()
	go func() {
		defer close(missingKeys)
		for {
			select {
			case missingKey := <-missingKeys:
				object, err := endpoints.s3Client.GetObject(ctxDownload, missingKey.fileKey)
				if err != nil {
					slog.Warn("failed to get object", "file_key", missingKey.fileKey, "error", err)
					continue
				}
				defer object.Body.Close()
				keyFilePath := filepath.Join(tmpdir, missingKey.name.Id()+".json")
				f, err := os.Create(keyFilePath)
				if err != nil {
					slog.Error("failed to create file", "file_key", missingKey.name.Id(), "error", err)
					continue
				}
				defer f.Close()
				_, err = io.Copy(f, object.Body)
				if err != nil {
					slog.Warn("failed to copy object to file", "file_key", missingKey.name.Id(), "error", err)
					continue
				}
				importedKeys++
				slog.Info("downloaded key", "key_id", missingKey.name.Id(), "path", keyFilePath)

			case <-completedScan:
				if len(missingKeys) == 0 {
					completedDownload <- true
					break
				} else {
					completedScan <- true
				}
			}
		}
	}()

	for s3File := range s3FileIter {
		backedUpKeys++
		if !strings.Contains(s3File, "@") {
			continue
		}
		keyId := strings.Split(s3File, "@")[0]
		keyName := resource.NewKeyName(keyId)
		_, exists := existingKeys[keyName]
		if !exists {
			missingKeys <- missingKey{
				name:    keyName,
				fileKey: filepath.Join(prefix, s3File),
			}
		}
		fmt.Printf("%s exists=%t\n", s3File, exists)
	}
	completedScan <- true
	<-completedDownload

	// import all of the keys
	if importedKeys > 0 {
		defer func() {
			// potentially restore ownership of signer.db
			execCmd := exec.Command("chown", "-R", endpoints.panel.TreasuryUser, endpoints.panel.TreasuryHome.SignerDb())
			bz, err := execCmd.CombinedOutput()
			if err != nil {
				slog.Error("failed to change ownership of signer.db", "error", err, "output", string(bz))
			}
		}()
		execList := []string{
			"backup",
			"import",
			"--db", endpoints.panel.TreasuryHome.SignerDb(),
			"--import-dir", tmpdir,
		}
		execCmd := exec.Command(signerBin, execList...)
		execCmd.Env = append(
			os.Environ(),
			fmt.Sprintf("%s=%s", ENV_SIGNER_BAK_PHRASE, mnemonic),
			fmt.Sprintf("%s=%s", NodeSpecificSignerBakPhrase(int(endpoints.panel.NodeId)), mnemonic),
		)
		err = endpoints.attachEarSecretToCmd(execCmd)
		if err != nil {
			return err
		}
		outputBz, err := execCmd.CombinedOutput()
		if err != nil {
			return servererrors.InternalErrorf("failed to run `%s`: %v", execCmd.String(), string(outputBz))
		}
		slog.Info("imported keys", "output", string(outputBz))
	}

	return c.JSON(RestoreMissingKeysResponse{
		ActiveKeys:   activeKeys,
		BackedUpKeys: backedUpKeys,
		ImportedKeys: importedKeys,
	})
}
