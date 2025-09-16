package server

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/johnwmail/nclip/internal/config"
	"github.com/johnwmail/nclip/internal/slug"
	"github.com/johnwmail/nclip/internal/storage"
)

// HTTPServer handles HTTP requests
type HTTPServer struct {
	config  *config.Config
	storage storage.Storage
	slugGen *slug.Generator
	server  *http.Server
	logger  *slog.Logger

	// rate limiting
	limiter *rateLimiter
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(cfg *config.Config, storage storage.Storage, logger *slog.Logger) *HTTPServer {
	srv := &HTTPServer{
		config:  cfg,
		storage: storage,
		slugGen: slug.New(cfg.SlugLength),
		logger:  logger,
	}
	// initialize rate limiter
	srv.limiter = newRateLimiter(cfg, logger)
	return srv
}

// Start starts the HTTP server
func (s *HTTPServer) Start() error {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc("/raw/", s.handleRaw)
	mux.HandleFunc("/download/", s.handleDownload)
	mux.HandleFunc("/burn/", s.handleBurn)

	if s.config.EnableMetrics {
		mux.HandleFunc("/metrics", s.handleMetrics)
	}

	mux.HandleFunc("/health", s.handleHealth)

	// Add middleware: CORS -> RateLimit -> Logging
	handler := s.loggingMiddleware(s.rateLimitMiddleware(s.corsMiddleware(mux)))

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.HTTPPort),
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	s.logger.Info("HTTP server started", "address", s.server.Addr)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP server error", "error", err)
		}
	}()

	return nil
}

// Stop stops the HTTP server
func (s *HTTPServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}

// handleRoot handles both GET (view paste) and POST (create paste) requests
func (s *HTTPServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleCreatePaste(w, r)
	case http.MethodGet:
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			// Respect EnableWebUI flag
			if !s.config.EnableWebUI {
				http.NotFound(w, r)
				return
			}
			s.handleIndex(w, r)
		} else {
			s.handleViewPaste(w, r, path)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleIndex shows the main page
func (s *HTTPServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <title>nclip</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; 
               margin: 40px auto; max-width: 800px; line-height: 1.6; color: #333; }
        .header { text-align: center; margin-bottom: 40px; }
		.methods { display: grid; grid-template-columns: 1fr; gap: 30px; margin: 30px 0; }
		.method { padding: 20px; border: 1px solid #ddd; border-radius: 8px; background: #fff; }
		.method h3 { margin-top: 0; color: #0066cc; }
        .code { background: #f5f5f5; padding: 10px; border-radius: 4px; font-family: monospace; }
        .footer { text-align: center; margin-top: 40px; color: #666; font-size: 0.9em; }
        @media (max-width: 600px) { .methods { grid-template-columns: 1fr; } }
		textarea { width: 100%; min-height: 140px; font-family: monospace; }
		.row { display: flex; gap: 10px; align-items: center; }
		.row > * { flex: 1; }
    </style>
</head>
<body>
    <div class="header">
        <h1>🚀 nclip</h1>
        <p>Share terminal output and code snippets</p>
    </div>
    
	<div class="methods">
		<div class="method">
            <h3>🌐 HTTP (curl)</h3>
            <p>Use curl to paste via HTTP:</p>
            <div class="code">
                echo "Hello World" | curl -d @- {{.BaseURL}}<br>
                curl -d @file.txt {{.BaseURL}}
            </div>
        </div>

		<div class="method">
			<h3>📝 Web UI</h3>
			<form id="pasteForm">
				<div class="row">
					<textarea id="text" placeholder="Type or paste text here..."></textarea>
				</div>
				<div class="row">
					<input type="file" id="file" />
					<button type="submit">Create Paste</button>
				</div>
			</form>
			<div id="result" class="code" style="display:none;"></div>
			<div id="stats" style="margin-top:10px;color:#666;font-size:0.9em;"></div>
		</div>
    </div>
    
    <div class="footer">
        <p>Powered by nclip • <a href="/health">Status</a> • <a href="https://github.com/johnwmail/nclip">GitHub</a></p>
    </div>

	<script>
	const form = document.getElementById('pasteForm');
	const fileInput = document.getElementById('file');
	const textInput = document.getElementById('text');
	const result = document.getElementById('result');
	const stats = document.getElementById('stats');

	form.addEventListener('submit', async (e) => {
		e.preventDefault();
		let body, headers = {};
		if (fileInput.files.length > 0) {
			const file = fileInput.files[0];
			headers['X-Filename'] = file.name;
			body = await file.arrayBuffer();
			body = new Uint8Array(body);
			headers['Content-Type'] = 'application/octet-stream';
		} else {
			body = textInput.value;
			headers['Content-Type'] = 'text/plain; charset=utf-8';
		}
		const resp = await fetch('/', { method: 'POST', body, headers });
		const url = await resp.text();
		result.style.display = 'block';
		result.textContent = url.trim();
	});

	async function refreshStats(){
		try{
			const r = await fetch('/health');
			if (!r.ok) return;
			const j = await r.json();
			stats.textContent = 'Pastes: ' + j.stats.total_pastes + ' • Size: ' + j.stats.total_size + ' bytes';
		}catch(_){ }
	}
	refreshStats();
	</script>
</body>
</html>`

	t, err := template.New("index").Parse(tmpl)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	data := struct {
		BaseURL string
	}{
		BaseURL: deriveBaseURL(s.config.GetBaseURL(), r),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.Execute(w, data); err != nil {
		s.logger.Error("Failed to execute template", "error", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
}

// handleCreatePaste handles POST requests to create new pastes
func (s *HTTPServer) handleCreatePaste(w http.ResponseWriter, r *http.Request) {
	// Read content from body
	content, err := io.ReadAll(io.LimitReader(r.Body, s.config.BufferSize))
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	if len(content) == 0 {
		http.Error(w, "Empty paste", http.StatusBadRequest)
		return
	}

	// Handle URL-encoded content (from curl -d @-)
	// curl -d removes newlines and sends as form data, but we want to treat it as raw text
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		// For form data, curl -d sends the content as a single form field
		// We need to parse it as form data first, then get the raw value
		if err := r.ParseForm(); err == nil && len(r.Form) > 0 {
			// Get the first (and usually only) form value which contains our content
			for key, values := range r.Form {
				if len(values) > 0 {
					// The key itself contains the content when using curl -d @-
					// because curl sends it as key=&
					if key != "" {
						content = []byte(key)
						break
					}
					// Fallback: use the value if key is empty
					if values[0] != "" {
						content = []byte(values[0])
						break
					}
				}
			}
		}
	}

	// Ensure content ends with newline (like original fiche behavior)
	// This handles curl -d @- which strips trailing newlines
	if len(content) > 0 && content[len(content)-1] != '\n' {
		content = append(content, '\n')
	}

	// Get client IP
	clientIP := getClientIP(r)

	// Extract metadata from headers
	if contentType == "" {
		contentType = "text/plain"
	}

	filename := r.Header.Get("X-Filename")
	language := r.Header.Get("X-Language")
	title := r.Header.Get("X-Title")
	expiresHeader := r.Header.Get("X-Expires")

	// Parse expiration
	var expiresIn time.Duration
	if expiresHeader != "" {
		if d, err := time.ParseDuration(expiresHeader); err == nil {
			expiresIn = d
		}
	}

	// Generate unique slug
	slugStr, err := s.slugGen.GenerateWithCollisionCheck(s.storage.Exists)
	if err != nil {
		s.logger.Error("Failed to generate slug", "error", err)
		http.Error(w, "Could not generate paste ID", http.StatusInternalServerError)
		return
	}

	// Create paste
	paste := &storage.Paste{
		ID:          slugStr,
		Content:     content,
		ContentType: contentType,
		Filename:    filename,
		Language:    language,
		Title:       title,
		CreatedAt:   time.Now(),
		ClientIP:    clientIP,
		Size:        int64(len(content)),
	}

	// Set expiration
	if expiresIn > 0 {
		expiresAt := time.Now().Add(expiresIn)
		paste.ExpiresAt = &expiresAt
	} else if expiration := s.config.GetExpiration(); expiration > 0 {
		expiresAt := time.Now().Add(expiration)
		paste.ExpiresAt = &expiresAt
	}

	// Store paste
	if err := s.storage.Store(paste); err != nil {
		s.logger.Error("Failed to store paste", "slug", slugStr, "error", err)
		http.Error(w, "Could not save paste", http.StatusInternalServerError)
		return
	}

	// Determine base URL: prefer explicit cfg.BaseURL; else derive from request
	baseURL := deriveBaseURL(s.config.GetBaseURL(), r)

	// Generate URL
	url := config.JoinBaseURLAndSlug(baseURL, slugStr)

	// Return URL (with newline for terminal compatibility)
	w.Header().Set("Content-Type", "text/plain")
	if _, err := fmt.Fprintf(w, "%s\n", url); err != nil {
		s.logger.Error("Failed to write response", "error", err)
	}

	s.logger.Info("Paste created via HTTP",
		"slug", slugStr,
		"client", clientIP,
		"size", len(content),
		"content_type", contentType)
}

// handleViewPaste displays a paste with basic formatting or raw content based on request
func (s *HTTPServer) handleViewPaste(w http.ResponseWriter, r *http.Request, id string) {
	// Support /burn/<id> path alias
	burn := false
	if strings.HasPrefix(id, "burn/") {
		burn = true
		id = strings.TrimPrefix(id, "burn/")
	}

	paste, err := s.storage.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Check if this is a browser request or command-line tool
	// Browser requests typically have Accept: text/html and User-Agent with browser info
	userAgent := r.Header.Get("User-Agent")
	accept := r.Header.Get("Accept")

	// Explicit raw override via query param
	if r.URL.Query().Get("raw") == "true" {
		userAgent = "curl" // force raw
	}

	// If it's curl, wget, or similar command-line tools, serve raw content (like original fiche)
	if isCLIRequest(userAgent, accept) {
		w.Header().Set("Content-Type", paste.ContentType)
		if _, err := w.Write(paste.Content); err != nil {
			s.logger.Error("Failed to write paste content", "error", err)
		}

		// Burn after reading if requested
		if burn || r.URL.Query().Get("burn") == "true" {
			_ = s.storage.Delete(id)
		}
		return
	}

	// Browser request - serve HTML with formatting
	s.handleViewPasteHTML(w, r, paste)

	// Burn after reading if requested for browser view
	if burn || r.URL.Query().Get("burn") == "true" {
		_ = s.storage.Delete(id)
	}
}

// isCLIRequest detects if the request is from a command-line tool vs browser
func isCLIRequest(userAgent, accept string) bool {
	// Check for common CLI tools
	cliTools := []string{"curl", "wget", "HTTPie", "Go-http-client"}
	for _, tool := range cliTools {
		if strings.Contains(userAgent, tool) {
			return true
		}
	}

	// If no Accept header or doesn't accept HTML, treat as CLI
	if accept == "" || !strings.Contains(accept, "text/html") {
		return true
	}

	// Empty or minimal User-Agent also suggests CLI tool
	if userAgent == "" || len(userAgent) < 10 {
		return true
	}

	return false
}

// handleViewPasteHTML renders the paste as HTML for browser viewing
func (s *HTTPServer) handleViewPasteHTML(w http.ResponseWriter, r *http.Request, paste *storage.Paste) {
	// Simple HTML escaping for now (we'll add syntax highlighting later)
	content := template.HTMLEscapeString(string(paste.Content))

	// Render template
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}} - nclip</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; 
               margin: 0; background: #f8f9fa; }
        .header { background: white; padding: 20px; border-bottom: 1px solid #e9ecef; margin-bottom: 20px; }
        .container { max-width: 1200px; margin: 0 auto; padding: 0 20px; }
        .meta { color: #666; font-size: 0.9em; margin-bottom: 10px; }
        .actions { margin-bottom: 20px; }
        .btn { display: inline-block; padding: 8px 16px; background: #007bff; color: white; 
               text-decoration: none; border-radius: 4px; margin-right: 10px; }
        .content { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        pre { margin: 0; overflow-x: auto; background: #f8f9fa; padding: 15px; border-radius: 4px; 
              font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace; line-height: 1.4; }
        code { font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace; }
    </style>
</head>
<body>
    <div class="header">
        <div class="container">
            <h1>{{.Title}}</h1>
            <div class="meta">
                Created: {{.CreatedAt.Format "2006-01-02 15:04:05 UTC"}} • 
                Size: {{.Size}} bytes • 
                Type: {{.ContentType}}
                {{if .ExpiresAt}} • Expires: {{.ExpiresAt.Format "2006-01-02 15:04:05 UTC"}}{{end}}
            </div>
            <div class="actions">
                <a href="/raw/{{.ID}}" class="btn">Raw</a>
                <a href="/download/{{.ID}}" class="btn">Download</a>
            </div>
        </div>
    </div>
    
    <div class="container">
        <div class="content">
            <pre><code>{{.Content}}</code></pre>
        </div>
    </div>
</body>
</html>`

	t, err := template.New("paste").Parse(tmpl)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	title := paste.Title
	if title == "" {
		title = fmt.Sprintf("Paste %s", paste.ID)
	}

	data := struct {
		*storage.Paste
		Title   string
		Content string
	}{
		Paste:   paste,
		Title:   title,
		Content: content,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.Execute(w, data); err != nil {
		s.logger.Error("Failed to execute paste template", "error", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
}

// handleRaw returns the raw paste content
func (s *HTTPServer) handleRaw(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/raw/")
	if id == "" {
		http.NotFound(w, r)
		return
	}

	paste, err := s.storage.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// For raw endpoint, always use text/plain so browsers display inline
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if _, err := w.Write(paste.Content); err != nil {
		s.logger.Error("Failed to write raw paste content", "error", err)
	}

	// Optional burn after reading
	if r.URL.Query().Get("burn") == "true" {
		_ = s.storage.Delete(id)
	}
}

// handleDownload returns the paste as a downloadable file
func (s *HTTPServer) handleDownload(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/download/")
	if id == "" {
		http.NotFound(w, r)
		return
	}

	paste, err := s.storage.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	filename := paste.Filename
	if filename == "" {
		filename = fmt.Sprintf("paste_%s.txt", paste.ID)
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	if _, err := w.Write(paste.Content); err != nil {
		s.logger.Error("Failed to write download content", "error", err)
	}

	// Optional burn after reading
	if r.URL.Query().Get("burn") == "true" {
		_ = s.storage.Delete(id)
	}
}

// handleHealth returns health status
func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	stats, err := s.storage.Stats()
	if err != nil {
		http.Error(w, "Storage error", http.StatusInternalServerError)
		return
	}

	health := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UTC(),
		"http_port": s.config.HTTPPort,
		"stats":     stats,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(health); err != nil {
		s.logger.Error("Failed to encode health response", "error", err)
	}
}

// handleMetrics returns basic metrics (placeholder for now)
func (s *HTTPServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	stats, err := s.storage.Stats()
	if err != nil {
		http.Error(w, "Storage error", http.StatusInternalServerError)
		return
	}

	// Simple text-based metrics format
	w.Header().Set("Content-Type", "text/plain")
	// Metrics writes are best-effort; errors are logged but not fatal
	writeMetric := func(format string, args ...interface{}) {
		if _, err := fmt.Fprintf(w, format, args...); err != nil {
			s.logger.Debug("Failed to write metric", "error", err)
		}
	}

	writeMetric("# HELP nclip_total_pastes Total number of pastes\n")
	writeMetric("# TYPE nclip_total_pastes counter\n")
	writeMetric("nclip_total_pastes %d\n", stats.TotalPastes)
	writeMetric("# HELP nclip_total_size_bytes Total storage used in bytes\n")
	writeMetric("# TYPE nclip_total_size_bytes counter\n")
	writeMetric("nclip_total_size_bytes %d\n", stats.TotalSize)
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.Split(xff, ",")[0]
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Use remote address
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// loggingMiddleware logs HTTP requests
func (s *HTTPServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}

		next.ServeHTTP(wrapped, r)

		s.logger.Info("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"client", getClientIP(r),
			"status", wrapped.statusCode,
			"duration", time.Since(start))
	})
}

// corsMiddleware adds CORS headers
func (s *HTTPServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Filename, X-Language, X-Title, X-Expires")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// TestHandler exposes the root handler for testing
func (s *HTTPServer) TestHandler() http.HandlerFunc {
	return s.handleRoot
}

// GetHandler returns the complete HTTP handler with middleware for use in Lambda
func (s *HTTPServer) GetHandler() http.Handler {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc("/raw/", s.handleRaw)
	mux.HandleFunc("/download/", s.handleDownload)
	mux.HandleFunc("/burn/", s.handleBurn)

	if s.config.EnableMetrics {
		mux.HandleFunc("/metrics", s.handleMetrics)
	}

	mux.HandleFunc("/health", s.handleHealth)

	// Add middleware
	return s.loggingMiddleware(s.rateLimitMiddleware(s.corsMiddleware(mux)))
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// deriveBaseURL returns explicit baseURL if set, otherwise builds from request headers
func deriveBaseURL(baseURL string, r *http.Request) string {
	if baseURL != "" {
		return baseURL
	}
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}

// handleBurn rewrites to view with burn=true
func (s *HTTPServer) handleBurn(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/burn/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	// append burn=true to query
	q := r.URL.Query()
	q.Set("burn", "true")
	r.URL.RawQuery = q.Encode()
	s.handleViewPaste(w, r, id)
}

// ---------- Rate limiting ----------
type rateSpec struct {
	limit  int
	window time.Duration
}

type rateLimiter struct {
	global   rateCounter
	perIP    map[string]*rateCounter
	perIPLim rateSpec
	logger   *slog.Logger
}

type rateCounter struct {
	spec       rateSpec
	windowBase time.Time
	count      int
}

func newRateLimiter(cfg *config.Config, logger *slog.Logger) *rateLimiter {
	gspec := parseRateSpec(cfg.RateLimitGlobal)
	ipspec := parseRateSpec(cfg.RateLimitPerIP)
	if gspec.limit <= 0 {
		gspec = rateSpec{limit: 0, window: time.Minute}
	}
	if ipspec.limit <= 0 {
		ipspec = rateSpec{limit: 0, window: time.Minute}
	}
	return &rateLimiter{
		global:   rateCounter{spec: gspec},
		perIP:    make(map[string]*rateCounter),
		perIPLim: ipspec,
		logger:   logger,
	}
}

func parseRateSpec(s string) rateSpec {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return rateSpec{limit: 0, window: time.Minute}
	}
	// accept formats like "60/min", "60 per minute", "60/minute"
	s = strings.ReplaceAll(s, "per", "/")
	s = strings.ReplaceAll(s, " ", "")
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return rateSpec{limit: 0, window: time.Minute}
	}
	n, err := strconv.Atoi(parts[0])
	if err != nil || n < 0 {
		return rateSpec{limit: 0, window: time.Minute}
	}
	unit := parts[1]
	switch unit {
	case "s", "sec", "secs", "second", "seconds":
		return rateSpec{limit: n, window: time.Second}
	case "m", "min", "mins", "minute", "minutes":
		return rateSpec{limit: n, window: time.Minute}
	case "h", "hr", "hour", "hours":
		return rateSpec{limit: n, window: time.Hour}
	default:
		return rateSpec{limit: n, window: time.Minute}
	}
}

func (rl *rateLimiter) allow(ip string) bool {
	now := time.Now()
	// global
	if rl.global.spec.limit > 0 {
		if !rl.global.inc(now) {
			return false
		}
	}
	// per IP
	if rl.perIPLim.limit > 0 {
		rc, ok := rl.perIP[ip]
		if !ok {
			rc = &rateCounter{spec: rl.perIPLim}
			rl.perIP[ip] = rc
		}
		if !rc.inc(now) {
			return false
		}
	}
	return true
}

func (rc *rateCounter) inc(now time.Time) bool {
	base := now.Truncate(rc.spec.window)
	if rc.windowBase != base {
		rc.windowBase = base
		rc.count = 0
	}
	rc.count++
	return rc.count <= rc.spec.limit
}

func (s *HTTPServer) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always allow health and metrics and OPTIONS
		if r.Method == http.MethodOptions || r.URL.Path == "/health" || r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}
		ip := getClientIP(r)
		if s.limiter != nil && !s.limiter.allow(ip) {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
