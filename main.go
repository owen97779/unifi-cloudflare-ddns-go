package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

var logger = log.New(os.Stdout, "cloudflare-dns-updater: ", log.LstdFlags)

var (
	CLOUDFLARE_API_ENDPOINT  = os.Getenv("CLOUDFLARE_API_ENDPOINT")
	CLOUDFLARE_API_KEY       = os.Getenv("CLOUDFLARE_API_KEY")
	CLOUDFLARE_EMAIL         = os.Getenv("CLOUDFLARE_EMAIL")
	CLOUDFLARE_ZONE_ID       = os.Getenv("CLOUDFLARE_ZONE_ID")
	CLOUDFLARE_DNS_NAME      = os.Getenv("CLOUDFLARE_DNS_NAME")
	CLOUDFLARE_DNS_RECORD_ID = os.Getenv("CLOUDFLARE_DNS_RECORD_ID")
)

type Record struct {
	Comment string `json:"comment"`
	Content string `json:"content"`
	Name    string `json:"name"`
	Proxied bool   `json:"proxied"`
	TTL     int    `json:"ttl"`
	Type    string `json:"type"`
}

func update_dns_record(client *http.Client, ip net.IP) error {
	record, err := json.Marshal(Record{
		Comment: "updated by unifi-cloudflare-ddns-go",
		Content: ip.String(),
		Name:    CLOUDFLARE_DNS_NAME,
		Proxied: false,
		TTL:     300,
		Type:    "A",
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(
		context.TODO(),
		http.MethodPut,
		strings.Join([]string{
			CLOUDFLARE_API_ENDPOINT,
			"zones",
			CLOUDFLARE_ZONE_ID,
			"dns_records",
			CLOUDFLARE_DNS_RECORD_ID}, "/"),
		strings.NewReader(string(record)))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+CLOUDFLARE_API_KEY)
	req.Header.Set("X-Auth-Email", CLOUDFLARE_EMAIL)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cloudflare API error: %d - %s", resp.StatusCode, string(body))
	}
	return nil
}

func main() {
	client := &http.Client{}

	http.HandleFunc("/update", func(w http.ResponseWriter, r *http.Request) {
		ipString := r.URL.Query().Get("ip")
		if ipString == "" {
			logger.Println("ip query param is missing")
			return
		}
		ip := net.ParseIP(ipString)
		if ip == nil {
			logger.Println("invalid ip address format", ip)
			return
		}
		hostname := r.URL.Query().Get("hostname")
		if hostname == "" {
			logger.Println("hostname query param is missing")
			return
		}

		err := update_dns_record(client, ip)
		if err != nil {
			logger.Println("error updating dns record:", err)
			return
		}
		logger.Printf("dns record updated successfully for hostname=%s, ip=%s", hostname, ip)
	})

	http.ListenAndServe(":8080", nil)
}
