package server

import (
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
)

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
