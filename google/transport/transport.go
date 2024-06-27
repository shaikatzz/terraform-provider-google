// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0
package transport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"google.golang.org/api/googleapi"
)

var DefaultRequestTimeout = 5 * time.Minute

type SendRequestOptions struct {
	Config                      *Config
	Method                      string
	Project                     string
	RawURL                      string
	UserAgent                   string
	Body                        map[string]any
	Timeout                     time.Duration
	Headers                     http.Header
	ErrorRetryPredicates        []RetryErrorPredicateFunc
	ErrorAbortPredicates        []RetryErrorPredicateFunc
	ErrorRetryBackoffPredicates []RetryErrorPredicateFunc
}

func wrapErrorRetryBackoffPredicates(fs []RetryErrorPredicateFunc) []RetryErrorPredicateFunc {
	if fs == nil {
		return fs
	}
	wrappedFuncs := make([]RetryErrorPredicateFunc, len(fs))
	for _, f := range fs {

		// Each function is wrapped with a closure with its own backoff struct
		funcToWrap := f
		backoff := struct {
			attempts       int64
			lastSleep      int64
			minimumBackoff time.Duration
			maximumBackoff time.Duration
		}{
			minimumBackoff: time.Duration(200),      // 200 ns
			maximumBackoff: time.Duration(60 * 1e9), // 60 seconds
		}

		var wf RetryErrorPredicateFunc = func(err error) (bool, string) {
			// Reuse backoff struct via closure
			b := &backoff

			isRetryable, msg := funcToWrap(err)
			if isRetryable {
				log.Printf("[DEBUG] Retryable error with backoff starting")

				// Sleep for period based on number of attempts so far
				// https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/
				// sleep = random_between(0, min(upperBound, base * 2 ** attempt))
				lowerBound := b.minimumBackoff
				upperBound := int64(math.Min(float64(b.maximumBackoff.Nanoseconds()), float64(b.lastSleep*int64(2)^b.attempts)))

				r := rand.New(rand.NewSource(time.Now().UnixNano()))
				sleep := r.Int63n((upperBound - lowerBound.Nanoseconds() + 1) + lowerBound.Nanoseconds())

				time.Sleep(time.Duration(sleep))
				switch {
				case time.Duration(sleep).Seconds() >= 1:
					log.Printf("[DEBUG] Slept for %s second(s)", time.Duration(sleep).Seconds())
				case time.Duration(sleep).Milliseconds() >= 1:
					log.Printf("[DEBUG] Slept for %s milliseconds(s)", time.Duration(sleep).Milliseconds())
				case time.Duration(sleep).Microseconds() >= 1:
					log.Printf("[DEBUG] Slept for %s microseconds(s)", time.Duration(sleep).Microseconds())
				default:
					log.Printf("[DEBUG] Slept for %s nanosecond(s)", time.Duration(sleep).Nanoseconds())
				}

				// Update backoff struct for next time
				b.attempts += 1
				b.lastSleep = sleep
			}
			return isRetryable, msg
		}
		wrappedFuncs = append(wrappedFuncs, wf)
	}
	return wrappedFuncs
}

func SendRequest(opt SendRequestOptions) (map[string]interface{}, error) {
	reqHeaders := opt.Headers
	if reqHeaders == nil {
		reqHeaders = make(http.Header)
	}
	reqHeaders.Set("User-Agent", opt.UserAgent)
	reqHeaders.Set("Content-Type", "application/json")

	if opt.Config.UserProjectOverride && opt.Project != "" {
		// When opt.Project is "NO_BILLING_PROJECT_OVERRIDE" in the function GetCurrentUserEmail,
		// set the header X-Goog-User-Project to be empty string.
		if opt.Project == "NO_BILLING_PROJECT_OVERRIDE" {
			reqHeaders.Set("X-Goog-User-Project", "")
		} else {
			// Pass the project into this fn instead of parsing it from the URL because
			// both project names and URLs can have colons in them.
			reqHeaders.Set("X-Goog-User-Project", opt.Project)
		}
	}

	if opt.Timeout == 0 {
		opt.Timeout = DefaultRequestTimeout
	}

	var res *http.Response
	err := Retry(RetryOptions{
		RetryFunc: func() error {
			var buf bytes.Buffer
			if opt.Body != nil {
				err := json.NewEncoder(&buf).Encode(opt.Body)
				if err != nil {
					return err
				}
			}

			u, err := AddQueryParams(opt.RawURL, map[string]string{"alt": "json"})
			if err != nil {
				return err
			}
			req, err := http.NewRequest(opt.Method, u, &buf)
			if err != nil {
				return err
			}

			req.Header = reqHeaders
			res, err = opt.Config.Client.Do(req)
			if err != nil {
				return err
			}

			if err := googleapi.CheckResponse(res); err != nil {
				googleapi.CloseBody(res)
				return err
			}

			return nil
		},
		Timeout:                     opt.Timeout,
		ErrorRetryPredicates:        opt.ErrorRetryPredicates,
		ErrorAbortPredicates:        opt.ErrorAbortPredicates,
		ErrorRetryBackoffPredicates: opt.ErrorRetryBackoffPredicates,
	})
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, fmt.Errorf("Unable to parse server response. This is most likely a terraform problem, please file a bug at https://github.com/hashicorp/terraform-provider-google/issues.")
	}

	// The defer call must be made outside of the retryFunc otherwise it's closed too soon.
	defer googleapi.CloseBody(res)

	// 204 responses will have no body, so we're going to error with "EOF" if we
	// try to parse it. Instead, we can just return nil.
	if res.StatusCode == 204 {
		return nil, nil
	}
	result := make(map[string]interface{})
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func AddQueryParams(rawurl string, params map[string]string) (string, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return "", err
	}
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func AddArrayQueryParams(rawurl string, param string, values []interface{}) (string, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return "", err
	}
	q := u.Query()
	for _, v := range values {
		q.Add(param, v.(string))
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func HandleNotFoundError(err error, d *schema.ResourceData, resource string) error {
	if IsGoogleApiErrorWithCode(err, 404) {
		log.Printf("[WARN] Removing %s because it's gone", resource)
		// The resource doesn't exist anymore
		d.SetId("")

		return nil
	}

	return errwrap.Wrapf(
		fmt.Sprintf("Error when reading or editing %s: {{err}}", resource), err)
}

func HandleDataSourceNotFoundError(err error, d *schema.ResourceData, resource, url string) error {
	if IsGoogleApiErrorWithCode(err, 404) {
		return fmt.Errorf("%s not found", url)
	}

	return errwrap.Wrapf(
		fmt.Sprintf("Error when reading or editing %s: {{err}}", resource), err)
}

func IsGoogleApiErrorWithCode(err error, errCode int) bool {
	gerr, ok := errwrap.GetType(err, &googleapi.Error{}).(*googleapi.Error)
	return ok && gerr != nil && gerr.Code == errCode
}

func IsApiNotEnabledError(err error) bool {
	gerr, ok := errwrap.GetType(err, &googleapi.Error{}).(*googleapi.Error)
	if !ok {
		return false
	}
	if gerr == nil {
		return false
	}
	if gerr.Code != 403 {
		return false
	}
	for _, e := range gerr.Errors {
		if e.Reason == "accessNotConfigured" {
			return true
		}
	}
	return false
}
