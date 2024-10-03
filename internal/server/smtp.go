package server

import (
	"io"
	"log/slog"
	"net/mail"

	"time"

	"github.com/emersion/go-smtp"
)

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
		From:    s.from,
		To:      s.to,
		Subject: subject,
		Body:    string(body),
		Date:    time.Now(),
	}

	s.server.mutex.Lock()
	defer s.server.mutex.Unlock()
	email.ID = s.server.nextID
	s.server.emails[s.server.nextID] = email
	s.server.nextID++

	slog.Info("Received email", "from", s.from, "to", s.to)

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

func (s *Server) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &Session{}, nil
}

func (s *Server) deleteEmail(id int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.emails, id)
}

func (s *Server) Rcpt(from string, to string) error {
	return nil
}

func (s *Server) Data(r io.Reader) error {
	return nil
}
