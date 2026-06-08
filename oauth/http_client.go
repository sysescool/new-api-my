package oauth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/QuantumNous/new-api/common"
	"golang.org/x/net/proxy"
)

const oauthProxyEnv = "OAUTH_PROXY"

func oauthProxyEnvName(provider string) string {
	var builder strings.Builder
	lastWasUnderscore := true
	for _, r := range provider {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(unicode.ToUpper(r))
			lastWasUnderscore = false
			continue
		}
		if !lastWasUnderscore {
			builder.WriteByte('_')
			lastWasUnderscore = true
		}
	}
	prefix := strings.Trim(builder.String(), "_")
	if prefix == "" {
		return ""
	}
	return prefix + "_OAUTH_PROXY"
}

func lookupProxyEnv(name string) string {
	if name == "" {
		return ""
	}
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return strings.TrimSpace(os.Getenv(strings.ToLower(name)))
}

func oauthProxyURL(providerProxyEnv string) string {
	if value := lookupProxyEnv(providerProxyEnv); value != "" {
		return value
	}
	return lookupProxyEnv(oauthProxyEnv)
}

func newOAuthHTTPClient(providerProxyEnv string, timeout time.Duration) (*http.Client, error) {
	proxyURL := oauthProxyURL(providerProxyEnv)
	if proxyURL == "" {
		// Fall back to Go's standard HTTP_PROXY, HTTPS_PROXY and NO_PROXY handling.
		return &http.Client{Timeout: timeout}, nil
	}
	return newOAuthProxyHTTPClient(proxyURL, timeout)
}

func newOAuthProxyHTTPClient(proxyURL string, timeout time.Duration) (*http.Client, error) {
	parsedURL, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{
		MaxIdleConns:        common.RelayMaxIdleConns,
		MaxIdleConnsPerHost: common.RelayMaxIdleConnsPerHost,
		IdleConnTimeout:     time.Duration(common.RelayIdleConnTimeout) * time.Second,
		ForceAttemptHTTP2:   true,
	}
	if common.TLSInsecureSkipVerify {
		transport.TLSClientConfig = common.InsecureTLSConfig
	}

	switch parsedURL.Scheme {
	case "http", "https":
		transport.Proxy = http.ProxyURL(parsedURL)
	case "socks5", "socks5h":
		var auth *proxy.Auth
		if parsedURL.User != nil {
			auth = &proxy.Auth{
				User:     parsedURL.User.Username(),
				Password: "",
			}
			if password, ok := parsedURL.User.Password(); ok {
				auth.Password = password
			}
		}
		dialer, err := proxy.SOCKS5("tcp", parsedURL.Host, auth, proxy.Direct)
		if err != nil {
			return nil, err
		}
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		}
	default:
		return nil, fmt.Errorf("unsupported OAuth proxy scheme: %s, must be http, https, socks5 or socks5h", parsedURL.Scheme)
	}

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}, nil
}
