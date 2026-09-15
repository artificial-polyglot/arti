package runpod

// Package runpod submits jobs to a RunPod serverless endpoint and, optionally,
// tracks them to completion.
//
// It does not run the workload itself. The endpoint is configured to pull the
// target Docker image (e.g. the Arti image) and run it with the parameters
// carried in the request. This package just hands RunPod the request and
// reports the job ID — so the same logic can be driven from a command
// (cmd/runpod), from a test, or from any other Go code, without shelling out
// to a binary.
//
// Functions return errors rather than calling os.Exit, so callers decide how
// to handle failure. Submit returns as soon as the job is queued; it does not
// wait for the run. Completion is normally reported by the workload itself via
// the request's NotifyOk/NotifyErr fields, so Track is provided but not used by
// the command.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// DefaultEndpoint is the RunPod serverless endpoint used unless RUNPOD_ENDPOINT
// overrides it. The endpoint's deployment decides which image is pulled.
const DefaultEndpoint = "fzfyyzux8a5dhi"

// Client submits jobs to a RunPod endpoint.
type Client struct {
	Endpoint   string
	APIKey     string
	HTTPClient *http.Client // optional; defaults to http.DefaultClient
}

// New builds a Client from the environment:
//
//	RUNNING_PHESANT  RunPod API key (required)
//	RUNPOD_ENDPOINT  endpoint ID (optional; defaults to DefaultEndpoint)
//
// It returns an error if the API key is not set.
func New() (*Client, error) {
	apiKey := os.Getenv("RUNNING_PHESANT")
	if apiKey == "" {
		return nil, errors.New("RUNNING_PHESANT (RunPod API key) is not set")
	}
	endpoint := os.Getenv("RUNPOD_ENDPOINT")
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	return &Client{Endpoint: endpoint, APIKey: apiKey}, nil
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

// SubmitFile reads the request YAML at yamlPath and submits it, returning the
// queued job ID.
func (c *Client) SubmitFile(appName, yamlPath string) (string, error) {
	yamlBytes, err := os.ReadFile(yamlPath)
	if err != nil {
		return "", fmt.Errorf("cannot read %s: %w", yamlPath, err)
	}
	return c.Submit(appName, string(yamlBytes))
}

// Submit sends one job to RunPod and returns the queued job ID. It returns as
// soon as the job is accepted; it does not wait for the run to finish.
func (c *Client) Submit(appName, yamlText string) (string, error) {
	// The request the RunPod handler expects. No image version is sent here —
	// the endpoint deployment determines the image pulled.
	payload := map[string]any{
		"input": map[string]any{
			"request_yaml":    yamlText,
			"appName":         appName,
			"timeout_minutes": 1000,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("cannot encode payload: %w", err)
	}

	url := fmt.Sprintf("https://api.runpod.ai/v2/%s/run", c.Endpoint)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.APIKey)

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("RunPod returned %d: %s", resp.StatusCode, string(respBody))
	}

	var job struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respBody, &job); err != nil {
		return "", fmt.Errorf("cannot parse response %q: %w", string(respBody), err)
	}
	return job.ID, nil
}

// Track polls RunPod every 30 seconds until the job completes or fails and
// returns the final status ("COMPLETED" or "FAILED").
//
// The command does not call this — the workload reports its own completion via
// the request's NotifyOk/NotifyErr. It's here for the occasional case where you
// want to watch a run interactively from Go.
func (c *Client) Track(jobID string) (string, error) {
	statusURL := fmt.Sprintf("https://api.runpod.ai/v2/%s/status/%s", c.Endpoint, jobID)
	for {
		time.Sleep(30 * time.Second)

		req, err := http.NewRequest(http.MethodGet, statusURL, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", c.APIKey)

		resp, err := c.httpClient().Do(req)
		if err != nil {
			return "", err
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return "", err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", fmt.Errorf("RunPod returned %d: %s", resp.StatusCode, string(body))
		}

		var status struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(body, &status); err != nil {
			return "", err
		}
		if status.Status == "COMPLETED" || status.Status == "FAILED" {
			return status.Status, nil
		}
	}
}

// Notify posts a message to the ntfy topic arti2. This is optional; completion
// notifications normally come from the workload itself. NTFY_API_TOKEN supplies
// the bearer token.
func Notify(message string) error {
	req, err := http.NewRequest(
		http.MethodPost,
		"https://ntfy.sh/arti2",
		bytes.NewBufferString(message),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("NTFY_API_TOKEN"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
