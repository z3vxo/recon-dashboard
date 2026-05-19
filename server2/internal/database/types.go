package database

var PortServices = map[int]string{
	21:    "FTP",
	22:    "SSH",
	23:    "Telnet",
	25:    "SMTP",
	53:    "DNS",
	80:    "HTTP",
	110:   "POP3",
	111:   "RPC",
	135:   "MSRPC",
	139:   "NetBIOS",
	143:   "IMAP",
	161:   "SNMP",
	389:   "LDAP",
	443:   "HTTPS",
	445:   "SMB",
	465:   "SMTPS",
	587:   "SMTP",
	636:   "LDAPS",
	993:   "IMAPS",
	995:   "POP3S",
	1433:  "MSSQL",
	1521:  "Oracle",
	2049:  "NFS",
	2083:  "cPanel SSL",
	2087:  "WHM SSL",
	3000:  "Dev/Node",
	3306:  "MySQL",
	3389:  "RDP",
	4443:  "HTTPS-alt",
	5432:  "PostgreSQL",
	5900:  "VNC",
	5984:  "CouchDB",
	6379:  "Redis",
	7443:  "HTTPS-alt",
	8000:  "HTTP-alt",
	8080:  "HTTP-alt",
	8081:  "HTTP-alt",
	8443:  "HTTPS-alt",
	8888:  "HTTP-alt",
	9090:  "HTTP-alt",
	9200:  "Elasticsearch",
	9300:  "Elasticsearch",
	9443:  "HTTPS-alt",
	27017: "MongoDB",
}

type HttpxEntry struct {
	URL         string   `json:"url"`
	StatusCode  int      `json:"status_code"`
	Title       string   `json:"title"`
	WebServer   string   `json:"webserver"`
	Tech        []string `json:"tech"`
	IPs         []string `json:"a"`
	CNAME       []string `json:"cname"`
	ContentType string   `json:"content_type"`
	OpenPorts   []int    `json:"open_ports"`
}

type Port struct {
	Port    string `json:"port"`
	Service string `json:"service"`
}

type Host struct {
	ID           int
	HostID       string
	DomainName   string
	StatusCode   string
	OpenPorts    string
	Title        string
	TechStack    string
	ContentType  string
	Server       string
	IPs          string
	CNAME        string
	Badges       string
	TriageStatus string
	Notes        string
}

type HostResponse struct {
	ID           int      `json:"id"`
	HostID       string   `json:"host_id"`
	DomainName   string   `json:"url"`
	SC           string   `json:"sc"`
	StatusCode   string   `json:"status"`
	OpenPorts    []Port   `json:"ports"`
	Title        string   `json:"title"`
	TechStack    []string `json:"tech"`
	ContentType  string   `json:"ctype"`
	Server       string   `json:"server"`
	IPs          []string `json:"ips"`
	CNAME        []string `json:"cname"`
	Badges       []string `json:"badges"`
	TriageStatus string   `json:"triage_status"`
	Notes        string   `json:"notes"`
}

type Stats struct {
	Total int `json:"total"`
	S200  int `json:"s200"`
	S403  int `json:"s403"`
	S500  int `json:"s500"`
}

type HostsResult struct {
	Stats Stats         `json:"stats"`
	Hosts []HostResponse `json:"hosts"`
}

type HitResponse struct {
	URL        string `json:"url"`
	StatusCode string `json:"status"`
	SC         string `json:"sc"`
	Size       string `json:"size"`
	Severity   string `json:"severity"`
}

type CountEntry struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type SummaryData struct {
	TotalHosts     int          `json:"total_hosts"`
	UniqueIPs      int          `json:"unique_ips"`
	TriageReviewed int          `json:"triage_reviewed"`
	JuicyHits      int          `json:"juicy_hits"`
	StatusCodes    []CountEntry `json:"status_codes"`
	TechStack      []CountEntry `json:"tech_stack"`
	TopCnames      []CountEntry `json:"top_cnames"`
	TopIPs         []CountEntry `json:"top_ips"`
	Badges         []CountEntry `json:"badges"`
}
