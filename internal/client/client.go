// Package client implements a small HTTP client used by every resource in
// the provider.
//
// Design assumption:
//
//	Nextcloud/OCC, LDAP and OnlyOffice configuration cannot be managed over a
//	public REST API out of the box, so this provider assumes a lightweight
//	"nextcloud-agent" runs on each WFE (Web Front End) and exposes the handful
//	of endpoints below. The agent is intentionally simple: it shells out to
//	`occ`, edits config.php, and reports health. See /docs/agent.md for the
//	expected contract. HAProxy is talked to directly via its Data Plane API,
//	and Ceph RGW via its admin ops API, since both already expose a real
//	REST surface.
package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Config holds everything needed to talk to a single WFE's diskbg-agent.
type Config struct {
	// AgentURL is the base URL of the diskbg-agent running on the target WFE,
	// e.g. https://wfe-01.example.com:8443
	AgentURL string
	// AgentToken authenticates against the agent (Bearer token).
	AgentToken string

	// HAProxyDataPlaneURL is the base URL of the HAProxy Data Plane API,
	// e.g. https://lb-01.example.com:5555/v3
	HAProxyDataPlaneURL string
	HAProxyUsername     string
	HAProxyPassword     string

	// CephRGWAdminURL is the base URL of the Ceph RGW admin ops API,
	// e.g. https://ceph-rgw.example.com:8080/admin
	CephRGWAdminURL  string
	CephRGWAccessKey string
	CephRGWSecretKey string

	InsecureSkipVerify bool
}

// Client is the concrete client passed to every resource via
// req.ProviderData.
type Client struct {
	cfg  Config
	http *http.Client
}

func New(cfg Config) *Client {
	return &Client{
		cfg: cfg,
		http: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify}, //nolint:gosec // opt-in only
			},
		},
	}
}

// APIError is returned whenever the agent (or HAProxy/Ceph) responds with a
// non-2xx status code.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("nextcloud: request failed with status %d: %s", e.StatusCode, e.Body)
}

// doJSON performs a request against the diskbg-agent, marshalling `body` as
// the JSON payload (if non-nil) and unmarshalling the response into `out`
// (if non-nil).
func (c *Client) doJSON(ctx context.Context, method, path string, body, out interface{}) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("nextcloud: marshalling request body: %w", err)
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.cfg.AgentURL+path, reader)
	if err != nil {
		return fmt.Errorf("nextcloud: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.AgentToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.AgentToken)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("nextcloud: performing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("nextcloud: reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("nextcloud: unmarshalling response: %w", err)
		}
	}
	return nil
}

// ---- Nextcloud / config.php --------------------------------------------

type NextcloudConfig struct {
	MaintenanceMode bool     `json:"maintenance_mode"`
	TrustedDomains  []string `json:"trusted_domains"`
	TrustedProxies  []string `json:"trusted_proxies"`
	DefaultLanguage string   `json:"default_language"`
	DefaultQuota    string   `json:"default_quota"`
	UploadLimitMB   int64    `json:"upload_limit_mb"`
	LogLevel        int64    `json:"log_level"`
	BackgroundJobs  string   `json:"background_jobs"` // ajax|webcron|cron
	TrashRetention  string   `json:"trash_retention_days"`
	VersionsRetain  string   `json:"versions_retention_days"`
}

func (c *Client) GetNextcloudConfig(ctx context.Context) (*NextcloudConfig, error) {
	var out NextcloudConfig
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/nextcloud/config", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) PutNextcloudConfig(ctx context.Context, cfg NextcloudConfig) (*NextcloudConfig, error) {
	var out NextcloudConfig
	if err := c.doJSON(ctx, http.MethodPut, "/api/v1/nextcloud/config", cfg, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---- LDAP ----------------------------------------------------------------

type LDAPConfig struct {
	ConfigID        string `json:"config_id"`
	Host            string `json:"host"`
	Port            int64  `json:"port"`
	BaseDN          string `json:"base_dn"`
	BindDN          string `json:"bind_dn"`
	BindPassword    string `json:"bind_password,omitempty"`
	UserFilter      string `json:"user_filter"`
	GroupFilter     string `json:"group_filter"`
	LoginFilter     string `json:"login_filter"`
	UUIDAttribute   string `json:"uuid_attribute"`
	EmailAttribute  string `json:"email_attribute"`
	DisplayNameAttr string `json:"display_name_attribute"`
	NestedGroups    bool   `json:"nested_groups"`
	CacheTTLSeconds int64  `json:"cache_ttl_seconds"`
	TLS             bool   `json:"tls"`
}

func (c *Client) GetLDAPConfig(ctx context.Context, configID string) (*LDAPConfig, error) {
	var out LDAPConfig
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/ldap/"+configID, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpsertLDAPConfig(ctx context.Context, cfg LDAPConfig) (*LDAPConfig, error) {
	var out LDAPConfig
	if err := c.doJSON(ctx, http.MethodPut, "/api/v1/ldap/"+cfg.ConfigID, cfg, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteLDAPConfig(ctx context.Context, configID string) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/v1/ldap/"+configID, nil, nil)
}

func (c *Client) TestLDAPConnection(ctx context.Context, configID string) (bool, string, error) {
	var out struct {
		OK      bool   `json:"ok"`
		Message string `json:"message"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/ldap/"+configID+"/test", nil, &out); err != nil {
		return false, "", err
	}
	return out.OK, out.Message, nil
}

// ---- OnlyOffice ------------------------------------------------------------

type OnlyOfficeConfig struct {
	DocumentServerURL string `json:"document_server_url"`
	InternalURL       string `json:"internal_url,omitempty"`
	StorageURL        string `json:"storage_url,omitempty"`
	JWTSecret         string `json:"jwt_secret,omitempty"`
	JWTHeader         string `json:"jwt_header,omitempty"`
	JWTEnabled        bool   `json:"jwt_enabled"`
	VerifySSL         bool   `json:"verify_ssl"`
	ConnectionTimeout int64  `json:"connection_timeout_seconds"`
}

func (c *Client) GetOnlyOfficeConfig(ctx context.Context) (*OnlyOfficeConfig, error) {
	var out OnlyOfficeConfig
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/onlyoffice/config", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) PutOnlyOfficeConfig(ctx context.Context, cfg OnlyOfficeConfig) (*OnlyOfficeConfig, error) {
	var out OnlyOfficeConfig
	if err := c.doJSON(ctx, http.MethodPut, "/api/v1/onlyoffice/config", cfg, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ValidateOnlyOffice performs the end-to-end check described in the design
// doc: document server reachable, JWT secrets match, Nextcloud <-> Document
// Server round trip, and a test document open.
func (c *Client) ValidateOnlyOffice(ctx context.Context) (bool, []string, error) {
	var out struct {
		OK     bool     `json:"ok"`
		Checks []string `json:"checks"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/onlyoffice/validate", nil, &out); err != nil {
		return false, nil, err
	}
	return out.OK, out.Checks, nil
}

// ---- Users / Groups (Nextcloud Provisioning API, proxied by the agent) ---

type User struct {
	UserID      string            `json:"user_id"`
	DisplayName string            `json:"display_name"`
	Email       string            `json:"email"`
	Password    string            `json:"password,omitempty"`
	QuotaBytes  string            `json:"quota,omitempty"`
	Groups      []string          `json:"groups,omitempty"`
	Enabled     bool              `json:"enabled"`
	Attributes  map[string]string `json:"attributes,omitempty"`
}

func (c *Client) GetUser(ctx context.Context, userID string) (*User, error) {
	var out User
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/users/"+userID, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateUser(ctx context.Context, u User) (*User, error) {
	var out User
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/users", u, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateUser(ctx context.Context, u User) (*User, error) {
	var out User
	if err := c.doJSON(ctx, http.MethodPut, "/api/v1/users/"+u.UserID, u, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteUser(ctx context.Context, userID string) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/v1/users/"+userID, nil, nil)
}

// ---- Apps ------------------------------------------------------------------

type App struct {
	AppID   string            `json:"app_id"`
	Enabled bool              `json:"enabled"`
	Version string            `json:"version,omitempty"`
	Config  map[string]string `json:"config,omitempty"`
}

func (c *Client) GetApp(ctx context.Context, appID string) (*App, error) {
	var out App
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/apps/"+appID, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpsertApp(ctx context.Context, a App) (*App, error) {
	var out App
	if err := c.doJSON(ctx, http.MethodPut, "/api/v1/apps/"+a.AppID, a, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteApp(ctx context.Context, appID string) error {
	// "delete" for an app resource means disable+remove, mirroring
	// `occ app:remove`.
	return c.doJSON(ctx, http.MethodDelete, "/api/v1/apps/"+appID, nil, nil)
}

// ---- OCC commands ----------------------------------------------------------

type OccResult struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

// RunOcc executes an arbitrary occ command via the agent. Because `occ`
// commands are inherently imperative, the resource built on top of this
// treats every apply as "run again", using triggers/keepers on the Terraform
// side to control re-execution.
func (c *Client) RunOcc(ctx context.Context, command string, args []string) (*OccResult, error) {
	var out OccResult
	fullArgs := append([]string{command}, args...)
	payload := map[string]interface{}{"args": fullArgs}
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/occ/exec", payload, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---- Health ------------------------------------------------------------

type HealthStatus struct {
	Apache     bool `json:"apache"`
	PHP        bool `json:"php"`
	Nextcloud  bool `json:"nextcloud"`
	LDAP       bool `json:"ldap"`
	PostgreSQL bool `json:"postgresql"`
	Redis      bool `json:"redis"`
	CephRGW    bool `json:"ceph_rgw"`
	OnlyOffice bool `json:"onlyoffice"`
}

func (c *Client) GetHealth(ctx context.Context) (*HealthStatus, error) {
	var out HealthStatus
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/health", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---- Redis -----------------------------------------------------------------

type RedisConfig struct {
	Host              string `json:"host"`
	Port              int64  `json:"port"`
	Password          string `json:"password,omitempty"`
	UseForCache       bool   `json:"use_for_cache"`
	UseForFileLocking bool   `json:"use_for_file_locking"`
	Timeout           int64  `json:"timeout_seconds"`
}

func (c *Client) GetRedisConfig(ctx context.Context) (*RedisConfig, error) {
	var out RedisConfig
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/redis/config", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) PutRedisConfig(ctx context.Context, cfg RedisConfig) (*RedisConfig, error) {
	var out RedisConfig
	if err := c.doJSON(ctx, http.MethodPut, "/api/v1/redis/config", cfg, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
