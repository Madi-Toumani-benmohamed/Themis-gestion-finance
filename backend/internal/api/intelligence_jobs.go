package api

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"themis/internal/intelligence"
)

// ScheduleIntelligenceRecompute is best-effort by design: a job outage must
// never roll back a successful financial mutation.
func (s *Server) ScheduleIntelligenceRecompute(owner int64, trigger string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := intelligence.Enqueue(ctx, s.DB, owner, trigger); err != nil {
		log.Printf("intelligence job enqueue failed (trigger=%s)", trigger)
	}
}

func (s *Server) IntelligenceJobStatus(w http.ResponseWriter, r *http.Request) {
	status, err := intelligence.GetJobStatus(r.Context(), s.DB, s.ownerID(r))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

// StartIntelligenceWorker starts one poller. PostgreSQL SKIP LOCKED and the
// per-owner advisory lock make multiple API replicas safe.
func (s *Server) StartIntelligenceWorker(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			if ctx.Err() != nil {
				return
			}
			job, err := intelligence.Claim(ctx, s.DB)
			if err != nil {
				log.Printf("intelligence job claim failed")
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					continue
				}
			}
			if job == nil {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					continue
				}
			}
			s.processIntelligenceJob(ctx, *job)
		}
	}()
	return done
}

func (s *Server) processIntelligenceJob(parent context.Context, job intelligence.Job) {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	heartbeatDone := make(chan struct{})
	go func() {
		defer close(heartbeatDone)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				heartbeatCtx, stop := context.WithTimeout(context.Background(), 3*time.Second)
				err := intelligence.RenewJob(heartbeatCtx, s.DB, job.OwnerID)
				stop()
				if err != nil {
					log.Printf("intelligence job heartbeat failed")
				}
			}
		}
	}()
	err := s.runIntelligenceJob(ctx, job.OwnerID)
	cancel()
	<-heartbeatDone
	finishCtx, finishCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer finishCancel()
	if err != nil {
		_ = intelligence.FailJob(finishCtx, s.DB, job.OwnerID, job.Attempts)
		log.Printf("intelligence recalculation job failed (trigger=%s)", job.Trigger)
		return
	}
	if err = intelligence.CompleteJob(finishCtx, s.DB, job.OwnerID); err != nil {
		log.Printf("intelligence job completion update failed")
	}
}

// Reuse the exact authenticated handler path so asynchronous and manual runs
// share validation, locking, detectors and registry reconciliation.
func (s *Server) runIntelligenceJob(ctx context.Context, owner int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "/api/intel/recompute", strings.NewReader(`{}`))
	if err != nil {
		return err
	}
	req = req.WithContext(context.WithValue(req.Context(), ctxUserKey, &authUser{ID: owner}))
	response := httptest.NewRecorder()
	s.recomputeIntelligence(response, req)
	if response.Code < 200 || response.Code >= 300 {
		return errors.New("intelligence recalculation failed")
	}
	return nil
}

type statusCaptureWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusCaptureWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
		w.ResponseWriter.WriteHeader(status)
	}
}
func (w *statusCaptureWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}

func intelligenceMutationTrigger(r *http.Request) string {
	p := r.URL.Path
	switch {
	case r.Method == http.MethodPost && p == "/api/statements":
		return "statement_import"
	case r.Method == http.MethodPost && strings.HasPrefix(p, "/api/statements/previews/") && strings.HasSuffix(p, "/confirm"):
		return "statement_import"
	case r.Method == http.MethodPatch && strings.HasPrefix(p, "/api/transactions/"):
		return "transaction_update"
	case r.Method == http.MethodPatch && strings.HasPrefix(p, "/api/statements/") && strings.HasSuffix(p, "/validation"):
		return "statement_validation"
	case r.Method == http.MethodDelete && strings.HasPrefix(p, "/api/statements/") && !strings.HasPrefix(p, "/api/statements/previews/"):
		return "statement_delete"
	case r.Method == http.MethodPut && strings.HasPrefix(p, "/api/recurring/review/"):
		return "recurrence_review"
	case r.Method == http.MethodPost && strings.HasPrefix(p, "/api/intel/discoveries/") && strings.HasSuffix(p, "/feedback"):
		return "discovery_feedback"
	case strings.HasPrefix(p, "/api/merchants/aliases") && (r.Method == http.MethodPost || r.Method == http.MethodDelete):
		return "merchant_alias"
	case strings.HasPrefix(p, "/api/rules") && (r.Method == http.MethodPost || r.Method == http.MethodPatch || r.Method == http.MethodDelete):
		return "categorization_rule"
	case strings.HasPrefix(p, "/api/domains") && (r.Method == http.MethodPost || r.Method == http.MethodPatch || r.Method == http.MethodDelete):
		return "category_change"
	case strings.HasPrefix(p, "/api/tags/") && (r.Method == http.MethodPatch || r.Method == http.MethodDelete):
		return "tag_change"
	case strings.HasPrefix(p, "/api/accounts") && (r.Method == http.MethodPost || r.Method == http.MethodPatch || r.Method == http.MethodDelete):
		return "account_coverage"
	case strings.HasPrefix(p, "/api/reimbursements") && (r.Method == http.MethodPost || r.Method == http.MethodDelete):
		return "reimbursement_change"
	case p == "/api/admin/reapply-rules" && r.Method == http.MethodPost:
		return "rules_reapplied"
	default:
		return ""
	}
}

func (s *Server) intelligenceEnqueueMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		trigger := intelligenceMutationTrigger(r)
		if trigger == "" {
			next.ServeHTTP(w, r)
			return
		}
		capture := &statusCaptureWriter{ResponseWriter: w}
		next.ServeHTTP(capture, r)
		if capture.status >= 200 && capture.status < 300 {
			if owner := s.ownerID(r); owner > 0 {
				s.ScheduleIntelligenceRecompute(owner, trigger)
			}
		}
	})
}
