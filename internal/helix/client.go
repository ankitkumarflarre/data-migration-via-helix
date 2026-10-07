// Package helix is a dependency-free client for the Helix storage engine's
// entity API (/api/entities). It mirrors _python/storage_engine_client.py:
// the same URL rules, credential precedence and redirect protection.
package helix

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Role decides which credential is used.
type Role string

const (
	Reader Role = "reader"
	Writer Role = "writer"
)

var scopes = map[Role]string{Reader: "helix:read", Writer: "helix:read,helix:write"}

// Keys in the env file a role may use, in order. A writer key can read.
var fileKeys = map[Role][]string{
	Reader: {"HARNESS_READER_KEY", "HARNESS_WRITER_KEY"},
	Writer: {"HARNESS_WRITER_KEY"},
}

// Config selects the deployment and where credentials come from.
type Config struct {
	URL     string // explicit URL; else HELIX_URL, HARNESS_URL, env file
	EnvFile string // KEY=VALUE file holding HARNESS_URL and HARNESS_*_KEY
	Timeout time.Duration
}

// Client talks to one Helix deployment.
type Client struct {
	base    string
	env     map[string]string
	http    *http.Client
	mu      sync.Mutex
	minted  map[Role]mintedToken
	mintCmd func(ctx context.Context, audience, scope string) (string, time.Duration, error)
}

type mintedToken struct {
	token   string
	expires time.Time
}

// Error is a non-2xx answer from Helix.
type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string { return fmt.Sprintf("helix HTTP %d: %s", e.Status, e.Message) }

// IsStatus reports whether err is a Helix error with the given status.
func IsStatus(err error, status int) bool {
	var he *Error
	return errors.As(err, &he) && he.Status == status
}

// New builds a client. The URL must be http(s) without credentials, query or fragment.
func New(cfg Config) (*Client, error) {
	env := readEnvFile(cfg.EnvFile)
	raw := firstNonEmpty(cfg.URL, os.Getenv("HELIX_URL"), os.Getenv("HARNESS_URL"), env["HARNESS_URL"])
	if raw == "" {
		return nil, errors.New("no Helix URL: set --helix-url, HELIX_URL, or HARNESS_URL in the env file")
	}
	base, err := BaseURL(raw)
	if err != nil {
		return nil, err
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	c := &Client{base: base, env: env, minted: map[Role]mintedToken{}, mintCmd: mycelMint}
	c.http = &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if req.Header.Get("Authorization") != "" && origin(via[0].URL) != origin(req.URL) {
				return errors.New("refusing an authenticated redirect to a different Helix origin")
			}
			return nil
		},
	}
	return c, nil
}

// Base returns the deployment URL.
func (c *Client) Base() string { return c.base }

// BaseURL validates and normalises a deployment URL.
func BaseURL(raw string) (string, error) {
	v := strings.TrimRight(strings.TrimSpace(raw), "/")
	u, err := url.Parse(v)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" ||
		u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("Helix URL must be an HTTP(S) URL without credentials, query or fragment")
	}
	return v, nil
}

func origin(u *url.URL) string {
	port := u.Port()
	if port == "" {
		port = map[string]string{"https": "443", "http": "80"}[u.Scheme]
	}
	return u.Scheme + "://" + strings.ToLower(u.Hostname()) + ":" + port
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func readEnvFile(path string) map[string]string {
	out := map[string]string{}
	if path == "" {
		return out
	}
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		k, v, _ := strings.Cut(line, "=")
		if v = strings.TrimSpace(v); v != "" {
			out[strings.TrimSpace(k)] = v
		}
	}
	return out
}

// token: HELIX_TOKEN > env-file key bound to this origin > `mycel token`.
func (c *Client) token(ctx context.Context, role Role) (string, error) {
	if t := os.Getenv("HELIX_TOKEN"); t != "" {
		return t, nil
	}
	baseURL, _ := url.Parse(c.base)
	for _, src := range []map[string]string{envMap(), c.env} {
		configured := src["HARNESS_URL"]
		if configured == "" {
			continue
		}
		cu, err := url.Parse(strings.TrimRight(configured, "/"))
		if err != nil || origin(cu) != origin(baseURL) {
			continue
		}
		for _, name := range fileKeys[role] {
			if src[name] != "" {
				return src[name], nil
			}
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if held, ok := c.minted[role]; ok && time.Until(held.expires) > time.Minute {
		return held.token, nil
	}
	tok, life, err := c.mintCmd(ctx, baseURL.Host, scopes[role])
	if err != nil {
		return "", err
	}
	c.minted[role] = mintedToken{tok, time.Now().Add(life)}
	return tok, nil
}

func envMap() map[string]string {
	out := map[string]string{}
	for _, k := range []string{"HARNESS_URL", "HARNESS_READER_KEY", "HARNESS_WRITER_KEY"} {
		if v := os.Getenv(k); v != "" {
			out[k] = v
		}
	}
	return out
}

func mycelMint(ctx context.Context, audience, scope string) (string, time.Duration, error) {
	cmd := exec.CommandContext(ctx, "mycel", "token", "--audience", audience, "--scope", scope, "--v")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil || strings.TrimSpace(stdout.String()) == "" {
		return "", 0, fmt.Errorf("`mycel token` failed: %s — run `mycel login` or set HARNESS_WRITER_KEY", strings.TrimSpace(stderr.String()))
	}
	life := 10 * time.Minute
	for _, line := range strings.Split(stderr.String(), "\n") {
		if f := strings.Fields(line); len(f) > 1 && f[0] == "expires" {
			if t, err := time.Parse(time.RFC3339, f[1]); err == nil {
				life = time.Until(t)
			}
		}
	}
	return strings.TrimSpace(stdout.String()), life, nil
}

// Do performs one call and decodes the `data` member of a 2xx JSON answer into out (if non-nil).
func (c *Client) Do(ctx context.Context, method, path string, role Role, query url.Values, headers map[string]string, body, out any) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	full := c.base + path
	if len(query) > 0 {
		full += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, full, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if role != "" {
		tok, err := c.token(ctx, role)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		var ne net.Error
		if errors.As(err, &ne) || strings.Contains(err.Error(), "redirect") {
			return fmt.Errorf("cannot reach %s: %w", c.base, err)
		}
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var env struct {
		Data  json.RawMessage `json:"data"`
		Error json.RawMessage `json:"error"`
	}
	_ = json.Unmarshal(raw, &env)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return &Error{Status: resp.StatusCode, Message: errorMessage(env.Error, raw)}
	}
	if out != nil && len(env.Data) > 0 {
		return json.Unmarshal(env.Data, out)
	}
	return nil
}

func errorMessage(e json.RawMessage, raw []byte) string {
	var obj struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(e, &obj) == nil && obj.Message != "" {
		return obj.Message
	}
	var s string
	if json.Unmarshal(e, &s) == nil && s != "" {
		return s
	}
	msg := strings.TrimSpace(string(raw))
	if len(msg) > 300 {
		msg = msg[:300]
	}
	return msg
}

func part(s string) string { return url.PathEscape(s) }

// Record is one stored record as the records API returns it.
type Record struct {
	ID      string         `json:"id"`
	Version int64          `json:"version"`
	Variant string         `json:"variant,omitempty"`
	Fields  map[string]any `json:"fields"`
}

// Create stores a new record of a leaf variant.
func (c *Client) Create(ctx context.Context, variant string, fields map[string]any, idempotencyKey string) (Record, error) {
	var rec Record
	h := map[string]string{}
	if idempotencyKey != "" {
		h["Idempotency-Key"] = idempotencyKey
	}
	err := c.Do(ctx, http.MethodPost, "/api/entities/records/"+part(variant), Writer, nil, h, map[string]any{"fields": fields}, &rec)
	return rec, err
}

// Get reads one record.
func (c *Client) Get(ctx context.Context, variant, id string) (Record, error) {
	var rec Record
	err := c.Do(ctx, http.MethodGet, "/api/entities/records/"+part(variant)+"/"+part(id), Reader, nil, nil, nil, &rec)
	return rec, err
}

// Patch updates fields; version must be the record's current version.
func (c *Client) Patch(ctx context.Context, variant, id string, version int64, fields map[string]any) (Record, error) {
	var rec Record
	err := c.Do(ctx, http.MethodPatch, "/api/entities/records/"+part(variant)+"/"+part(id), Writer, nil, nil,
		map[string]any{"version": version, "fields": fields}, &rec)
	return rec, err
}

// Delete removes a record.
func (c *Client) Delete(ctx context.Context, variant, id string) error {
	return c.Do(ctx, http.MethodDelete, "/api/entities/records/"+part(variant)+"/"+part(id), Writer, nil, nil, nil, nil)
}

// FindByKey returns records of an entity whose field equals value (where=field="value").
func (c *Client) FindByKey(ctx context.Context, entity, field, value string) ([]Record, error) {
	quoted, _ := json.Marshal(value)
	q := url.Values{"where": {field + "=" + string(quoted)}, "limit": {"2"}}
	var page struct {
		Records []Record `json:"records"`
	}
	err := c.Do(ctx, http.MethodGet, "/api/entities/list/"+part(entity), Reader, q, nil, nil, &page)
	return page.Records, err
}

// Field is one field of a variant as /describe reports it.
type Field struct {
	Key       string   `json:"key"`
	Type      string   `json:"type"`
	Enum      []string `json:"enum"`
	Required  bool     `json:"required"`
	WriteOnce bool     `json:"write_once"`
	Owner     string   `json:"owner"`
	Reference *struct {
		Entity string `json:"entity"`
	} `json:"reference"`
}

// Description is the /describe answer for one variant.
type Description struct {
	Entity string  `json:"entity"`
	Fields []Field `json:"fields"`
}

// Describe returns a variant's fields.
func (c *Client) Describe(ctx context.Context, variant string) (Description, error) {
	var d Description
	err := c.Do(ctx, http.MethodGet, "/api/entities/describe/"+part(variant), Reader, nil, nil, nil, &d)
	return d, err
}

// CatalogueEntity is the subset of /catalogue the migrator uses.
type CatalogueEntity struct {
	Entity string   `json:"entity"`
	Title  string   `json:"title"`
	Module string   `json:"module"`
	Leaves []string `json:"leaves"`
}

// Catalogue lists entities with their leaf variants.
func (c *Client) Catalogue(ctx context.Context) (bundle string, entities []CatalogueEntity, err error) {
	var out struct {
		BundleVersion string            `json:"bundle_version"`
		Entities      []CatalogueEntity `json:"entities"`
	}
	err = c.Do(ctx, http.MethodGet, "/api/entities/catalogue", Reader, nil, nil, nil, &out)
	return out.BundleVersion, out.Entities, err
}

// Ping checks reachability with /api/entities/readiness.
func (c *Client) Ping(ctx context.Context) error {
	return c.Do(ctx, http.MethodGet, "/api/entities/readiness", Reader, nil, nil, nil, nil)
}
