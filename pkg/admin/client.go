package admin

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	baseUrl string
	apiKey  string
}

func NewClient(apiKey string) *Client {
	baseUrl := "https://admin.cordialapis.com"
	return &Client{baseUrl, apiKey}
}

type Error struct {
	Code    int    `json:"code"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Status, e.Message)
}

type requestOptions struct {
	queryArgs url.Values
}

func (c *Client) do(method string, url string, inputMaybe interface{}, outputMaybe interface{}, requestOptions requestOptions) error {
	var body io.Reader
	var jsonBytes []byte
	var err error
	if inputMaybe != nil {
		jsonBytes, err = json.Marshal(inputMaybe)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		body = bytes.NewReader(jsonBytes)
	}
	if requestOptions.queryArgs != nil {
		url += "?" + requestOptions.queryArgs.Encode()
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	apiKey := c.apiKey
	if apiKey != "" {
		if strings.Contains(apiKey, ":") {
			apiKey = base64.StdEncoding.EncodeToString([]byte(apiKey))
		}
		req.Header.Set("Authorization", "Basic "+apiKey)
	}
	slog.With("method", method, "url", url, "body", string(jsonBytes)).Debug("admin request")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	slog.With("method", method, "url", url, "body", string(respBody)).Debug("admin response")

	if resp.StatusCode > 201 {
		var apiErr Error
		if err := json.Unmarshal(respBody, &apiErr); err != nil {
			return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, respBody)
		}
		return &apiErr
	}

	if outputMaybe != nil {
		if err := json.Unmarshal(respBody, outputMaybe); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

func (c *Client) GetApiKey(apiKeyID string) (*ApiKey, error) {
	var input interface{} = nil
	var output = &ApiKey{}

	err := c.do("GET", c.baseUrl+"/v1/api-keys/"+apiKeyID, input, output, requestOptions{})
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (c *Client) GetNetworkKey(nodeName string) (string, error) {
	var input interface{} = nil
	networkKey := ""
	var output = &networkKey

	err := c.do("GET", c.baseUrl+"/v1/"+nodeName+"/network-key", input, output, requestOptions{})
	if err != nil {
		return "", err
	}
	return networkKey, nil
}

func (c *Client) GetNodeById(treasury string, node string) (*Node, error) {
	var input interface{} = nil
	var output = &Node{}

	err := c.do("GET", c.baseUrl+"/v1/treasuries/"+treasury+"/nodes/"+node, input, output, requestOptions{})
	if err != nil {
		return nil, err
	}
	return output, nil
}
func (c *Client) GetNode(name string) (*Node, error) {
	var input interface{} = nil
	var output = &Node{}

	err := c.do("GET", c.baseUrl+"/v1/"+name, input, output, requestOptions{})
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (c *Client) UpdateNode(name string, node *Node) (*Node, error) {
	var input = node
	var output = &Node{}

	err := c.do("PUT", c.baseUrl+"/v1/"+name, input, output, requestOptions{})
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (c *Client) ListNodes(treasury string) (*NodePage, error) {
	var input interface{} = nil
	var output = &NodePage{}

	err := c.do("GET", c.baseUrl+"/v1/treasuries/"+treasury+"/nodes", input, output, requestOptions{})
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (c *Client) ListUsers(nextPageToken string) (*UserPage, error) {
	var input interface{} = nil
	var output = &UserPage{}

	options := requestOptions{}
	if nextPageToken != "" {
		options.queryArgs = url.Values{}
		options.queryArgs.Set("page_token", nextPageToken)
	}

	err := c.do("GET", c.baseUrl+"/v1/users", input, output, options)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (c *Client) GetTreasuryById(treasuryId string) (*Treasury, error) {
	var input interface{} = nil
	var output = &Treasury{}

	err := c.do("GET", c.baseUrl+"/v1/treasuries/"+treasuryId, input, output, requestOptions{})
	if err != nil {
		return nil, err
	}
	return output, nil
}
func (c *Client) GetTreasury(name string) (*Treasury, error) {
	return c.GetTreasuryById(strings.TrimPrefix(name, "treasuries/"))
}
