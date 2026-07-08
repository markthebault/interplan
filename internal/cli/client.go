package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/markthebault/interplan/internal/protocol"
)

type apiClient struct {
	base string
	http *http.Client
}

func newAPIClient(port int) apiClient {
	return newAPIClientForHost("127.0.0.1", port)
}

func newAPIClientForHost(host string, port int) apiClient {
	return apiClient{
		base: "http://" + host + ":" + strconv.Itoa(port),
		http: &http.Client{Timeout: 5 * time.Second},
	}
}

func (c apiClient) health() bool {
	resp, err := c.http.Get(c.base + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	var body struct {
		OK              bool   `json:"ok"`
		Name            string `json:"name"`
		ProtocolVersion int    `json:"protocol_version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return false
	}
	return resp.StatusCode == http.StatusOK && body.OK && body.Name == "interplan" && body.ProtocolVersion == 2
}

func (c apiClient) open(file string, reopen bool, publicHost string) (protocol.SessionResponse, int, error) {
	body, _ := json.Marshal(protocol.SessionRequest{File: file, Reopen: reopen, PublicHost: publicHost})
	var out protocol.SessionResponse
	status, err := c.doJSON(http.MethodPost, c.base+"/api/sessions", body, &out)
	return out, status, err
}

func (c apiClient) poll(file string, timeout time.Duration) (protocol.PollResponse, error) {
	values := url.Values{"file": []string{file}}
	if timeout > 0 {
		values.Set("timeoutMs", strconv.Itoa(int(timeout/time.Millisecond)))
		c.http.Timeout = timeout + 5*time.Second
	} else {
		c.http.Timeout = 0
	}
	var out protocol.PollResponse
	_, err := c.doJSON(http.MethodGet, c.base+"/api/poll?"+values.Encode(), nil, &out)
	return out, err
}

func (c apiClient) end(file string) (protocol.SessionResponse, error) {
	body, _ := json.Marshal(map[string]string{"file": file, "ended_by": "agent"})
	var out protocol.SessionResponse
	_, err := c.doJSON(http.MethodPost, c.base+"/api/end", body, &out)
	return out, err
}

func (c apiClient) agentReply(file, message string) error {
	body, _ := json.Marshal(map[string]string{"file": file, "message": message})
	_, err := c.doJSON(http.MethodPost, c.base+"/api/agent-reply", body, nil)
	return err
}

func (c apiClient) doJSON(method, endpoint string, body []byte, out any) (int, error) {
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, endpoint, reader)
	if err != nil {
		return 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if out == nil {
		if resp.StatusCode >= 400 {
			return resp.StatusCode, fmt.Errorf("server returned %s", resp.Status)
		}
		return resp.StatusCode, nil
	}
	err = json.NewDecoder(resp.Body).Decode(out)
	if err != nil {
		return resp.StatusCode, err
	}
	if resp.StatusCode >= 400 {
		return resp.StatusCode, fmt.Errorf("server returned %s", resp.Status)
	}
	return resp.StatusCode, nil
}
