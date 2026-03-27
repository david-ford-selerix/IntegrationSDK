package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"
)

//go:embed ui/*
var embeddedUI embed.FS

type appConfig struct {
	EnrollmentURL string        `json:"enrollmentUrl"`
	Username      string        `json:"username"`
	Password      string        `json:"password"`
	Timeout       time.Duration `json:"-"`
}

type samlPreviewRequest struct {
	Version          string   `json:"version"`
	AcsURL           string   `json:"acsUrl"`
	EmployeeID       string   `json:"employeeId"`
	TransmittalXML   string   `json:"transmittalXml"`
	LayoutAttributes []string `json:"layoutAttributes"`
}

type soapResponse struct {
	StatusCode int
	Body       string
}

func main() {
	ctx, stop := signalNotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := loadConfig()

	uiFS, err := uiFileSystem()
	if err != nil {
		log.Fatalf("load ui: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, uiFS, cfg)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	appURL := fmt.Sprintf("http://%s", ln.Addr().String())
	log.Printf("IntegrationSDK app running at %s", appURL)

	srv := &http.Server{
		Handler:           loggingMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if serveErr := srv.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			log.Fatalf("server: %v", serveErr)
		}
	}()

	if openErr := openBrowser(appURL); openErr != nil {
		log.Printf("could not open browser automatically: %v", openErr)
		log.Printf("please open this URL manually: %s", appURL)
	}

	<-ctx.Done()
	log.Println("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}

func registerRoutes(mux *http.ServeMux, uiFS fs.FS, cfg appConfig) {
	mux.Handle("/", http.FileServer(http.FS(uiFS)))
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"enrollmentUrl":  cfg.EnrollmentURL,
			"username":       cfg.Username,
			"password":       cfg.Password,
			"timeoutSeconds": int(cfg.Timeout.Seconds()),
		})
	})

	mux.HandleFunc("/api/templates", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"uploadGroup":  uploadGroupTemplate(),
			"uploadCensus": uploadCensusTemplate(),
			"getGroup":     getGroupTemplate(),
			"getCensus":    getCensusTemplate(),
		})
	})

	mux.HandleFunc("/api/service/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		handleServiceCall(w, r, cfg, "Upload", "UploadResult")
	})

	mux.HandleFunc("/api/service/query", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		handleServiceCall(w, r, cfg, "Query", "QueryResult")
	})

	mux.HandleFunc("/api/service/get-login-guid", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Endpoint    string `json:"endpoint"`
			Username    string `json:"username"`
			Password    string `json:"password"`
			PortfolioID string `json:"portfolioId"`
			EmployeeID  string `json:"employeeId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request payload"})
			return
		}

		endpoint := strings.TrimSpace(req.Endpoint)
		if endpoint == "" {
			endpoint = cfg.EnrollmentURL
		}
		if endpoint == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing endpoint URL"})
			return
		}

		body := fmt.Sprintf(`<GetLoginGUID xmlns="https://benefits-selection.com/qx/enrollment"><user>%s</user><passwd>%s</passwd><portfolioID>%s</portfolioID><uniqueID>%s</uniqueID></GetLoginGUID>`,
			xmlEscape(req.Username), xmlEscape(req.Password), xmlEscape(req.PortfolioID), xmlEscape(req.EmployeeID))

		soapRes, err := callSOAP(r.Context(), cfg.Timeout, endpoint, "https://benefits-selection.com/qx/enrollment/GetLoginGUID", body)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}

		value := readSOAPElement(soapRes.Body, "GetLoginGUIDResult")
		if value == "" {
			value = readSOAPFault(soapRes.Body)
		}
		if value == "" && (soapRes.StatusCode < 200 || soapRes.StatusCode >= 300) {
			value = fmt.Sprintf("SOAP HTTP status %d", soapRes.StatusCode)
		}

		writeJSON(w, http.StatusOK, map[string]string{"result": value, "raw": soapRes.Body})
	})

	mux.HandleFunc("/api/saml/preview", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req samlPreviewRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request payload"})
			return
		}

		if strings.TrimSpace(req.AcsURL) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "acsUrl is required"})
			return
		}

		xmlPayload := buildSAMLResponse(req)
		encoded := base64.StdEncoding.EncodeToString([]byte(xmlPayload))

		writeJSON(w, http.StatusOK, map[string]string{
			"xml":        xmlPayload,
			"samlBase64": encoded,
			"postForm":   buildSAMLPostForm(req.AcsURL, encoded),
		})
	})
}

func signalNotifyContext(parent context.Context, signals ...os.Signal) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, signals...)
}

func loadConfig() appConfig {
	timeout := 30 * time.Second
	if raw := os.Getenv("INTEGRATIONSDK_SERVICE_TIMEOUT_SECONDS"); raw != "" {
		if parsed, err := time.ParseDuration(raw + "s"); err == nil && parsed > 0 {
			timeout = parsed
		}
	}

	return appConfig{
		EnrollmentURL: strings.TrimSpace(os.Getenv("INTEGRATIONSDK_ENROLLMENT_URL")),
		Username:      strings.TrimSpace(os.Getenv("INTEGRATIONSDK_SERVICE_USERNAME")),
		Password:      os.Getenv("INTEGRATIONSDK_SERVICE_PASSWORD"),
		Timeout:       timeout,
	}
}

func handleServiceCall(w http.ResponseWriter, r *http.Request, cfg appConfig, operation, resultElement string) {
	var req struct {
		Endpoint string `json:"endpoint"`
		Username string `json:"username"`
		Password string `json:"password"`
		Data     string `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request payload"})
		return
	}

	endpoint := strings.TrimSpace(req.Endpoint)
	if endpoint == "" {
		endpoint = cfg.EnrollmentURL
	}
	if endpoint == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing endpoint URL"})
		return
	}

	body := fmt.Sprintf(`<%s xmlns="https://benefits-selection.com/qx/enrollment"><user>%s</user><passwd>%s</passwd><data>%s</data></%s>`,
		operation,
		xmlEscape(req.Username),
		xmlEscape(req.Password),
		xmlEscape(req.Data),
		operation,
	)

	soapAction := "https://benefits-selection.com/qx/enrollment/" + operation
	soapRes, err := callSOAP(r.Context(), cfg.Timeout, endpoint, soapAction, body)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	value := readSOAPElement(soapRes.Body, resultElement)
	if value == "" {
		value = readSOAPFault(soapRes.Body)
	}
	if value == "" && (soapRes.StatusCode < 200 || soapRes.StatusCode >= 300) {
		value = fmt.Sprintf("SOAP HTTP status %d", soapRes.StatusCode)
	}

	writeJSON(w, http.StatusOK, map[string]string{"result": value, "raw": soapRes.Body})
}

func callSOAP(ctx context.Context, timeout time.Duration, endpoint, action, body string) (soapResponse, error) {
	envelope := `<?xml version="1.0" encoding="utf-8"?>` +
		`<soap:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body>` +
		body +
		`</soap:Body></soap:Envelope>`

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(envelope))
	if err != nil {
		return soapResponse{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", action)

	client := &http.Client{Timeout: timeout}
	res, err := client.Do(req)
	if err != nil {
		return soapResponse{}, fmt.Errorf("call SOAP endpoint: %w", err)
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return soapResponse{}, fmt.Errorf("read SOAP response: %w", err)
	}

	return soapResponse{
		StatusCode: res.StatusCode,
		Body:       string(resBody),
	}, nil
}

func readSOAPElement(raw, element string) string {
	decoder := xml.NewDecoder(strings.NewReader(raw))
	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		start, ok := tok.(xml.StartElement)
		if !ok || !strings.EqualFold(start.Name.Local, element) {
			continue
		}

		var value string
		if err := decoder.DecodeElement(&value, &start); err == nil {
			return value
		}
	}
	return ""
}

func readSOAPFault(raw string) string {
	decoder := xml.NewDecoder(strings.NewReader(raw))
	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		start, ok := tok.(xml.StartElement)
		if !ok || !strings.EqualFold(start.Name.Local, "faultstring") {
			continue
		}

		var value string
		if err := decoder.DecodeElement(&value, &start); err == nil {
			return "SOAP fault: " + value
		}
	}
	return ""
}

func buildSAMLResponse(req samlPreviewRequest) string {
	version := req.Version
	if version == "" {
		version = "2.0"
	}

	attributes := map[string]string{
		"Transmittal": req.TransmittalXML,
	}
	for _, item := range req.LayoutAttributes {
		attributes[item] = "yes"
	}

	var attrBuilder strings.Builder
	for name, value := range attributes {
		attrBuilder.WriteString(`<Attribute Name="` + xmlEscape(name) + `"><AttributeValue>` + xmlEscape(value) + `</AttributeValue></Attribute>`)
	}

	issueInstant := time.Now().UTC().Format(time.RFC3339)
	return `<Response Version="` + xmlEscape(version) + `" IssueInstant="` + xmlEscape(issueInstant) + `" Destination="` + xmlEscape(req.AcsURL) + `">` +
		`<Issuer>Vendor</Issuer>` +
		`<Status><StatusCode Value="Success" /></Status>` +
		`<Assertion><Subject><NameID>` + xmlEscape(req.EmployeeID) + `</NameID></Subject><AttributeStatement>` + attrBuilder.String() + `</AttributeStatement></Assertion>` +
		`</Response>`
}

func buildSAMLPostForm(acsURL, samlBase64 string) string {
	return `<form method="post" action="` + html.EscapeString(acsURL) + `">` +
		`<input type="hidden" name="SAMLResponse" value="` + html.EscapeString(samlBase64) + `" />` +
		`<button type="submit">POST SAMLResponse</button>` +
		`</form>`
}

func xmlEscape(v string) string {
	var buf bytes.Buffer
	xml.EscapeText(&buf, []byte(v))
	return buf.String()
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func uploadGroupTemplate() string {
	return `<Transmittal><Type>UploadPortfolio</Type><Portfolio><Name>Test Group</Name><GroupNumber>TESTXXXX</GroupNumber><EnrollmentStartDate>2010-12-01</EnrollmentStartDate><EnrollmentEndDate>2011-02-15</EnrollmentEndDate><PlanYearStartDate>2011-01-01</PlanYearStartDate><Employer><Name>Test Employer</Name><Address><Line1>123 Main Ln</Line1><City>Chicago</City><State>IL</State><Zip>54342</Zip></Address></Employer></Portfolio></Transmittal>`
}

func uploadCensusTemplate() string {
	return `<Transmittal><Type>UploadApplicants</Type><Group><GroupName>Test Group</GroupName></Group><Applicants><Applicant><Relationship>Employee</Relationship><FirstName>TestFirst</FirstName><LastName>TestLast</LastName><BirthDate>1980-12-24</BirthDate><Sex>Male</Sex><SSN>111-11-1111</SSN></Applicant></Applicants></Transmittal>`
}

func getGroupTemplate() string {
	return `<Transmittal><Type>GetPortfolio</Type><Group><GroupName>Test Group</GroupName></Group></Transmittal>`
}

func getCensusTemplate() string {
	return `<Transmittal><Type>Query</Type><Group><GroupName>Test Group</GroupName></Group><Applicants><Applicant><Relationship>Employee</Relationship><SSN>111-11-1111</SSN></Applicant></Applicants></Transmittal>`
}

func uiFileSystem() (fs.FS, error) {
	sub, err := fs.Sub(embeddedUI, "ui")
	if err != nil {
		return nil, err
	}
	return sub, nil
}

func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}

	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Start()
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}
