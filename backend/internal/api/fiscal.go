package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"themis/internal/fiscal"
)

func (s *Server) fiscalRules(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, fiscal.Available())
}
func (s *Server) fiscalSimulate(w http.ResponseWriter, r *http.Request) {
	var in fiscal.Input
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&in) != nil || decoder.Decode(new(any)) != io.EOF {
		writeErr(w, 400, errors.New("saisie fiscale invalide"))
		return
	}
	out, err := fiscal.Compute(in)
	if err != nil {
		status := 400
		var e *fiscal.Error
		if errors.As(err, &e) && e.Code == "unsupported" {
			status = 422
		}
		writeErr(w, status, err)
		return
	}
	writeJSON(w, 200, out)
}
