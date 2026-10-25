package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"themis/internal/money"
)

const engagementMaxCents money.Cents = 9007199254740991
const engagementMaxMonths = 1200

func engagementNumber(c money.Cents) json.Number { return json.Number(c.String()) }
func engagementAmount(raw string, positive bool) (money.Cents, error) {
	amount, err := parseTransactionAggregate(raw)
	if err != nil || amount < 0 || amount > engagementMaxCents || (positive && amount == 0) {
		return 0, fmt.Errorf("montant positif au centime requis, dans les limites exactes")
	}
	return amount, nil
}
func engagementAdd(a, b money.Cents) (money.Cents, error) {
	if a < 0 || b < 0 || a > engagementMaxCents-b {
		return 0, fmt.Errorf("somme monétaire hors limites")
	}
	return a + b, nil
}
func engagementProduct(amount money.Cents, count int) (money.Cents, error) {
	if amount <= 0 || count <= 0 || count > engagementMaxMonths || amount > engagementMaxCents/money.Cents(count) {
		return 0, fmt.Errorf("échéancier invalide ou hors limites")
	}
	return amount * money.Cents(count), nil
}
func engagementMonth(raw string) (time.Time, error) {
	value, err := time.Parse("2006-01", raw)
	if err != nil || value.Year() < 1 || value.Format("2006-01") != raw {
		return value, fmt.Errorf("mois valide YYYY-MM requis")
	}
	return value, nil
}
func engagementDate(raw string) (time.Time, error) {
	value, err := time.Parse("2006-01-02", raw)
	if err != nil || value.Year() < 1 || value.Format("2006-01-02") != raw {
		return value, fmt.Errorf("date valide YYYY-MM-DD requise")
	}
	return value, nil
}
func engagementToday(now time.Time) time.Time {
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}
func engagementPathID(r *http.Request, key string) (int64, error) {
	value, err := strconv.ParseInt(r.PathValue(key), 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("identifiant invalide")
	}
	return value, nil
}
func decodeEngagement(w http.ResponseWriter, r *http.Request, target any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384))
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		return fmt.Errorf("corps JSON invalide")
	}
	if dec.Decode(new(any)) != io.EOF {
		return fmt.Errorf("un seul objet JSON est attendu")
	}
	return nil
}
func engagementOptionalAmount(raw json.RawMessage) (*money.Cents, error) {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}
	var number json.Number
	if err := json.Unmarshal(raw, &number); err != nil {
		return nil, fmt.Errorf("montant décimal invalide")
	}
	amount, err := engagementAmount(string(number), false)
	if err != nil {
		return nil, err
	}
	return &amount, nil
}
func engagementSubmission(key string) error {
	if len(key) < 16 || len(key) > 100 {
		return fmt.Errorf("identifiant de soumission requis")
	}
	for _, ch := range key {
		if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '-' || ch == '_') {
			return fmt.Errorf("identifiant de soumission invalide")
		}
	}
	return nil
}
func engagementNote(raw string) (string, error) {
	note := strings.TrimSpace(raw)
	if len(note) > 2000 {
		return "", fmt.Errorf("note trop longue")
	}
	return note, nil
}

func engagementParseCents(raw string) (int64, error) {
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 || value > int64(engagementMaxCents) {
		return 0, fmt.Errorf("centimes hors limites exactes")
	}
	return value, nil
}

// Euro decimals preserve the historical API. Integer-cent companions are the
// authoritative browser representation, including unusually large totals.
func engagementCentsFields(fields map[string]*json.Number) map[string]*int64 {
	out := make(map[string]*int64, len(fields))
	for key, number := range fields {
		out[key] = nil
		if number == nil {
			continue
		}
		value, err := engagementAmount(string(*number), false)
		if err != nil {
			continue
		}
		n := int64(value)
		out[key] = &n
	}
	return out
}

// Bootstrap middleware intentionally permits the rest of the legacy API before
// account setup. Financial engagements always require a positive authenticated
// owner, including first launch and a failed user-count query.
func (s *Server) engagementAuthenticated(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.ownerID(r) <= 0 {
			writeErr(w, http.StatusUnauthorized, fmt.Errorf("authentification requise"))
			return
		}
		next(w, r)
	}
}
