package database

import (
	"database/sql"
	"strings"
)

type AgentQuery struct {
	Type   string
	Limit  int
	Offset int
	Status string
}

type AgentSummaryEntry struct {
	HostID string `json:"host_id"`
	URL    string `json:"url"`
	Status string `json:"status"`
}

type AgentFullEntry struct {
	HostID       string   `json:"host_id"`
	URL          string   `json:"url"`
	Status       string   `json:"status"`
	Title        string   `json:"title"`
	Server       string   `json:"server"`
	TechStack    []string `json:"tech"`
	OpenPorts    []Port   `json:"ports"`
	IPs          []string `json:"ips"`
	CNAME        []string `json:"cname"`
	ContentType  string   `json:"ctype"`
	Badges       []string `json:"badges"`
	TriageStatus string   `json:"triage_status"`
	Notes        string   `json:"notes"`
}

type AgentResponse struct {
	Total   int `json:"total"`
	Limit   int `json:"limit"`
	Offset  int `json:"offset"`
	Results any `json:"results"`
}

func AgentData(domain string, q AgentQuery) (AgentResponse, error) {
	db, err := getDB(domain)
	if err != nil {
		return AgentResponse{}, err
	}

	where, args := buildWhere(q.Status)

	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM domains d"+where, args...).Scan(&total); err != nil {
		return AgentResponse{}, err
	}

	switch q.Type {
	case "summary":
		return querySummary(db, where, args, q, total)
	default:
		return queryFull(db, where, args, q, total)
	}
}

func buildWhere(status string) (string, []any) {
	if status == "" {
		return "", nil
	}
	return " WHERE d.status_code = ?", []any{status}
}

func querySummary(db *sql.DB, where string, args []any, q AgentQuery, total int) (AgentResponse, error) {
	query := "SELECT d.host_id, d.domain_name, d.status_code FROM domains d" + where +
		" ORDER BY d.domain_name LIMIT ? OFFSET ?"
	args = append(args, q.Limit, q.Offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return AgentResponse{}, err
	}
	defer rows.Close()

	var results []AgentSummaryEntry
	for rows.Next() {
		var e AgentSummaryEntry
		if err := rows.Scan(&e.HostID, &e.URL, &e.Status); err != nil {
			return AgentResponse{}, err
		}
		results = append(results, e)
	}
	if results == nil {
		results = []AgentSummaryEntry{}
	}

	return AgentResponse{Total: total, Limit: q.Limit, Offset: q.Offset, Results: results}, rows.Err()
}

func queryFull(db *sql.DB, where string, args []any, q AgentQuery, total int) (AgentResponse, error) {
	query := `SELECT d.id, d.host_id, d.domain_name, d.status_code, d.open_ports, d.title,
			COALESCE((SELECT GROUP_CONCAT(tech, ', ') FROM domain_tech   WHERE domain_id = d.id), '') AS tech_stack,
			d.content_type, d.server,
			COALESCE((SELECT GROUP_CONCAT(ip,   ', ') FROM domain_ips    WHERE domain_id = d.id), '') AS ips,
			COALESCE((SELECT GROUP_CONCAT(cname,', ') FROM domain_cnames WHERE domain_id = d.id), '') AS cnames,
			COALESCE((SELECT GROUP_CONCAT(badge,', ') FROM domain_badges WHERE domain_id = d.id), '') AS badges,
			d.triage_status, d.notes
		FROM domains d` + where + " ORDER BY d.domain_name LIMIT ? OFFSET ?"
	args = append(args, q.Limit, q.Offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return AgentResponse{}, err
	}
	defer rows.Close()

	var results []AgentFullEntry
	for rows.Next() {
		var h Host
		if err := rows.Scan(
			&h.ID, &h.HostID, &h.DomainName, &h.StatusCode, &h.OpenPorts,
			&h.Title, &h.TechStack, &h.ContentType, &h.Server,
			&h.IPs, &h.CNAME, &h.Badges, &h.TriageStatus, &h.Notes,
		); err != nil {
			return AgentResponse{}, err
		}
		results = append(results, toAgentFull(h))
	}
	if results == nil {
		results = []AgentFullEntry{}
	}

	return AgentResponse{Total: total, Limit: q.Limit, Offset: q.Offset, Results: results}, rows.Err()
}

func toAgentFull(h Host) AgentFullEntry {
	tr := transformHost(h)
	tech := tr.TechStack
	if tech == nil {
		tech = []string{}
	}
	// strip empty strings from split results
	ips := filterEmpty(strings.Split(h.IPs, ","))
	cname := filterEmpty(strings.Split(h.CNAME, ","))
	badges := filterEmpty(strings.Split(h.Badges, ","))

	return AgentFullEntry{
		HostID:       h.HostID,
		URL:          h.DomainName,
		Status:       h.StatusCode,
		Title:        h.Title,
		Server:       h.Server,
		TechStack:    tech,
		OpenPorts:    tr.OpenPorts,
		IPs:          ips,
		CNAME:        cname,
		ContentType:  h.ContentType,
		Badges:       badges,
		TriageStatus: h.TriageStatus,
		Notes:        h.Notes,
	}
}

func filterEmpty(s []string) []string {
	var out []string
	for _, v := range s {
		if t := strings.TrimSpace(v); t != "" {
			out = append(out, t)
		}
	}
	if out == nil {
		out = []string{}
	}
	return out
}
