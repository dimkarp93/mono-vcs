package gitlab

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dimkarp93/mono-vcs/internal/config"
)

type Project struct {
	ID                int    `json:"id"`
	PathWithNamespace string `json:"path_with_namespace"`
	HTTPURLToRepo     string `json:"http_url_to_repo"`
}

var client = &http.Client{Timeout: 30 * time.Second}

func FetchProjects(glURL, token string) ([]Project, error) {
	base := strings.TrimRight(glURL, "/") + "/api/v4/projects"
	var out []Project
	page := 1
	for {
		q := url.Values{}
		q.Set("membership", "true")
		q.Set("simple", "true")
		q.Set("archived", "false")
		q.Set("per_page", strconv.Itoa(config.PerPage))
		q.Set("page", strconv.Itoa(page))
		q.Set("order_by", "path")
		q.Set("sort", "asc")

		req, err := http.NewRequest(http.MethodGet, base+"?"+q.Encode(), nil)
		if err != nil {
			return nil, fmt.Errorf("Cannot reach GitLab at %s: %v", glURL, err)
		}
		req.Header.Set("PRIVATE-TOKEN", token)
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("Cannot reach GitLab at %s: %v", glURL, err)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			if resp.StatusCode == http.StatusUnauthorized {
				return nil, errors.New("GitLab returned 401 Unauthorized — check --gl-token")
			}
			return nil, fmt.Errorf("GitLab API error: HTTP %d %s",
				resp.StatusCode, http.StatusText(resp.StatusCode))
		}
		var batch []Project
		err = json.NewDecoder(resp.Body).Decode(&batch)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("Cannot reach GitLab at %s: %v", glURL, err)
		}
		if len(batch) == 0 {
			break
		}
		out = append(out, batch...)
		if len(batch) < config.PerPage {
			break
		}
		page++
	}
	return out, nil
}

func FetchRemoteBranchSHA(glURL, token string, projectID int, branch string) string {
	encoded := url.PathEscape(branch)
	u := fmt.Sprintf("%s/api/v4/projects/%d/repository/branches/%s",
		strings.TrimRight(glURL, "/"), projectID, encoded)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("PRIVATE-TOKEN", token)
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	var data struct {
		Commit struct {
			ID string `json:"id"`
		} `json:"commit"`
	}
	if json.NewDecoder(resp.Body).Decode(&data) != nil {
		return ""
	}
	return data.Commit.ID
}
