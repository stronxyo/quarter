package webapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// A request can fail in three ways that say nothing about the game asked
// for, so none of them is remembered: no answer at all (ErrUnreachable),
// too many requests (*RateLimitError), or a failing server (ErrServer).

// ErrUnreachable means no answer came: no network, the name didn't
// resolve, the connection failed, or it timed out.
var ErrUnreachable = errors.New("no answer")

// ErrServer means the server answered with an error of its own.
var ErrServer = errors.New("server error")

// RateLimitError means the server asked for fewer requests (HTTP 429, or the
// 403 the Steam store also sends). RetryAfter is how long it asked to wait,
// zero when it didn't say.
type RateLimitError struct {
	Status     int
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("rate limited (HTTP %d), retry after %v", e.Status, e.RetryAfter)
	}
	return fmt.Sprintf("rate limited (HTTP %d)", e.Status)
}

// Unreachable marks a request that got no answer. A cancelled request is
// returned as it is: that is the caller's choice, not a failure.
func Unreachable(what string, err error) error {
	if errors.Is(err, context.Canceled) {
		return err
	}
	return fmt.Errorf("%s: %w: %w", what, ErrUnreachable, err)
}

// Refused classifies an answer that wasn't a success: a rate limit, or a
// server error. limitCodes are the statuses that mean "too many requests".
func Refused(what string, resp *http.Response, now time.Time, limitCodes ...int) error {
	for _, code := range limitCodes {
		if resp.StatusCode == code {
			return fmt.Errorf("%s: %w", what, &RateLimitError{Status: code, RetryAfter: retryAfter(resp.Header.Get("Retry-After"), now)})
		}
	}
	return fmt.Errorf("%s: HTTP %d: %w", what, resp.StatusCode, ErrServer)
}

// retryAfter reads a Retry-After header: seconds, or an HTTP date (RFC 9110
// section 10.2.3). Anything else, or a time already past, is zero.
func retryAfter(h string, now time.Time) time.Duration {
	h = strings.TrimSpace(h)
	if h == "" {
		return 0
	}
	if s, err := strconv.Atoi(h); err == nil {
		return max(time.Duration(s)*time.Second, 0)
	}
	if t, err := http.ParseTime(h); err == nil {
		return max(t.Sub(now), 0)
	}
	return 0
}
