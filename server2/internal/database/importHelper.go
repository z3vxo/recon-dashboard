package database

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func joinInts(ports []int) string {
	parts := make([]string, len(ports))
	for i, p := range ports {
		parts[i] = strconv.Itoa(p)
	}
	return strings.Join(parts, ", ")
}

func computeBadges(url, title string, techStack []string) string {
	combined := strings.ToLower(url + " " + title)
	seen := make(map[string]bool)

	badgeRules := []struct {
		badge    string
		patterns []string
	}{
		{"auth", []string{"login", "signin", "auth", "oauth", "sso", "saml", "cas", "keycloak", "okta", "auth0", "clerk", "register", "signup"}},
		{"admin", []string{"admin", "wp-admin", "dashboard", "portal", "cpanel", "manager", "console"}},
		{"api", []string{"api", "swagger", "openapi", "graphql", "/rest/", "rest/", "v1/", "v2/", "v3/", "/graphiql"}},
		{"dev", []string{"debug", "/dev/", "-dev.", ".dev.", "sandbox", "uat", ".env", "trace", "prepod", "ppr", "prprod", "devtools"}},
		{"cicd", []string{"jenkins", "gitlab", "gitea", "drone", "argo", "circleci", "/code/", "deploy", "pipeline"}},
		{"monitoring", []string{"grafana", "kibana", "prometheus", "nagios", "zabbix", "sonarqube", "sentry", "datadog", "newrelic", "uptimerobot"}},
		{"docs", []string{"docs", "documentation", "readme", "changelog", "/status", "/health", "/info", "/version", "sitemap", "robots.txt"}},
		{"storage", []string{"upload", "s3", "bucket", "zapier", "backup", "download", "export", "cdn", "minio"}},
		{"cms", []string{"wordpress", "drupal", "wp-content", "ghost", "webflow", "joomla"}},
		{"collab", []string{"jira", "confluence", "slack", "trello", "notion", "basecamp", "asana"}},
	}

	for _, rule := range badgeRules {
		for _, p := range rule.patterns {
			if strings.Contains(combined, p) {
				seen[rule.badge] = true
			}
		}
	}

	for _, p := range []string{"welcome to nginx", "apache2 default", "iis windows server", "default website", "it works!", "welcome to apache"} {
		if strings.Contains(strings.ToLower(title), p) {
			seen["default"] = true
		}
	}

	interestingTech := map[string]string{
		"jenkins": "cicd", "gitlab": "cicd", "gitea": "cicd",
		"grafana": "monitoring", "kibana": "monitoring", "prometheus": "monitoring",
		"sentry": "monitoring", "sonarqube": "monitoring",
		"phpmyadmin": "admin", "adminer": "admin",
		"tomcat": "interesting", "weblogic": "interesting", "elasticsearch": "interesting",
		"redis": "database", "mongodb": "database", "couchdb": "database",
		"rabbitmq": "interesting", "solr": "interesting",
		"wordpress": "cms", "drupal": "cms",
	}

	for _, tech := range techStack {
		if badge, ok := interestingTech[strings.ToLower(tech)]; ok {
			seen[badge] = true
		}
	}

	badges := make([]string, 0, len(seen))
	for b := range seen {
		badges = append(badges, b)
	}
	sort.Strings(badges)

	return strings.Join(badges, ",")

	// interestingPatterns := []string{"login", "admin", "dashboard", "portal", "jenkins", "grafana", "kibana", "gitlab", "jira", "confluence", "phpmyadmin", "cpanel", "wp-admin"}
	// for _, p := range interestingPatterns {
	// 	if strings.Contains(combined, p) {
	// 		badges = append(badges, "interesting")
	// 		break
	// 	}
	// }

	// apiPatterns := []string{"api", "swagger", "openapi", "graphql"}
	// for _, p := range apiPatterns {
	// 	if strings.Contains(combined, p) {
	// 		badges = append(badges, "api")
	// 		break
	// 	}
	// }

	// return strings.Join(badges, ",")
}

func severityFromStatus(status string) string {
	switch {
	case strings.HasPrefix(status, "2"):
		return "high"
	case status == "401" || status == "403":
		return "medium"
	case strings.HasPrefix(status, "5"):
		return "medium"
	default:
		return "low"
	}
}

func ImportPathHits(domain string) error {
	db, err := getDB(domain)
	if err != nil {
		return fmt.Errorf("failed to get db: %w", err)
	}

	home, _ := os.UserHomeDir()
	fullPath := filepath.Join(home, ".recon", domain, "probe", "httpx", domain+"_path_hits.txt")
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil // no hits file is fine
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return fmt.Errorf("failed opening path hits file: %w", err)
	}
	defer file.Close()

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`INSERT INTO juicy_hits (url, status_code, size, severity)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(url) DO UPDATE SET
			status_code = excluded.status_code,
			size        = excluded.size,
			severity    = excluded.severity`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			continue
		}
		url, status, size := parts[0], parts[1], parts[2]
		severity := severityFromStatus(status)

		if _, err := stmt.Exec(url, status, size, severity); err != nil {
			tx.Rollback()
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func ImportHttpx(domain string) error {
	db, err := getDB(domain)
	if err != nil {
		return fmt.Errorf("failed to get db: %w", err)
	}
	home, _ := os.UserHomeDir()
	fullPath := filepath.Join(home, ".recon", domain, "probe", "httpx", domain+"_httpx_enriched.json")

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return errors.New("file does not exist")
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return errors.New("failed opening file")
	}
	defer file.Close()

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	stmtDomain, err := tx.Prepare(`
		INSERT INTO domains (domain_name, status_code, title, server, content_type, open_ports, host_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(domain_name) DO UPDATE SET
			status_code  = excluded.status_code,
			title        = excluded.title,
			server       = excluded.server,
			content_type = excluded.content_type,
			open_ports   = excluded.open_ports,
			host_id      = CASE WHEN domains.host_id = '' OR domains.host_id IS NULL THEN excluded.host_id ELSE domains.host_id END`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmtDomain.Close()

	stmtGetID, err := tx.Prepare(`SELECT id FROM domains WHERE domain_name = ?`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmtGetID.Close()

	type juncStmt struct{ del, ins *sql.Stmt }
	makeJunc := func(table, col string) (juncStmt, error) {
		del, err := tx.Prepare(fmt.Sprintf(`DELETE FROM %s WHERE domain_id = ?`, table))
		if err != nil {
			return juncStmt{}, err
		}
		ins, err := tx.Prepare(fmt.Sprintf(`INSERT OR IGNORE INTO %s (domain_id, %s) VALUES (?, ?)`, table, col))
		if err != nil {
			return juncStmt{}, err
		}
		return juncStmt{del, ins}, nil
	}

	jIPs, err := makeJunc("domain_ips", "ip")
	if err != nil { tx.Rollback(); return err }
	defer jIPs.del.Close()
	defer jIPs.ins.Close()

	jCnames, err := makeJunc("domain_cnames", "cname")
	if err != nil { tx.Rollback(); return err }
	defer jCnames.del.Close()
	defer jCnames.ins.Close()

	jTech, err := makeJunc("domain_tech", "tech")
	if err != nil { tx.Rollback(); return err }
	defer jTech.del.Close()
	defer jTech.ins.Close()

	jBadges, err := makeJunc("domain_badges", "badge")
	if err != nil { tx.Rollback(); return err }
	defer jBadges.del.Close()
	defer jBadges.ins.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var entry HttpxEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}

		result, err := stmtDomain.Exec(entry.URL, entry.StatusCode, entry.Title, entry.WebServer, entry.ContentType, joinInts(entry.OpenPorts), GenerateHostID())
		if err != nil {
			tx.Rollback()
			fmt.Println("failed to insert domain:", err)
			return err
		}

		var domainID int64
		if id, err := result.LastInsertId(); err == nil && id > 0 {
			domainID = id
		} else {
			stmtGetID.QueryRow(entry.URL).Scan(&domainID)
		}

		jIPs.del.Exec(domainID)
		for _, v := range entry.IPs {
			jIPs.ins.Exec(domainID, strings.TrimSpace(v))
		}

		jCnames.del.Exec(domainID)
		for _, v := range entry.CNAME {
			jCnames.ins.Exec(domainID, strings.TrimSpace(v))
		}

		jTech.del.Exec(domainID)
		for _, v := range entry.Tech {
			jTech.ins.Exec(domainID, strings.TrimSpace(v))
		}

		badges := computeBadges(entry.URL, entry.Title, entry.Tech)
		jBadges.del.Exec(domainID)
		for _, v := range splitTrim(badges) {
			jBadges.ins.Exec(domainID, v)
		}
	}
	if err := scanner.Err(); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()

}
