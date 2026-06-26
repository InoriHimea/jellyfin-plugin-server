package db

import (
	"github.com/google/uuid"
)

type Repo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Enabled     bool   `json:"enabled"`
	Priority    int    `json:"priority"`
	LastFetched string `json:"last_fetched,omitempty"`
	ETag        string `json:"etag,omitempty"`
	CreatedAt   string `json:"created_at"`
}

var defaultRepos = []struct {
	Name     string
	URL      string
	Priority int
}{
	{
		Name:     "Jellyfin Official",
		URL:      "https://repo.jellyfin.org/releases/plugin/manifest.json",
		Priority: 100,
	},
	{
		Name:     "Intro Skipper",
		URL:      "https://raw.githubusercontent.com/jumoog/intro-skipper/master/manifest.json",
		Priority: 80,
	},
	{
		Name:     "Open Subtitles",
		URL:      "https://raw.githubusercontent.com/nickeyl/jellyfin-opensubtitles/master/manifest.json",
		Priority: 70,
	},
	{
		Name:     "Ani-Sync",
		URL:      "https://raw.githubusercontent.com/vosmiic/jellyfin-ani-sync/master/manifest.json",
		Priority: 60,
	},
}

// SeedDefaultRepos inserts built-in repos if the table is empty.
func SeedDefaultRepos() error {
	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM repos`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	for _, r := range defaultRepos {
		if _, err := DB.Exec(
			`INSERT INTO repos (id, name, url, enabled, priority, created_at)
			 VALUES (?, ?, ?, 1, ?, ?)`,
			uuid.NewString(), r.Name, r.URL, r.Priority, Now(),
		); err != nil {
			return err
		}
	}
	return nil
}

func ListRepos() ([]Repo, error) {
	rows, err := DB.Query(
		`SELECT id, name, url, enabled, priority,
		        COALESCE(last_fetched,''), COALESCE(etag,''), created_at
		 FROM repos ORDER BY priority DESC, name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []Repo
	for rows.Next() {
		var r Repo
		var enabled int
		if err := rows.Scan(&r.ID, &r.Name, &r.URL, &enabled, &r.Priority, &r.LastFetched, &r.ETag, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.Enabled = enabled == 1
		repos = append(repos, r)
	}
	return repos, nil
}

func GetRepo(id string) (*Repo, error) {
	r := &Repo{}
	var enabled int
	err := DB.QueryRow(
		`SELECT id, name, url, enabled, priority,
		        COALESCE(last_fetched,''), COALESCE(etag,''), created_at
		 FROM repos WHERE id=?`, id,
	).Scan(&r.ID, &r.Name, &r.URL, &enabled, &r.Priority, &r.LastFetched, &r.ETag, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	r.Enabled = enabled == 1
	return r, nil
}

func CreateRepo(name, repoURL string, priority int) (*Repo, error) {
	id := uuid.NewString()
	now := Now()
	_, err := DB.Exec(
		`INSERT INTO repos (id, name, url, enabled, priority, created_at) VALUES (?, ?, ?, 1, ?, ?)`,
		id, name, repoURL, priority, now,
	)
	if err != nil {
		return nil, err
	}
	return &Repo{ID: id, Name: name, URL: repoURL, Enabled: true, Priority: priority, CreatedAt: now}, nil
}

func UpdateRepo(id, name, repoURL string, enabled bool, priority int) error {
	en := 0
	if enabled {
		en = 1
	}
	_, err := DB.Exec(
		`UPDATE repos SET name=?, url=?, enabled=?, priority=? WHERE id=?`,
		name, repoURL, en, priority, id,
	)
	return err
}

func DeleteRepo(id string) error {
	_, err := DB.Exec(`DELETE FROM repos WHERE id=?`, id)
	return err
}
