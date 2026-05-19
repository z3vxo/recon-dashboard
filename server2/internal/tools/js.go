package tools

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/z3vxo/vantage/internal/database"
)

func extractHostname(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return u.Hostname(), nil
}

func appendURLs(path string, data []byte) int {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return 0
	}
	defer f.Close()
	count := 0
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
			f.WriteString(line + "\n")
			count++
		}
	}
	return count
}

func runStdout(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), err
}

func runGau(finalURLs, hostURL string) {
	domain, err := extractHostname(hostURL)
	if err != nil {
		slog.Error("js: gau failed to extract hostname", "host", hostURL, "err", err)
		return
	}
	res, err := runStdout("gau", domain)
	if err != nil {
		slog.Error("js: gau failed", "host", domain, "err", err)
		return
	}
	n := appendURLs(finalURLs, res)
	slog.Debug("js: gau done", "host", domain, "urls", n)
}

func runWayback(finalURLs, hostURL string) {
	domain, err := extractHostname(hostURL)
	if err != nil {
		slog.Error("js: waybackurls failed to extract hostname", "host", hostURL, "err", err)
		return
	}
	res, err := runStdout("waybackurls", domain)
	if err != nil {
		slog.Error("js: waybackurls failed", "host", domain, "err", err)
		return
	}
	n := appendURLs(finalURLs, res)
	slog.Debug("js: waybackurls done", "host", domain, "urls", n)
}

func runKatana(finalURLs, hostURL string, headless bool) {
	args := []string{"-u", hostURL, "-d", "2", "-jc"}
	if headless {
		args = append(args, "-hl", "-nos")
	}
	out, err := runStdout("katana", args...)
	if err != nil {
		slog.Error("js: katana failed", "host", hostURL, "err", err)
		return
	}
	n := appendURLs(finalURLs, out)
	slog.Debug("js: katana done", "host", hostURL, "urls", n)
}

func extractJsURLs(finalURLs, hostURL, jsURLsPath string) error {
	hostname, err := extractHostname(hostURL)
	if err != nil {
		return err
	}
	cmd := fmt.Sprintf("grep -iE '\\.js(\\?|$)' %s | grep '%s' | sort -u > %s", finalURLs, hostname, jsURLsPath)
	if out, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
			return fmt.Errorf("js filter failed: %w — %s", err, string(out))
		}
	}
	return nil
}

func ScrapeJsFiles(hostURL, domain, jsDir, urlsPath string) error {
	if err := os.MkdirAll(jsDir, 0755); err != nil {
		return fmt.Errorf("failed to create js dir: %w", err)
	}

	info, err := os.Stat(urlsPath)
	if os.IsNotExist(err) || (err == nil && info.Size() == 0) {
		slog.Info("js: no JS URLs found, skipping download", "host", hostURL)
		return nil
	}

	cmd := exec.Command("httpx",
		"-l", urlsPath,
		"-sr",
		"-srd", jsDir,
		"-mc", "200",
		"-silent",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("httpx scrape failed: %w — %s", err, string(out))
	}

	return nil
}

// runLinkFinder runs linkfinder on a single file and returns discovered URLs
func runLinkFinder(filePath string) []string {
	out, err := exec.Command("python3", os.Getenv("HOME")+"/tools/linkFinder/linkfinder.py", "-i", filePath, "-o", "cli").CombinedOutput()
	if err != nil {
		return nil
	}
	var links []string
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			links = append(links, line)
		}
	}
	return links
}

// isNoise returns true for SecretsFinder hits that are clearly false positives:
// minified JS code, placeholder strings, or values that are too long/complex to be real credentials
func isNoise(value string) bool {
	// Too long to be a real secret (minified code)
	if len(value) > 200 {
		return true
	}
	// Contains JS syntax — it's code, not a credential
	noisePatterns := []string{
		"this.", "function", "return ", ".get", ".set",
		"null,", "null}", "undefined", "=>", "(){", "()",
		"wrapMethods", "wrapStruct", "gateMethod",
	}
	lower := strings.ToLower(value)
	for _, p := range noisePatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

// runSecretsFinder runs SecretFinder on a single file and returns secrets
// Output format: "Type    ->    value"
func runSecretsFinder(filePath string) []database.JsSecret {
	out, err := exec.Command("python3", os.Getenv("HOME")+"/tools/secretFinder/SecretFinder.py", "-i", filePath, "-o", "cli").CombinedOutput()
	if err != nil {
		return nil
	}
	var secrets []database.JsSecret
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		parts := strings.SplitN(line, "->", 2)
		if len(parts) != 2 {
			continue
		}
		secretType := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if secretType == "" || value == "" {
			continue
		}
		// if isNoise(value) {
		// 	continue
		// }
		secrets = append(secrets, database.JsSecret{
			File:  filePath,
			Type:  secretType,
			Value: value,
		})
	}
	return secrets
}

// runTruffleHog runs trufflehog on the JS directory and returns findings
type truffleHogResult struct {
	SourceMetadata struct {
		Data struct {
			Filesystem struct {
				File string `json:"file"`
			} `json:"Filesystem"`
		} `json:"Data"`
	} `json:"SourceMetadata"`
	DetectorName string `json:"DetectorName"`
	Raw          string `json:"Raw"`
}

func runTruffleHog(jsDir string) []database.JsSecret {
	out, err := exec.Command("trufflehog", "filesystem", jsDir, "--json", "--no-verification").CombinedOutput()
	if err != nil {
		return nil
	}
	var secrets []database.JsSecret
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Bytes()
		var r truffleHogResult
		if err := json.Unmarshal(line, &r); err != nil {
			continue
		}
		if r.Raw == "" {
			continue
		}
		secrets = append(secrets, database.JsSecret{
			File:  r.SourceMetadata.Data.Filesystem.File,
			Type:  r.DetectorName,
			Value: r.Raw,
		})
	}
	return secrets
}

func analyzeJsFiles(jsDir, domain, hostURL string) error {
	// httpx saves files into response/<hostname>/filename — walk the full tree
	var files []string
	err := filepath.Walk(jsDir+"/response", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil || len(files) == 0 {
		return nil
	}

	var mu sync.Mutex
	var allSecrets []database.JsSecret
	var allLinks []database.JsLink
	var wg sync.WaitGroup

	sem := make(chan struct{}, 30)

	for _, f := range files {
		wg.Add(2)
		go func(filePath string) {
			defer wg.Done()
			sem <- struct{}{}
			links := runLinkFinder(filePath)
			<-sem
			mu.Lock()
			for _, l := range links {
				allLinks = append(allLinks, database.JsLink{File: filePath, URL: l})
			}
			mu.Unlock()
		}(f)
		go func(filePath string) {
			defer wg.Done()
			sem <- struct{}{}
			secrets := runSecretsFinder(filePath)
			<-sem
			mu.Lock()
			allSecrets = append(allSecrets, secrets...)
			mu.Unlock()
		}(f)
	}
	wg.Wait()

	// TruffleHog on the whole dir as a second pass
	thSecrets := runTruffleHog(jsDir + "/response")
	allSecrets = append(allSecrets, thSecrets...)

	return database.SaveJsResults(domain, hostURL, allSecrets, allLinks)
}

func ScrapeAndScan(host, id, domain string, headless bool) {
	SetJob(id, JobResult{Status: JobPending})

	home, _ := os.UserHomeDir()
	jsDir := fmt.Sprintf("%s/.recon/%s/%s/js", home, domain, SanitizeForFilename(host))
	finalURLs := jsDir + "/final_urls.txt"
	urlsPath := jsDir + "/js_urls.txt"

	if err := os.MkdirAll(jsDir, 0755); err != nil {
		SetJob(id, JobResult{Status: JobFailed, Error: err.Error()})
		return
	}

	// clear previous run
	os.Remove(finalURLs)

	var wg sync.WaitGroup
	wg.Add(3)
	go func() { defer wg.Done(); runGau(finalURLs, host) }()
	go func() { defer wg.Done(); runWayback(finalURLs, host) }()
	go func() { defer wg.Done(); runKatana(finalURLs, host, headless) }()
	wg.Wait()

	if info, err := os.Stat(finalURLs); err == nil {
		slog.Info("js: all URLs saved", "host", host, "final_urls_bytes", info.Size())
	} else {
		slog.Warn("js: no URLs found", "host", host)
	}

	if err := extractJsURLs(finalURLs, host, urlsPath); err != nil {
		SetJob(id, JobResult{Status: JobFailed, Error: err.Error()})
		return
	}

	if info, err := os.Stat(urlsPath); err == nil {
		slog.Info("js: JS URLs filtered", "host", host, "js_list_bytes", info.Size())
	} else {
		slog.Warn("js: no JS URLs found after filter", "host", host)
	}

	if err := ScrapeJsFiles(host, domain, jsDir, urlsPath); err != nil {
		SetJob(id, JobResult{Status: JobFailed, Error: err.Error()})
		return
	}

	if entries, err := os.ReadDir(jsDir + "/response"); err == nil {
		slog.Info("js: files downloaded", "host", host, "count", len(entries))
	} else {
		slog.Warn("js: no files downloaded", "host", host, "err", err)
	}

	if err := analyzeJsFiles(jsDir, domain, host); err != nil {
		SetJob(id, JobResult{Status: JobFailed, Error: err.Error()})
		return
	}

	os.RemoveAll(jsDir + "/response")
	SetJob(id, JobResult{Status: JobDone})
}
