package treasury

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/cordialsys/panel/pkg/resource"
)

type Client struct {
	baseUrl string
}

func NewClient() *Client {
	baseUrl := "http://127.0.0.1:8777"
	return &Client{baseUrl}
}

func (c *Client) GetUser(id string) (*http.Response, error) {
	req, err := http.NewRequest("GET", c.baseUrl+"/v1/users/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	return resp, nil
}

func (c *Client) GetFeature(id string) (*resource.Feature, error) {
	req, err := http.NewRequest("GET", c.baseUrl+"/v1/features/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get feature: %v", string(body))
	}
	var feature resource.Feature
	err = json.Unmarshal(body, &feature)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %v", err)
	}
	return &feature, nil
}
