package server

import (
	"encoding/base64"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Call the next handler
		next.ServeHTTP(w, r)

		// Log the request
		slog.Debug("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"remoteAddr", r.RemoteAddr,
			"userAgent", r.UserAgent(),
			"duration", time.Since(start),
		)
	})
}

// ----

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
		slog.Error("Cannot conver email ID to delete", "id", id)
		http.Error(w, "Invalid email ID", http.StatusBadRequest)
		return
	}

	s.deleteEmail(id)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) deleteEmail(id int) {
	slog.Debug("Deleting email", "id", id)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.emails, id)
	slog.Info("Deleted email", "id", id)
}

func (s *Server) handleEmail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		slog.Error("Invalid email ID", "id", id)
		http.Error(w, "Invalid email ID", http.StatusBadRequest)
		return
	}

	s.mutex.Lock()
	email, ok := s.emails[id]
	s.mutex.Unlock()

	if !ok {
		slog.Error("Cannot open email", "id", id)
		http.Error(w, "Email not found", http.StatusNotFound)
		return
	}

	if email.IsHTML {
		email.Body = base64.StdEncoding.EncodeToString([]byte(email.Body))
	}

	s.tmpl.ExecuteTemplate(w, "email.html", email)
}
