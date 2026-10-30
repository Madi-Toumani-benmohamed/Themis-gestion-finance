package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"themis/internal/money"
	"time"
)

type tagFolder struct {
	Name          string      `json:"name"`
	Total         json.Number `json:"total"`
	Count         int         `json:"count"`
	FirstDate     string      `json:"first_date"`
	LastDate      string      `json:"last_date"`
	Days          int         `json:"days"`
	IsProject     bool        `json:"is_project"`
	Pinned        bool        `json:"pinned"`
	Hidden        bool        `json:"hidden"`
	ShowAsFolder  bool        `json:"show_as_folder"`
	SelectedTotal json.Number `json:"selected_month_total"`
}

func folderWindow(r *http.Request) (month, from, to string, err error) {
	month = r.URL.Query().Get("month")
	first, last, _, err := reliabilityWindow(month, time.Now())
	return first.Format("2006-01"), first.Format("2006-01-02"), last.Format("2006-01-02"), err
}
func (s *Server) tagFolders(w http.ResponseWriter, r *http.Request) {
	month, first, last, err := folderWindow(r)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	prefs := map[string]string{}
	prows, err := s.DB.QueryContext(r.Context(), `SELECT tag_name,state FROM tag_folder_prefs WHERE owner_id=$1`, s.ownerID(r))
	if err != nil {
		writeErr(w, 500, fmt.Errorf("préférences de projets indisponibles"))
		return
	}
	for prows.Next() {
		var name, state string
		if err = prows.Scan(&name, &state); err != nil {
			prows.Close()
			writeErr(w, 500, err)
			return
		}
		prefs[strings.ToLower(name)] = state
	}
	err = prows.Err()
	prows.Close()
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	rows, err := s.DB.QueryContext(r.Context(), `WITH tag_tx AS (
 SELECT DISTINCT tg.name,t.id,t.tx_date,-t.amount AS amt FROM tags tg JOIN transaction_tags tt ON tt.tag_id=tg.id
 JOIN transactions t ON t.id=tt.transaction_id AND t.owner_id=tg.owner_id
 WHERE t.owner_id=$1 AND t.amount<0 AND t.tx_date<=$3::date`+notTransferT+`)
 SELECT name,SUM(amt)::text,count(*),min(tx_date)::text,max(tx_date)::text,
 COALESCE(SUM(amt) FILTER(WHERE tx_date>=$2::date),0)::text,
 (SELECT max(n) FROM (SELECT count(DISTINCT b.id) AS n FROM (SELECT DISTINCT tx_date FROM tag_tx WHERE name=outer_tx.name) a
 JOIN tag_tx b ON b.name=outer_tx.name AND b.tx_date>=a.tx_date AND b.tx_date<a.tx_date+INTERVAL '2 months' GROUP BY a.tx_date) counts)
 FROM tag_tx outer_tx GROUP BY name ORDER BY SUM(amt) DESC,name`, s.ownerID(r), first, last)
	if err != nil {
		writeErr(w, 500, fmt.Errorf("projets par tag indisponibles"))
		return
	}
	defer rows.Close()
	folders, others := []tagFolder{}, []tagFolder{}
	for rows.Next() {
		var f tagFolder
		var raw, selected string
		var maxCount int
		if err = rows.Scan(&f.Name, &raw, &f.Count, &f.FirstDate, &f.LastDate, &selected, &maxCount); err != nil {
			writeErr(w, 500, err)
			return
		}
		total, e := parseAnalysisAmount(raw)
		if e != nil {
			writeErr(w, 500, e)
			return
		}
		selectedTotal, e := parseAnalysisAmount(selected)
		if e != nil {
			writeErr(w, 500, e)
			return
		}
		f.Total = simulationNumber(total)
		f.SelectedTotal = simulationNumber(selectedTotal)
		f.Days = daysBetween(f.FirstDate, f.LastDate)
		f.IsProject = total >= 15000 && f.Count >= 2 && maxCount*100 >= f.Count*70
		f.Pinned = prefs[strings.ToLower(f.Name)] == "pinned"
		f.Hidden = prefs[strings.ToLower(f.Name)] == "hidden"
		f.ShowAsFolder = (f.IsProject || f.Pinned) && !f.Hidden
		if f.ShowAsFolder {
			folders = append(folders, f)
		} else {
			others = append(others, f)
		}
	}
	if rows.Err() != nil {
		writeErr(w, 500, fmt.Errorf("projets interrompus"))
		return
	}
	writeJSON(w, 200, map[string]any{"folders": folders, "others": others, "month": month, "query_to": last, "scope": "history_through_selected_month"})
}
func (s *Server) tagFolderDetail(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	if name == "" {
		writeErr(w, 400, fmt.Errorf("tag requis"))
		return
	}
	month, _, last, err := folderWindow(r)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	// One grouped query yields all detail totals from the same statement snapshot.
	rows, err := s.DB.QueryContext(r.Context(), `SELECT COALESCE(p.name,d.name,'Sans catégorie'),
 COALESCE(NULLIF(t.merchant,''),NULLIF(t.display_label,''),t.label),(-SUM(t.amount))::text,count(*),min(t.tx_date)::text,max(t.tx_date)::text,SUM(-SUM(t.amount)) OVER()::text
 FROM transactions t LEFT JOIN domains d ON d.id=t.domain_id AND d.owner_id=t.owner_id
 LEFT JOIN domains p ON p.id=d.parent_id AND p.owner_id=t.owner_id
 WHERE t.owner_id=$1 AND t.amount<0 AND t.tx_date<=$3::date`+notTransferT+`
 AND EXISTS(SELECT 1 FROM transaction_tags tt JOIN tags tg ON tg.id=tt.tag_id AND tg.owner_id=t.owner_id WHERE tt.transaction_id=t.id AND tg.name=$2)
 GROUP BY 1,2 ORDER BY 1,2`, s.ownerID(r), name, last)
	if err != nil {
		writeErr(w, 500, fmt.Errorf("détail du projet indisponible"))
		return
	}
	defer rows.Close()
	byCat, byMerchant := map[string]int64{}, map[string]int64{}
	var total int64
	var count int
	firstDate, lastDate := "", ""
	for rows.Next() {
		var category, merchant, raw, firstD, lastD, fullRaw string
		var n int
		if err = rows.Scan(&category, &merchant, &raw, &n, &firstD, &lastD, &fullRaw); err != nil {
			writeErr(w, 500, err)
			return
		}
		if _, e := parseAnalysisAmount(fullRaw); e != nil {
			writeErr(w, 500, e)
			return
		}
		amount, e := parseAnalysisAmount(raw)
		if e != nil {
			writeErr(w, 500, e)
			return
		}
		byCat[category] += int64(amount)
		byMerchant[merchant] += int64(amount)
		total += int64(amount)
		count += n
		if firstDate == "" || firstD < firstDate {
			firstDate = firstD
		}
		if lastD > lastDate {
			lastDate = lastD
		}
	}
	if rows.Err() != nil {
		writeErr(w, 500, fmt.Errorf("détail interrompu"))
		return
	}
	cats, merchants := []map[string]any{}, []map[string]any{}
	for key, value := range byCat {
		cats = append(cats, map[string]any{"category": key, "total": simulationNumber(money.Cents(value))})
	}
	for key, value := range byMerchant {
		merchants = append(merchants, map[string]any{"name": key, "total": simulationNumber(money.Cents(value))})
	}
	writeJSON(w, 200, map[string]any{"name": name, "total": simulationNumber(money.Cents(total)), "count": count, "first_date": firstDate, "last_date": lastDate, "days": daysBetween(firstDate, lastDate), "by_category": cats, "by_merchant": merchants, "month": month, "query_to": last, "scope": "history_through_selected_month"})
}
func (s *Server) setTagFolderPref(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)
	name := strings.TrimSpace(r.PathValue("name"))
	if name == "" {
		writeErr(w, 400, fmt.Errorf("tag requis"))
		return
	}
	var in struct {
		State string `json:"state"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in); err != nil {
		writeErr(w, 400, fmt.Errorf("préférence invalide"))
		return
	}
	var err error
	switch in.State {
	case "auto":
		_, err = s.DB.ExecContext(r.Context(), `DELETE FROM tag_folder_prefs WHERE owner_id=$1 AND tag_name=$2`, owner, name)
	case "pinned", "hidden":
		var exists bool
		if err = s.DB.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM tags WHERE owner_id=$1 AND name=$2)`, owner, name).Scan(&exists); err != nil {
			writeErr(w, 500, fmt.Errorf("tag indisponible"))
			return
		}
		if !exists {
			writeErr(w, 404, fmt.Errorf("tag introuvable"))
			return
		}
		_, err = s.DB.ExecContext(r.Context(), `INSERT INTO tag_folder_prefs(owner_id,tag_name,state) VALUES($1,$2,$3) ON CONFLICT(owner_id,tag_name) DO UPDATE SET state=EXCLUDED.state`, owner, name, in.State)
	default:
		writeErr(w, 400, fmt.Errorf("state doit être pinned, hidden ou auto"))
		return
	}
	if err != nil {
		writeErr(w, 500, fmt.Errorf("préférence non enregistrée"))
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}
func daysBetween(a, b string) int {
	ta, e1 := time.Parse("2006-01-02", a)
	tb, e2 := time.Parse("2006-01-02", b)
	if e1 != nil || e2 != nil {
		return 0
	}
	return int(tb.Sub(ta).Hours()/24) + 1
}
