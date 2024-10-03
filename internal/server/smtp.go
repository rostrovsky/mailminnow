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

func (s *Server) NewSession(c *smtp.Conn) (smtp.Session, error) {
	slog.Debug("SMTP new session")
	return &Session{
		server: s,
	}, nil
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	slog.Debug("SMTP", "cmd", "MAIL FROM", "from", from)
	s.from = from
	return nil
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	slog.Debug("SMTP", "cmd", "RCPT TO", "to", to)
	s.to = append(s.to, to)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	slog.Debug("SMTP", "cmd", "DATA")
	_, err := s.data.ReadFrom(r)
	if err != nil {
		slog.Error("SMTP error reading data", "error", err)
		return err
	}

	// Once the data is received, save the email
	slog.Debug("SMTP - saving email")
	msg, err := mail.ReadMessage(&s.data)
	if err != nil {
		slog.Error("SMTP error reading message", "error", err)
		return err
	}

	subject := msg.Header.Get("Subject")
	body, err := io.ReadAll(msg.Body)
	if err != nil {
		slog.Error("SMTP error reading body", "error", err)
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
	slog.Debug("SMTP saved email",
		"from", email.From,
		"to", email.To,
		"subject", subject,
		"length", len(email.Body),
		"date", email.Date,
		"id", email.ID)
	slog.Info("SMTP received email", "from", email.From, "to", email.To, "id", email.ID)

	return nil
}

func (s *Session) Reset() {
	slog.Debug("SMTP", "cmd", "RSET")
	s.from = ""
	s.to = nil
	s.data.Reset()
}

func (s *Session) Logout() error {
	return nil
}
