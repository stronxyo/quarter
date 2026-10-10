package webapi

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

// Some games give the store a short link as their website (Once Human's is
// https://t.co/gRX2BR4tj9). Quarter shows and opens the official site
// instead (Pablo, 2026-10-02: "use the official site domain names"): it
// asks the short-link service where the link goes and stops there, so the
// game's site itself is never visited. A short link that can't be followed
// (offline, dead, a loop) leaves no website: a short link is never shown.

// shortHosts are the link-shortening services followed, by host name.
var shortHosts = map[string]bool{
	"t.co": true, "bit.ly": true, "bitly.com": true, "j.mp": true, "goo.gl": true,
	"tinyurl.com": true, "ow.ly": true, "buff.ly": true, "is.gd": true, "v.gd": true,
	"rebrand.ly": true, "cutt.ly": true, "shorturl.at": true, "rb.gy": true,
	"tiny.cc": true, "t.ly": true, "s.id": true, "lnkd.in": true, "dlvr.it": true,
	"ift.tt": true, "fb.me": true, "amzn.to": true,
}

// maxShortHops is how many short links in a row are followed.
const maxShortHops = 5

// isShortLink reports whether link is on a link-shortening service.
func isShortLink(link string) bool {
	u, err := url.Parse(link)
	return err == nil && shortHosts[strings.ToLower(u.Hostname())]
}

// unshorten is where a short link leads: the first address that isn't
// another short link, as a full http or https link, or "" if it can't be
// told. Only the short-link services are asked (HEAD, no page downloaded).
func (c *StoreClient) unshorten(ctx context.Context, link string) string {
	client := &http.Client{
		Timeout: c.http.Timeout,
		CheckRedirect: func(next *http.Request, via []*http.Request) error {
			if len(via) >= maxShortHops || !isShortLink(next.URL.String()) {
				return http.ErrUseLastResponse // stop before visiting it
			}
			return nil
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, link, nil)
	if err != nil {
		return ""
	}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	resp.Body.Close()
	to, err := resp.Location()
	if err != nil || isShortLink(to.String()) {
		return "" // no redirect, or still a short link after maxShortHops
	}
	return website(to.String())
}
