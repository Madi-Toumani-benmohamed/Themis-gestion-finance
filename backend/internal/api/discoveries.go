package api

import (
	"fmt"
	"net/http"
	"time"
)

var serviceSegments = map[string][]string{
	"vidéo à la demande": {"Netflix", "Disney+", "Canal+", "Amazon Prime", "OCS", "Paramount", "Apple TV"},
	"musique":            {"Spotify", "Deezer", "Apple", "YouTube"},
	"stockage cloud":     {"Google", "Dropbox", "iCloud", "Microsoft"},
}

func (s *Server) discoveries(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	month, err := s.analysisMonth(r.Context(), s.ownerID(r), r.URL.Query().Get("month"), now)
	if err != nil {
		analysisError(w, r, err)
		return
	}
	report, err := s.loadRecurringReview(r.Context(), s.ownerID(r), month, now)
	if err != nil {
		writeErr(w, 500, fmt.Errorf("repérages indisponibles"))
		return
	}
	out := analysisRecurringSignals(report)
	signals, limitations, _, err := s.loadAnalysisSignals(r.Context(), s.ownerID(r), month, now)
	if err != nil {
		writeErr(w, 500, fmt.Errorf("signaux analytiques indisponibles"))
		return
	}
	out = append(out, signals...)
	writeJSON(w, 200, map[string]any{"month": month, "period": report.Period, "query_from": analysisSignalStart(month), "query_to": report.ObservationEnd, "discoveries": out, "limitations": limitations})
}
