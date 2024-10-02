package server

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/mail"
	"strconv"
	"sync"
	"time"

	"github.com/emersion/go-smtp"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

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

func (s *Session) AuthPlain(username, password string) error {
	// For simplicity, accept all auth attempts
	return nil
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	s.from = from
	return nil
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	s.to = append(s.to, to)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	_, err := s.data.ReadFrom(r)
	if err != nil {
		return err
	}

	// Once the data is received, save the email
	msg, err := mail.ReadMessage(r)
	if err != nil {
		return err
	}

	subject := msg.Header.Get("Subject")
	body, err := io.ReadAll(msg.Body)
	if err != nil {
		return err
	}

	email := Email{
		From: s.from,
		To:   s.to,
		Subject: subject,
		Body: string(body),
		Date: time.Now(),
	}

	s.server.mutex.Lock()
	defer s.server.mutex.Unlock()
	email.ID = s.server.nextID
	s.server.emails[s.server.nextID] = email
	s.server.nextID++

	fmt.Printf("Received email from: %s to: %v\n", s.from, s.to)

	return nil

}

func (s *Session) Reset() {
	s.from = ""
	s.to = nil
	s.data.Reset()
}

func (s *Session) Logout() error {
	return nil
}


func NewServer() *Server {
	tmpl := template.Must(template.ParseGlob("templates/*.html"))
	return &Server{
		emails: make(map[int]Email),
		tmpl:   tmpl,
	}
}

func (s *Server) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &Session{}, nil
}

func (s *Server) deleteEmail(id int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.emails, id)
}

func (s *Server) handleInbox(w http.ResponseWriter, r *http.Request) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	emails := make([]Email, 0, len(s.emails))
	for _, email := range s.emails {
		emails = append(emails, email)
	}
	s.tmpl.ExecuteTemplate(w, "inbox.html", emails)
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid email ID", http.StatusBadRequest)
		return
	}

	s.deleteEmail(id)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleEmail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid email ID", http.StatusBadRequest)
		return
	}

	s.mutex.Lock()
	email, ok := s.emails[id]
	s.mutex.Unlock()

	if !ok {
		http.Error(w, "Email not found", http.StatusNotFound)
		return
	}

	s.tmpl.ExecuteTemplate(w, "email.html", email)
}

func (s *Server) Rcpt(from string, to string) error {
	return nil
}

func (s *Server) Data(r io.Reader) error {
	return nil
}

func RunServer(cmd *cobra.Command, args []string) {
	server := NewServer()

	// Start SMTP server
	go func() {
		s := smtp.NewServer(server)
		s.Addr = fmt.Sprintf(":%d", viper.GetInt("smtp_port"))
		s.Domain = viper.GetString("domain")
		s.ReadTimeout = 10 * time.Second
		s.WriteTimeout = 10 * time.Second
		s.MaxMessageBytes = 1024 * 1024
		s.MaxRecipients = 50
		s.AllowInsecureAuth = true

		log.Printf("Starting MailMinnow SMTP server at :%d\n", viper.GetInt("smtp_port"))
		if err := s.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	// Start HTTP server
	r := mux.NewRouter()
	r.HandleFunc("/", server.handleInbox).Methods("GET")
	r.HandleFunc("/email/{id}", server.handleEmail).Methods("GET")
	r.HandleFunc("/delete/{id}", server.handleDelete).Methods("POST")

	log.Printf("Starting HTTP server at :%d\n", viper.GetInt("http_port"))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", viper.GetInt("http_port")), r))
}
