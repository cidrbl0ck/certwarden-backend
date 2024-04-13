package app

import (
	"certwarden-backend/pkg/output"
	"net/http"

	"github.com/rs/cors"
)

// middlewareApplyCORS applies the CORS package which manages all CORS headers.
// if no cross origins are permitted, this function is a no-op and just returns next
func middlewareApplyCORS(next handlerFunc, permittedCrossOrigins []string) handlerFunc {
	// are any cross origins allowed? if not, do not use CORS
	if permittedCrossOrigins == nil {
		return next
	}

	// set up CORS
	c := cors.New(cors.Options{
		// permitted cross origins
		// WARNING: nil / empty slice == allow all!
		AllowedOrigins: permittedCrossOrigins,

		// credentials must be allowed for access to work properly
		AllowCredentials: true,

		// allowed request headers (client can send to server)
		AllowedHeaders: []string{
			// general
			"content-type",

			// access token
			"authorization",

			// pem download authentication
			"X-API-Key", "apiKey",

			// conditionals for pem downloads
			"if-match", "if-modified-since", "if-none-match", "if-range", "if-unmodified-since",

			// retry tracker for refresh token logic on frontend
			"x-no-retry",
		},

		// allowed methods the client can send to the server
		AllowedMethods: []string{http.MethodDelete, http.MethodGet, http.MethodHead,
			http.MethodPost, http.MethodPut},

		// headers for client to expose to the cross origin requester (in server response)
		ExposedHeaders: []string{
			// general
			"content-length", "content-security-policy", "content-type", "referrer-policy",
			"strict-transport-security", "vary", "x-content-type-options", "x-frame-options",

			// set name of file when client downloads something (used with pem, zip)
			"content-disposition",

			// conditionals for pem downloads
			"last-modified", "etag",
		},
	})

	// return custom handlerFunc
	return func(w http.ResponseWriter, r *http.Request) *output.Error {
		// apply cors
		c.HandlerFunc(w, r)

		// then next
		return next(w, r)
	}
}
