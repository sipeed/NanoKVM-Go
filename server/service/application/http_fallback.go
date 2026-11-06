package application

import (
	"fmt"
	"net/http"
)

// doApplicationRequestWithHTTPFallback keeps HTTPS as the preferred
// transport. If the HTTPS request itself fails, retry the same request once
// with only the URL scheme changed to HTTP.
func doApplicationRequestWithHTTPFallback(
	client *http.Client,
	request *http.Request,
) (*http.Response, error) {
	response, err := client.Do(request)
	if err == nil || request.URL == nil || request.URL.Scheme != "https" {
		return response, err
	}
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}

	fallbackRequest := request.Clone(request.Context())
	fallbackURL := *request.URL
	fallbackURL.Scheme = "http"
	fallbackRequest.URL = &fallbackURL

	response, fallbackErr := client.Do(fallbackRequest)
	if fallbackErr != nil {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		return nil, fmt.Errorf(
			"HTTPS request failed (%v); HTTP fallback failed: %w",
			err,
			fallbackErr,
		)
	}
	return response, nil
}
