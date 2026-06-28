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
	// Jellyfin official repos — stable (34 plugins) and unstable/nightly (35 plugins).
	// Stable covers: Fanart, LDAP Auth, Trakt, Open Subtitles, TheTVDB, AniDB/AniList/
	// AniSearch/Kitsu, TMDb Box Sets, Bookshelf, Playback Reporting, Webhook, etc.
	// Unstable adds: Cover Art Archive, Artwork, and preview builds of stable plugins.
	{
		Name:     "Jellyfin Official (Stable)",
		URL:      "https://repo.jellyfin.org/master/plugin/manifest.json",
		Priority: 100,
	},
	{
		Name:     "Jellyfin Official (Unstable / Nightly)",
		URL:      "https://repo.jellyfin.org/master/plugin-unstable/manifest.json",
		Priority: 98,
	},
	// Intro Skipper: version-specific manifests
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
	// Community plugins
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
		Name:     "FinTube",
		URL:      "https://raw.githubusercontent.com/AECX/FinTube/master/manifest.json",
		Priority: 65,
	},
	{
		Name:     "Ani-Sync",
		URL:      "https://raw.githubusercontent.com/vosmiic/jellyfin-ani-sync/master/manifest.json",
		Priority: 63,
	},
	{
		Name:     "AVDC (AV元数据)",
		URL:      "https://raw.githubusercontent.com/xjasonlyu/jellyfin-plugin-avdc/main/manifest.json",
		Priority: 60,
	},
	{
		Name:     "MetaTube (AV元数据)",
		URL:      "https://raw.githubusercontent.com/metatube-community/jellyfin-plugin-metatube/dist/manifest.json",
		Priority: 58,
	},
	{
		Name:     "Letterboxd Sync",
		URL:      "https://raw.githubusercontent.com/builtbyproxy/jellyfin-plugin-letterboxd/main/manifest.json",
		Priority: 57,
	},
	{
		Name:     "MDBList Ratings",
		URL:      "https://raw.githubusercontent.com/Druidblack/Jellyfin.Plugin.MDBList_Ratings/master/manifest.json",
		Priority: 56,
	},
	{
		Name:     "ListenBrainz",
		URL:      "https://raw.githubusercontent.com/lyarenei/jellyfin-plugin-listenbrainz/master/manifest.json",
		Priority: 55,
	},
	{
		Name:     "Collection Import",
		URL:      "https://raw.githubusercontent.com/lostb1t/jellyfin-plugin-collection-import/main/manifest.json",
		Priority: 53,
	},
	{
		Name:     "Intros (dkanada)",
		URL:      "https://raw.githubusercontent.com/dkanada/jellyfin-plugin-intros/master/manifest.json",
		Priority: 52,
	},
	{
		Name:     "Auto Collections",
		URL:      "https://raw.githubusercontent.com/KeksBombe/jellyfin-plugin-auto-collections/main/manifest.json",
		Priority: 50,
	},

	// ── 中文 / Chinese metadata ────────────────────────────────────────────────
	// MetaShark: 2080⭐, Douban + TMDb dual-source Chinese metadata
	{
		Name:     "MetaShark (豆瓣+TMDb)",
		URL:      "https://github.com/cxfksword/jellyfin-plugin-metashark/releases/download/manifest/manifest.json",
		Priority: 48,
	},
	// Danmu: 635⭐, Chinese danmaku/弹幕 overlay.
	// manifest_cn.json points to CN-optimised download mirrors (faster from mainland).
	{
		Name:     "Danmu (弹幕, CN)",
		URL:      "https://github.com/cxfksword/jellyfin-plugin-danmu/releases/download/manifest/manifest_cn.json",
		Priority: 47,
	},
	{
		Name:     "Danmu (弹幕)",
		URL:      "https://github.com/cxfksword/jellyfin-plugin-danmu/releases/download/manifest/manifest.json",
		Priority: 46,
	},
	// Douban: 663⭐, standalone Douban metadata provider
	{
		Name:     "Douban (豆瓣)",
		URL:      "https://github.com/Libitum/jellyfin-plugin-douban/releases/latest/download/manifest.json",
		Priority: 44,
	},

	// ── Auth / SSO ─────────────────────────────────────────────────────────────
	// SSO: 1454⭐, SAML/OpenID Single Sign-On; manifest on manifest-release branch
	{
		Name:     "SSO Authentication",
		URL:      "https://raw.githubusercontent.com/9p4/jellyfin-plugin-sso/manifest-release/manifest.json",
		Priority: 42,
	},

	// ── Anime ─────────────────────────────────────────────────────────────────
	// Shokofin: 289⭐, AniDB-backed anime library management; metadata/stable branch
	{
		Name:     "Shokofin (AniDB)",
		URL:      "https://raw.githubusercontent.com/ShokoAnime/Shokofin/metadata/stable/manifest.json",
		Priority: 40,
	},

	// ── UI / Player ────────────────────────────────────────────────────────────
	// danieladov multi-repo: Merge Versions (638⭐) + Theme Songs (158⭐) + Skin Manager (419⭐)
	{
		Name:     "Merge Versions + Theme Songs + Skin Manager",
		URL:      "https://raw.githubusercontent.com/danieladov/JellyfinPluginManifest/master/manifest.json",
		Priority: 38,
	},
	// InPlayerEpisodePreview: 383⭐, chapter/episode preview thumbnails inside the player
	{
		Name:     "InPlayer Episode Preview",
		URL:      "https://raw.githubusercontent.com/Namo2/InPlayerEpisodePreview/master/manifest.json",
		Priority: 36,
	},
	// IAmParadox27: File Transformation + Plugin Pages + Home Screen Sections (UI framework)
	{
		Name:     "IAmParadox27 (UI Framework)",
		URL:      "https://raw.githubusercontent.com/IAmParadox27/jellyfin-plugin-repo/main/manifest-cache.json",
		Priority: 35,
	},

	// ── Subtitles ──────────────────────────────────────────────────────────────
	// SubBuzz: multi-source subtitle downloader, versioned per Jellyfin version
	{
		Name:     "SubBuzz (10.11, multi-source subs)",
		URL:      "https://raw.githubusercontent.com/josdion/subbuzz/master/repo/jellyfin_10.11.json",
		Priority: 34,
	},

	// ── Metadata / Library tools ───────────────────────────────────────────────
	// Viperinius: NFO Chapters + Spotify playlist import in one manifest
	{
		Name:     "NFO Chapters + Spotify Import",
		URL:      "https://raw.githubusercontent.com/Viperinius/jellyfin-plugins/master/manifest.json",
		Priority: 32,
	},
	// ankenyr: YouTube Metadata + Smart Playlist
	{
		Name:     "YouTube Metadata + Smart Playlist",
		URL:      "https://raw.githubusercontent.com/ankenyr/jellyfin-plugin-repo/master/manifest.json",
		Priority: 30,
	},

	// ── Scrobbling ─────────────────────────────────────────────────────────────
	{
		Name:     "Last.fm Scrobbler",
		URL:      "https://raw.githubusercontent.com/pepebarrascout/jellyfin-plugin-lastfm/main/manifest.json",
		Priority: 28,
	},
	// MediaTracker: sync with self-hosted MediaTracker (alternative to Trakt)
	{
		Name:     "MediaTracker Sync",
		URL:      "https://raw.githubusercontent.com/bonukai/jellyfin-plugin-mediatracker/main/manifest.json",
		Priority: 26,
	},

	// ── Notifications ──────────────────────────────────────────────────────────
	{
		Name:     "Discord Notifier",
		URL:      "https://raw.githubusercontent.com/cedev-1/jellyfin-plugin-DiscordNotifier/main/manifest.json",
		Priority: 24,
	},

	// ── Library management ─────────────────────────────────────────────────────
	// Mind the Gaps: scan for missing episodes in library
	{
		Name:     "Mind the Gaps (缺集检测)",
		URL:      "https://raw.githubusercontent.com/IDisposable/jellyfin-plugin-mindthegaps/main/manifest.json",
		Priority: 22,
	},
	// trakt-ex: InoriHimea's extended Trakt fork with additional sync features
	{
		Name:     "Trakt Extended (trakt-ex)",
		URL:      "https://raw.githubusercontent.com/InoriHimea/jellyfin-plugin-trakt-ex/master/repo/manifest.json",
		Priority: 20,
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
