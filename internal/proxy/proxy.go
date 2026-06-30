package proxy

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jhanvi857/vexor/internal/config"
)

func NewProxy(target string) (*httputil.ReverseProxy, error) {
	parsedURL, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	proxy := httputil.NewSingleHostReverseProxy(parsedURL)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		clientIP := clientIPFromRequest(req)
		req.Header.Del("X-Forwarded-For")
		req.Header.Del("X-Real-IP")
		originalDirector(req)
		if clientIP != "" {
			req.Header.Set("X-Real-IP", clientIP)
		}
	}
	proxy.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return proxy, nil
}

func clientIPFromRequest(req *http.Request) string {
	remoteAddr := strings.TrimSpace(req.RemoteAddr)
	if remoteAddr == "" {
		return ""
	}

	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil && host != "" {
		return host
	}

	return remoteAddr
}

var parsedTrustedIPs []net.IP
var parsedTrustedSubnets []*net.IPNet
var parseOnce sync.Once

func initTrustedProxies() {
	for _, cidrOrIP := range config.TrustedProxies {
		cidrOrIP = strings.TrimSpace(cidrOrIP)
		if cidrOrIP == "" {
			continue
		}
		if _, ipnet, err := net.ParseCIDR(cidrOrIP); err == nil {
			parsedTrustedSubnets = append(parsedTrustedSubnets, ipnet)
		} else if ip := net.ParseIP(cidrOrIP); ip != nil {
			parsedTrustedIPs = append(parsedTrustedIPs, ip)
		}
	}
}

func IsTrustedProxy(remoteAddr string) bool {
	if len(config.TrustedProxies) == 0 {
		return true
	}
	parseOnce.Do(initTrustedProxies)

	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(strings.TrimSpace(host))
	if ip == nil {
		return false
	}

	for _, tip := range parsedTrustedIPs {
		if tip.Equal(ip) {
			return true
		}
	}
	for _, subnet := range parsedTrustedSubnets {
		if subnet.Contains(ip) {
			return true
		}
	}
	return false
}
