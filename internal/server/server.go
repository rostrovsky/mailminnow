package server

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/emersion/go-smtp"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//go:embed templates/*.html
var templateFS embed.FS

type Email struct {
	ID      int
	From    string
	To      []string
	Subject string
	Body    string
	Date    time.Time
	IsHTML  bool
}

type Server struct {
	emails map[int]Email
	nextID int
	mutex  sync.Mutex
	tmpl   *template.Template
}

type Session struct {
	server *Server
	from   string
	to     []string
	data   bytes.Buffer
}

//  --------------------------

func NewServer() *Server {
	slog.Info("Creating server...")

	tmpl := template.New("")
	tmpl = tmpl.Funcs(template.FuncMap{
		"safeHTML": func(html string) template.HTML {
			return template.HTML(html)
		},
	})

	tmpl, err := tmpl.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		slog.Error("Error parsing embedded templates", "error", err)
		os.Exit(1)
	}

	return &Server{
		emails: make(map[int]Email),
		tmpl:   tmpl,
	}
}

func RunServer(cmd *cobra.Command, args []string) {
	server := NewServer()

	smtpPort := viper.GetInt("smtp_port")
	httpPort := viper.GetInt("http_port")
	domain := viper.GetString("domain")

	logLevel := "INFO"
	if viper.GetBool("verbose") {
		logLevel = "DEBUG"
	}

	slog.Info("Server configuration",
		"smtpPort", smtpPort,
		"httpPort", httpPort,
		"domain", domain,
		"logLevel", logLevel)

	// Start SMTP server
	smtpStarted := make(chan bool, 1)
	go func() {
		s := smtp.NewServer(server)
		s.Addr = fmt.Sprintf(":%d", smtpPort)
		s.Domain = domain
		s.ReadTimeout = 10 * time.Second
		s.WriteTimeout = 10 * time.Second
		s.MaxMessageBytes = 1024 * 1024
		s.MaxRecipients = 50
		s.AllowInsecureAuth = true

		slog.Info("Starting SMTP server...", "port", smtpPort)
		smtpStarted <- true
		if err := s.ListenAndServe(); err != nil {
			slog.Error("SMTP server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for SMTP server to start
	<-smtpStarted
	slog.Info("SMTP server started successfully", "port", smtpPort)

	// Start HTTP server
	r := mux.NewRouter()
	r.HandleFunc("/", server.handleInbox).Methods("GET")
	r.HandleFunc("/email/{id}", server.handleEmail).Methods("GET")
	r.HandleFunc("/delete/{id}", server.handleDelete).Methods("POST")

	loggingRouter := LoggingMiddleware(r)

	slog.Info("Starting HTTP server...", "port", httpPort)
	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", httpPort), loggingRouter)
		if err != nil {
			slog.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("HTTP server started successfully", "port", httpPort)
	slog.Info("MailMinnow is now running. Press Ctrl+C to stop.")

	select {}
}
