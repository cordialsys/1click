package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/cordialsys/panel/server/panel"
)

type Client struct {
	remote *url.URL
}

func NewClient(remote *url.URL) *Client {
	return &Client{remote}
}

type Error struct {
	Code    int
	Status  string
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Status, e.Message)
}

type Options struct {
	query         url.Values
	skipHttpError bool
}

func (c *Client) Do(method string, path string, inputMaybe any, outputMaybe any, optionsMaybe ...Options) error {
	// Build the full URL
	u := *c.remote
	u.Path = path

	options := Options{}
	if len(optionsMaybe) > 0 {
		options = optionsMaybe[0]
	}

	if options.query != nil {
		u.RawQuery = options.query.Encode()
	}

	log := slog.With("method", method, "url", u.String())
	log.Debug("request")

	// Create request with optional input body
	var err error
	req, err := http.NewRequest(method, u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Body = http.NoBody
	if inputMaybe != nil {
		body, err := json.Marshal(inputMaybe)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		if len(body) > 0 {
			req.Body = io.NopCloser(bytes.NewReader(body))
			req.ContentLength = int64(len(body))
			req.Header.Set("Content-Type", "application/json")
		}
	}

	// Make the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	log.Debug("response", "status", resp.StatusCode, "body", string(body))

	// Check for error response
	if !options.skipHttpError {
		if resp.StatusCode >= 400 {
			var apiErr Error
			if err := json.Unmarshal(body, &apiErr); err != nil {
				return fmt.Errorf("failed to decode error response: %w", err)
			}
			apiErr.Code = resp.StatusCode
			apiErr.Status = resp.Status
			return &apiErr
		}
	}

	// Decode response if output type provided
	if outputMaybe != nil {
		if err := json.Unmarshal(body, outputMaybe); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

type RequestActivateApiKey struct {
	ApiKey      string  `json:"api_key"`
	Connector   *bool   `json:"connector,omitempty"`
	Network     *string `json:"network,omitempty"`
	OtelEnabled *bool   `json:"otel_enabled,omitempty"`
}

func (c *Client) ActivateApiKey(apiKey string, connector *bool) error {
	return c.Do("POST", "/v1/activate/api-key", &RequestActivateApiKey{
		ApiKey:    apiKey,
		Connector: connector,
	}, nil)
}

type ActivateBinariesOptions struct {
	Version string
}

func (c *Client) ActivateBinaries(options ActivateBinariesOptions) error {
	query := url.Values{}
	if options.Version != "" {
		query.Set("version", options.Version)
	}
	return c.Do("POST", "/v1/activate/binaries", nil, nil, Options{query: query})
}

func (c *Client) ActivateNetwork() error {
	return c.Do("POST", "/v1/activate/network", nil, nil)
}

type RequestActivateBackup struct {
	// Required: list of backup keys
	Baks []panel.Bak `json:"baks"`
}

func (c *Client) ActivateBackup(baks []panel.Bak) error {
	return c.Do("POST", "/v1/activate/backup", &RequestActivateBackup{Baks: baks}, nil)
}

type RequestActivateOtel struct {
	// Required: boolean
	Enabled bool `json:"enabled"`
}

func (c *Client) ActivateOtel(enabled bool) error {
	return c.Do("POST", "/v1/activate/otel", &RequestActivateOtel{Enabled: enabled}, nil)
}

func (c *Client) TreasuryHealth() (json.RawMessage, error) {
	var resp json.RawMessage
	if err := c.Do("GET", "/v1/treasury/healthy", nil, &resp, Options{
		query:         url.Values{"verbose": {"true"}},
		skipHttpError: true,
	}); err != nil {
		return nil, err
	}
	return resp, nil
}

type ServiceState string

const ServiceStateActive ServiceState = "active"
const ServiceStateInactive ServiceState = "inactive"
const ServiceStateDeactivating ServiceState = "deactivating"
const ServiceStateActivating ServiceState = "activating"
const ServiceStateFailed ServiceState = "failed"

type Service struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	// These are mapped directly from systemd
	LoadState   string       `json:"load_state"`
	ActiveState ServiceState `json:"active_state"`
	SubState    string       `json:"sub_state"`
	JobType     string       `json:"job_type,omitempty"`
}

func (c *Client) ListServices() ([]Service, error) {
	var resp []Service
	if err := c.Do("GET", "/v1/services", nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) UpdateService(service string, action string) error {
	return c.Do("POST", "/v1/services/"+service+"/"+action, nil, nil)
}

func (c *Client) GetService(service string) (Service, error) {
	var resp Service
	if err := c.Do("GET", "/v1/services/"+service, nil, &resp); err != nil {
		return Service{}, err
	}
	return resp, nil
}

func (c *Client) GenerateTreasury() error {
	return c.Do("POST", "/v1/treasury", nil, nil)
}
func (c *Client) DeleteTreasury(supervisor bool) error {
	query := url.Values{}
	if supervisor {
		query.Set("supervisor", "")
	}
	return c.Do("DELETE", "/v1/treasury", nil, nil, Options{query: query})
}

func (c *Client) CompleteTreasury() error {
	return c.Do("POST", "/v1/treasury/complete", nil, nil)
}

func (c *Client) GetPanel() (*panel.Panel, error) {
	var resp panel.Panel
	if err := c.Do("GET", "/v1/panel", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

type Peer struct {
	// Required
	Socket string `json:"socket"`
	NodeId string `json:"node_id"`
	// Participant is required for signer peers.
	Participant int `json:"participant,omitempty"`

	// Optional:
	// Set only if the signer peers are different from the engine peers
	SignerSocket string `json:"signer_socket"`
}
type RequestSyncTreasuryPeers struct {
	// Peers (will be taken from admin API if omitted)
	Peers []Peer `json:"peers"`
	// Do not try to exclude peers that are referencing self
	Force *bool `json:"force,omitempty"`
	// The listen address [+port] to use for engine gossip
	Listen string `json:"listen,omitempty"`
	// The listen address [+port] to use for signer gossip
	ListenSigner string `json:"listen_signer,omitempty"`
}

func (c *Client) SyncTreasuryPeers() error {
	return c.Do("POST", "/v1/treasury/peers/sync", &RequestSyncTreasuryPeers{}, nil)
}
