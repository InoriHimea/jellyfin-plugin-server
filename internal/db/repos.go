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
	// Intro Skipper: version-specific manifests from intro-skipper/manifest
	{
		Name:     "Intro Skipper (10.11)",
		URL:      "https://raw.githubusercontent.com/intro-skipper/manifest/main/10.11/manifest.json",
		Priority: 92,
	},
	{
		Name:     "Intro Skipper (10.10)",
		URL:      "https://raw.githubusercontent.com/intro-skipper/manifest/main/10.10/manifest.json",
		Priority: 90,
	},
	// Community plugins verified from awesome-jellyfin
	{
		Name:     "JellyScrub",
		URL:      "https://raw.githubusercontent.com/nicknsy/jellyscrub/master/manifest.json",
		Priority: 78,
	},
	{
		Name:     "Streamyfin",
		URL:      "https://raw.githubusercontent.com/streamyfin/jellyfin-plugin-streamyfin/main/manifest.json",
		Priority: 75,
	},
	{
		Name:     "Lyrics",
		URL:      "https://raw.githubusercontent.com/Felitendo/jellyfin-plugin-lyrics/main/manifest.json",
		Priority: 70,
	},
	{
		Name:     "FinTube",
		URL:      "https://raw.githubusercontent.com/AECX/FinTube/master/manifest.json",
		Priority: 65,
	},
	{
		Name:     "Ani-Sync",
		URL:      "https://raw.githubusercontent.com/vosmiic/jellyfin-ani-sync/master/manifest.json",
		Priority: 60,
	},
	{
		Name:     "ListenBrainz",
		URL:      "https://raw.githubusercontent.com/lyarenei/jellyfin-plugin-listenbrainz/master/manifest.json",
		Priority: 58,
	},
	{
		Name:     "AVDC Metadata",
		URL:      "https://raw.githubusercontent.com/xjasonlyu/jellyfin-plugin-avdc/main/manifest.json",
		Priority: 55,
	},
	{
		Name:     "Intros",
		URL:      "https://raw.githubusercontent.com/dkanada/jellyfin-plugin-intros/master/manifest.json",
		Priority: 53,
	},
	{
		Name:     "Auto Collections",
		URL:      "https://raw.githubusercontent.com/KeksBombe/jellyfin-plugin-auto-collections/main/manifest.json",
		Priority: 50,
	},
}

// SeedDefaultRepos upserts built-in repos by URL (INSERT OR IGNORE), so new
// defaults are always added while user-added or user-modified repos are untouched.
func SeedDefaultRepos() error {
	for _, r := range defaultRepos {
		if _, err := DB.Exec(
			`INSERT OR IGNORE INTO repos (id, name, url, enabled, priority, created_at)
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
