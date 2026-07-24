package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

// ---- HAProxy Data Plane API -------------------------------------------

type HAProxyBalance struct {
	Algorithm string `json:"algorithm,omitempty"`
}

type HAProxyBackend struct {
	Name    string                   `json:"name"`
	Mode    string                   `json:"mode"`              // http|tcp
	Balance *HAProxyBalance          `json:"balance,omitempty"` // roundrobin|leastconn|source...
	Servers map[string]HAProxyServer `json:"servers,omitempty"`
	Checks  *HAProxyHealthCheck      `json:"health_check,omitempty"`
}

type HAProxyServer struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Port    int64  `json:"port"`
	Check   string `json:"check,omitempty"`
	Weight  int64  `json:"weight,omitempty"`
}

type HAProxyHealthCheck struct {
	Type     string `json:"type"` // http|tcp
	URI      string `json:"uri,omitempty"`
	Interval int64  `json:"interval_ms,omitempty"`
}

func (c *Client) doHAProxyJSON(ctx context.Context, method, path string, body, out interface{}) error {
	// For mutating requests, fetch current config version first.
	if method == http.MethodPut || method == http.MethodPost || method == http.MethodDelete {
		ver, err := c.getHAProxyConfigVersion(ctx)
		if err != nil {
			return err
		}
		sep := "?"
		if strings.Contains(path, "?") {
			sep = "&"
		}
		path = fmt.Sprintf("%s%sversion=%d", path, sep, ver)
	}

	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("nextcloud: marshalling haproxy request: %w", err)
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.cfg.HAProxyDataPlaneURL+path, reader)
	if err != nil {
		return fmt.Errorf("nextcloud: building haproxy request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.HAProxyUsername != "" {
		req.SetBasicAuth(c.cfg.HAProxyUsername, c.cfg.HAProxyPassword)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("nextcloud: performing haproxy request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("nextcloud: reading haproxy response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}
	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("nextcloud: unmarshalling haproxy response: %w", err)
		}
	}
	return nil
}

func (c *Client) getHAProxyConfigVersion(ctx context.Context) (int64, error) {
	var version int64
	if err := c.doHAProxyJSON(ctx, http.MethodGet, "/services/haproxy/configuration/version", nil, &version); err != nil {
		return 0, fmt.Errorf("nextcloud: fetching haproxy config version: %w", err)
	}
	return version, nil
}

func (c *Client) GetHAProxyBackend(ctx context.Context, name string) (*HAProxyBackend, error) {
	var out HAProxyBackend
	if err := c.doHAProxyJSON(ctx, http.MethodGet, "/services/haproxy/configuration/backends/"+name, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateHAProxyBackend(ctx context.Context, b HAProxyBackend) (*HAProxyBackend, error) {
	var out HAProxyBackend
	if err := c.doHAProxyJSON(ctx, http.MethodPost, "/services/haproxy/configuration/backends", b, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateHAProxyBackend(ctx context.Context, b HAProxyBackend) (*HAProxyBackend, error) {
	var out HAProxyBackend
	if err := c.doHAProxyJSON(ctx, http.MethodPut, "/services/haproxy/configuration/backends/"+b.Name, b, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteHAProxyBackend(ctx context.Context, name string) error {
	return c.doHAProxyJSON(ctx, http.MethodDelete, "/services/haproxy/configuration/backends/"+name, nil, nil)
}

// ---- Ceph RGW admin ops API ------------------------------------------

type CephRGWUser struct {
	UserID      string `json:"uid"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email,omitempty"`
	AccessKey   string `json:"access_key,omitempty"`
	SecretKey   string `json:"secret_key,omitempty"`
	MaxBuckets  int64  `json:"max_buckets,omitempty"`
	Suspended   bool   `json:"suspended"`
}

func (c *Client) doCephJSON(ctx context.Context, method, path string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, method, c.cfg.CephRGWAdminURL+"/admin"+path, nil)
	if err != nil {
		return fmt.Errorf("nextcloud: building ceph request: %w", err)
	}

	signer := v4.NewSigner()
	emptyHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	creds := aws.Credentials{
		AccessKeyID:     c.cfg.CephRGWAccessKey,
		SecretAccessKey: c.cfg.CephRGWSecretKey,
	}
	if err := signer.SignHTTP(ctx, creds, req, emptyHash, "s3", "us-east-1", time.Now()); err != nil {
		return fmt.Errorf("nextcloud: signing ceph request: %w", err)
	}

	if dump, dumpErr := httputil.DumpRequestOut(req, true); dumpErr == nil {
		fmt.Fprintf(os.Stderr, "=== OUTGOING CEPH REQUEST ===\n%s\n=== END ===\n", dump)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("nextcloud: performing ceph request: %w", err)
	}
	defer resp.Body.Close()

	if dump, dumpErr := httputil.DumpResponse(resp, true); dumpErr == nil {
		fmt.Fprintf(os.Stderr, "=== CEPH RESPONSE ===\n%s\n=== END ===\n", dump)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("nextcloud: reading ceph response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}
	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("nextcloud: unmarshalling ceph response: %w", err)
		}
	}
	return nil
}

func (c *Client) GetCephRGWUser(ctx context.Context, uid string) (*CephRGWUser, error) {
	var out CephRGWUser
	q := url.Values{}
	q.Set("uid", uid)
	if err := c.doCephJSON(ctx, http.MethodGet, "/user?"+q.Encode(), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateCephRGWUser(ctx context.Context, u CephRGWUser) (*CephRGWUser, error) {
	var out CephRGWUser
	q := url.Values{}
	q.Set("uid", u.UserID)
	q.Set("display-name", u.DisplayName)
	if u.MaxBuckets != 0 {
		q.Set("max-buckets", strconv.FormatInt(u.MaxBuckets, 10))
	}
	if err := c.doCephJSON(ctx, http.MethodPut, "/user?"+q.Encode(), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteCephRGWUser(ctx context.Context, uid string) error {
	q := url.Values{}
	q.Set("uid", uid)
	return c.doCephJSON(ctx, http.MethodDelete, "/user?"+q.Encode(), nil)
}
