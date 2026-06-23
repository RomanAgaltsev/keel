package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// checkRepo interprets the single-repo GET shared by the REST providers: 200 decodes
// the body (T.remote() yields the RemoteRepo), 404 means absent, and any other status
// becomes an apiError. label prefixes every error (e.g. "github: check owner/name").
// The request is issued via do so the response body is opened and closed in one place.
// On the read path an "already exists" conflict is not meaningful (existence is the 200
// case), so the ErrRepoExists mapping lives only in each provider's CreateRepo.
func checkRepo[T interface{ remote() RemoteRepo }](label string, do func() (*http.Response, error)) (bool, RemoteRepo, error) {
	resp, err := do()
	if err != nil {
		return false, RemoteRepo{}, fmt.Errorf("%s: %w", label, err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		var r T
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
			return false, RemoteRepo{}, err
		}
		return true, r.remote(), nil
	case http.StatusNotFound:
		return false, RemoteRepo{}, nil
	default:
		return false, RemoteRepo{}, apiError(label, resp, 0, "")
	}
}
