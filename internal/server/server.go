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
	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
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

	slog.Info("Server configuration",
		"smtpPort", smtpPort,
		"httpPort", httpPort,
		"domain", domain)

	// Start SMTP server
	go func() {
		s := smtp.NewServer(server)
		s.Addr = fmt.Sprintf(":%d", smtpPort)
		s.Domain = domain
		s.ReadTimeout = 10 * time.Second
		s.WriteTimeout = 10 * time.Second
		s.MaxMessageBytes = 1024 * 1024
		s.MaxRecipients = 50
		s.AllowInsecureAuth = true

		slog.Info("Starting SMTP server", "port", smtpPort)
		if err := s.ListenAndServe(); err != nil {
			slog.Error("SMTP server error", "error", err)
			os.Exit(1)
		}
	}()

	// Start HTTP server
	r := mux.NewRouter()
	r.HandleFunc("/", server.handleInbox).Methods("GET")
	r.HandleFunc("/email/{id}", server.handleEmail).Methods("GET")
	r.HandleFunc("/delete/{id}", server.handleDelete).Methods("POST")

	slog.Info("Starting HTTP server", "port", httpPort)
	err := http.ListenAndServe(fmt.Sprintf(":%d", httpPort), r)
	if err != nil {
		slog.Error("HTTP server error", "error", err)
		os.Exit(1)
	}
}
