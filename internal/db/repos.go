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
	// Bangumi (番组计划): kookxiang's implementation, self-hosted at kookxiang.dev
	// Uses repository.json filename — valid Jellyfin manifest format, 21 versions
	{
		Name:     "Bangumi (番组计划)",
		URL:      "https://jellyfin-plugin-bangumi.kookxiang.dev/repository.json",
		Priority: 49,
	},
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
	// MyAnimeList metadata provider
	{
		Name:     "MyAnimeList",
		URL:      "https://raw.githubusercontent.com/ryandash/jellyfin-plugin-myanimelist/main/manifest.json",
		Priority: 39,
	},
	// MyAnimeSync: watch history sync to MyAnimeList
	{
		Name:     "MyAnimeSync",
		URL:      "https://raw.githubusercontent.com/iankiller77/MyAnimeSync/main/manifest.json",
		Priority: 38,
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
	// LizardByte: Themerr — auto-add theme songs via ThemerrDB (GitHub Pages hosted)
	{
		Name:     "LizardByte (Themerr)",
		URL:      "https://lizardbyte.github.io/jellyfin-plugin-repo/manifest.json",
		Priority: 33,
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
	// TMDb Trailers: dedicated trailer channel from TMDb
	{
		Name:     "TMDb Trailers",
		URL:      "https://raw.githubusercontent.com/crobibero/jellyfin-plugin-tmdb-trailers/master/manifest.json",
		Priority: 29,
	},
	// AnimeThemes: fetch anime OP/ED theme songs from animethemes.moe
	{
		Name:     "AnimeThemes (动漫OP/ED)",
		URL:      "https://raw.githubusercontent.com/EusthEnoptEron/jellyfin-plugin-animethemes/main/manifest.json",
		Priority: 27,
	},
	// TheSportsDB: sports event metadata (leagues, teams, events)
	{
		Name:     "TheSportsDB (体育赛事)",
		URL:      "https://raw.githubusercontent.com/retrorat1/Jellyfin.Plugin.TheSportsDB/main/manifest.json",
		Priority: 25,
	},
	// Episode Poster Generator: auto-generate episode posters from screenshots
	{
		Name:     "Episode Poster Generator",
		URL:      "https://raw.githubusercontent.com/JPKribs/jellyfin-plugin-episodepostergenerator/master/manifest.json",
		Priority: 23,
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
	{
		Name:     "Telegram Notifier",
		URL:      "https://raw.githubusercontent.com/RomainPierre7/jellyfin-plugin-TelegramNotifier/main/manifest.json",
		Priority: 21,
	},
	// Newsletter: generate and send email digests of new library additions
	{
		Name:     "Newsletters (邮件摘要)",
		URL:      "https://raw.githubusercontent.com/Cloud9Developer/Jellyfin-Newsletter-Plugin/master/manifest.json",
		Priority: 19,
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

	// ── Anime metadata ──────────────────────────────────────────────────────────
	// Shikimori: Russian anime tracker — metadata + watch state sync
	{
		Name:     "Shikimori (俄罗斯动漫追踪)",
		URL:      "https://raw.githubusercontent.com/te9c/jellyfin-plugin-shikimori/main/manifest.json",
		Priority: 18,
	},
	// AnimeMultiSource: aggregate anime metadata from multiple providers in one pass
	{
		Name:     "Anime Multi Source",
		URL:      "https://raw.githubusercontent.com/webbster64/jellyfin-plugin-AnimeMultiSource/main/manifest.json",
		Priority: 16,
	},

	// ── Letterboxd (alternative implementation) ────────────────────────────────
	// Gizmo091 variant — maintained separately from builtbyproxy
	{
		Name:     "Letterboxd Sync (Gizmo091)",
		URL:      "https://raw.githubusercontent.com/Gizmo091/jellyfin-plugin-letterboxd-sync/master/manifest.json",
		Priority: 14,
	},

	// ── Broadcast / scheduling ─────────────────────────────────────────────────
	// Air Times: show broadcast air times and next-episode countdowns
	{
		Name:     "Air Times (播出时间)",
		URL:      "https://raw.githubusercontent.com/k0d13/jellyfin-air-times/main/manifest.json",
		Priority: 12,
	},

	// ── Discovery & recommendations ────────────────────────────────────────────
	{
		Name:     "Local Recommendations",
		URL:      "https://raw.githubusercontent.com/rdpharr/jellyfin-plugin-localrecs/main/manifest.json",
		Priority: 10,
	},

	// ── UI tweaks ──────────────────────────────────────────────────────────────
	{
		Name:     "Jellyfin Tweaks",
		URL:      "https://raw.githubusercontent.com/n00bcodr/JellyfinTweaks/main/manifest.json",
		Priority: 8,
	},

	// ── AI subtitles ───────────────────────────────────────────────────────────
	// WhisperSubs: on-device speech-to-text subtitle generation via OpenAI Whisper
	{
		Name:     "WhisperSubs (AI字幕生成)",
		URL:      "https://raw.githubusercontent.com/GeiserX/whisper-subs/main/manifest.json",
		Priority: 6,
	},

	// ── Artwork ────────────────────────────────────────────────────────────────
	// ArtworkMultiSource: fetch artwork from multiple providers simultaneously
	{
		Name:     "Artwork Multi Source",
		URL:      "https://raw.githubusercontent.com/Druidblack/Jellyfin.Plugin.ArtworkMultiSource/main/manifest.json",
		Priority: 4,
	},

	// ── Russian metadata ───────────────────────────────────────────────────────
	// KinoPoisk: Russian movie database metadata (Яндекс КиноПоиск)
	{
		Name:     "КиноПоиск (KinoPoisk)",
		URL:      "https://raw.githubusercontent.com/LinFor/jellyfin-plugin-kinopoisk/master/dist/manifest.json",
		Priority: 2,
	},

	// ── TMDb extended ──────────────────────────────────────────────────────────
	// TMDbPlus: enhanced TMDb metadata with extra fields (cxfksword)
	{
		Name:     "TMDbPlus",
		URL:      "https://github.com/cxfksword/jellyfin-plugin-tmdbplus/releases/download/manifest/manifest.json",
		Priority: 1,
	},

	// ── Poster enhancement ─────────────────────────────────────────────────────
	// BetterPoster: auto-selects best available poster art for each item
	{
		Name:     "Btttr Posters Plugin",
		URL:      "https://raw.githubusercontent.com/TheAceOfficials/BetterPoster-for-Jellyfin/main/manifest.json",
		Priority: -1,
	},

	// ── IPTV / STRM / live sources ─────────────────────────────────────────────
	// AniLiberty: create STRM files for AniLibria streaming anime
	{
		Name:     "AniLiberty STRM Plugin",
		URL:      "https://queukat.github.io/AniLibriaStrmPlugin/plugins/manifest.json",
		Priority: -3,
	},
	// JellySTRMprobe + Xtream Library (firestaerter3 repo)
	{
		Name:     "JellySTRMprobe + Xtream Library",
		URL:      "https://firestaerter3.github.io/jellyfin-plugin-repo/manifest.json",
		Priority: -5,
	},

	// ── German public broadcast ────────────────────────────────────────────────
	// Mediathek Downloader: download content from German ARD/ZDF Mediathek
	{
		Name:     "Mediathek Downloader",
		URL:      "https://raw.githubusercontent.com/CatNoir2006/jellyfin-plugin-manifest/main/manifest.json",
		Priority: -7,
	},

	// ── Transcoding UX ─────────────────────────────────────────────────────────
	// Transcode Nag: warn users when direct play is available instead of transcoding
	{
		Name:     "Transcode Nag",
		URL:      "https://raw.githubusercontent.com/voc0der/jellyfin-transcode-nag/main/manifest.json",
		Priority: -9,
	},

	// ── Intro / credit skipping ────────────────────────────────────────────────
	// Intro Skipper: audio fingerprinting to auto-detect and skip TV intros/credits
	// Not in official stable; maintained by ConfusedPolarBear (community handoff pending)
	{
		Name:     "Intro Skipper (片头跳过)",
		URL:      "https://raw.githubusercontent.com/ConfusedPolarBear/intro-skipper/master/manifest.json",
		Priority: -11,
	},
	// TheIntroDB: alternative intro-skip using TheIntroDB.com crowd-sourced timestamps
	{
		Name:     "TheIntroDB (跳片头)",
		URL:      "https://raw.githubusercontent.com/TheIntroDB/jellyfin-plugin/main/manifest.json",
		Priority: -12,
	},

	// ── Metadata: music ────────────────────────────────────────────────────────
	// Lyrics: auto-download synchronized lyrics (LRC) from multiple sources
	{
		Name:     "Lyrics (自动下载歌词)",
		URL:      "https://raw.githubusercontent.com/Felitendo/jellyfin-plugin-lyrics/main/manifest.json",
		Priority: -13,
	},
	// AudioMuse AI: real-time sonic-analysis-based music queue recommendation
	{
		Name:     "AudioMuse AI (AI音乐推荐)",
		URL:      "https://raw.githubusercontent.com/NeptuneHub/audiomuse-ai-plugin/master/manifest.json",
		Priority: -14,
	},

	// ── Authentication ─────────────────────────────────────────────────────────
	// Authelia: delegate auth to a self-hosted Authelia instance
	{
		Name:     "Authelia Authentication",
		URL:      "https://raw.githubusercontent.com/nikarh/jellyfin-plugin-authelia/main/manifest.json",
		Priority: -15,
	},
	// Jellyfin Security: TOTP 2FA, email OTP, trusted devices, TV pairing
	{
		Name:     "Jellyfin Security (2FA/TOTP)",
		URL:      "https://raw.githubusercontent.com/ZL154/JellyfinSecurity/main/manifest.json",
		Priority: -16,
	},

	// ── UI enhancements ────────────────────────────────────────────────────────
	// InPlayerEpisodePreview: episode list panel inside the video player
	{
		Name:     "InPlayer Episode Preview",
		URL:      "https://raw.githubusercontent.com/Namo2/InPlayerEpisodePreview/master/manifest.json",
		Priority: -17,
	},
	// JMSFusion: smart media slider, hover trailers, recommendations, badges
	{
		Name:     "JMSFusion (MonWUI 高级UI)",
		URL:      "https://raw.githubusercontent.com/G-grbz/Jellyfin-MonWUI-Plugin/main/manifest.json",
		Priority: -18,
	},
	// Cinema Mode: play local trailers and pre-rolls before main feature
	{
		Name:     "Cinema Mode (影院模式)",
		URL:      "https://raw.githubusercontent.com/CherryFloors/jellyfin-plugin-cinemamode/main/manifest.json",
		Priority: -19,
	},
	// JavaScript Injector: inject custom JS into the Jellyfin web UI
	{
		Name:     "JavaScript Injector",
		URL:      "https://raw.githubusercontent.com/n00bcodr/Jellyfin-JavaScript-Injector/main/manifest.json",
		Priority: -20,
	},
	// JellyFlare: rotating announcement banners in the Jellyfin UI
	{
		Name:     "JellyFlare (公告横幅)",
		URL:      "https://raw.githubusercontent.com/MorganKryze/JellyFlare/main/manifest.json",
		Priority: -21,
	},
	// Seasonals: seasonal decorations/animations (Halloween, Christmas, etc.)
	{
		Name:     "Seasonals (季节主题动画)",
		URL:      "https://raw.githubusercontent.com/CodeDevMLH/Jellyfin-Seasonals/main/manifest.json",
		Priority: -22,
	},
	// Jellyfin-Roulette: pick a random unwatched item from a playlist
	{
		Name:     "Jellyfin Roulette (随机选片)",
		URL:      "https://raw.githubusercontent.com/ztffn/Jellyfin-Roulette/main/manifest.json",
		Priority: -23,
	},
	// Jellysleep: set a sleep timer that pauses or stops playback
	{
		Name:     "Jellysleep (睡眠定时器)",
		URL:      "https://raw.githubusercontent.com/jon4hz/jellyfin-plugin-jellysleep/main/manifest.json",
		Priority: -24,
	},

	// ── Library management ─────────────────────────────────────────────────────
	// Language Tags: automatically tag media with detected audio/subtitle language
	{
		Name:     "Language Tags (语言自动标签)",
		URL:      "https://raw.githubusercontent.com/TheXaman/jellyfin-plugin-languageTags/main/manifest.json",
		Priority: -25,
	},
	// Jellyfin Ignore: exclude specific paths/folders from library scans
	{
		Name:     "Jellyfin Ignore (路径排除)",
		URL:      "https://raw.githubusercontent.com/fdett/jellyfin-ignore/master/manifest.json",
		Priority: -26,
	},

	// ── Server administration ──────────────────────────────────────────────────
	// StreamLimiter: cap simultaneous transcoding/streaming sessions per user
	{
		Name:     "Stream Limiter (并发流限制)",
		URL:      "https://raw.githubusercontent.com/JellyboxAD/Jellyfin.Plugin.StreamLimit/main/manifest.json",
		Priority: -27,
	},
	// Streamyfin: companion plugin for the Streamyfin iOS/Android client
	{
		Name:     "Streamyfin Companion",
		URL:      "https://raw.githubusercontent.com/streamyfin/jellyfin-plugin-streamyfin/main/manifest.json",
		Priority: -28,
	},
	// RemoteUpload: remote download from URL + direct file management via browser
	{
		Name:     "RemoteUpload (远程下载)",
		URL:      "https://raw.githubusercontent.com/GrandguyJS/media-upload-plugin/main/manifest.json",
		Priority: -29,
	},
	// Meilisearch: replaces Jellyfin's built-in search with a Meilisearch backend
	{
		Name:     "Meilisearch (全文搜索)",
		URL:      "https://raw.githubusercontent.com/arnesacnussem/jellyfin-plugin-meilisearch/master/manifest.json",
		Priority: -30,
	},
	// AlexaSkill: expose Jellyfin controls via an Amazon Alexa skill
	{
		Name:     "Alexa Skill",
		URL:      "https://raw.githubusercontent.com/infinityofspace/jellyfin-alexa-plugin/main/manifest.json",
		Priority: -31,
	},

	// ── AI / upscaling ─────────────────────────────────────────────────────────
	// AI Upscaler: real-time AI super-resolution for video streams
	{
		Name:     "AI Upscaler (AI超分辨率)",
		URL:      "https://raw.githubusercontent.com/Kuschel-code/JellyfinUpscalerPlugin/main/manifest.json",
		Priority: -32,
	},

	// ── Adult content ──────────────────────────────────────────────────────────
	// PhoenixAdult: metadata scraper for 200+ adult sites
	{
		Name:     "PhoenixAdult (多站成人元数据)",
		URL:      "https://raw.githubusercontent.com/DirtyRacer1337/Jellyfin.Plugin.PhoenixAdult/master/manifest.json",
		Priority: -33,
	},
	// Stash: pull metadata from a self-hosted Stash instance
	{
		Name:     "Stash Metadata",
		URL:      "https://raw.githubusercontent.com/DirtyRacer1337/Jellyfin.Plugin.Stash/main/manifest.json",
		Priority: -34,
	},

	// ── Letterboxd (third implementation) ─────────────────────────────────────
	{
		Name:     "Letterboxd Sync (danielveigasilva)",
		URL:      "https://raw.githubusercontent.com/danielveigasilva/jellyfin-plugin-letterboxd-sync/master/manifest.json",
		Priority: -35,
	},

	// ── IPTV: Xtream (primary, GitHub Pages) ───────────────────────────────────
	// Kevinjil: full Xtream-compatible API integration, highest-star Xtream plugin
	{
		Name:     "Jellyfin Xtream (Kevinjil)",
		URL:      "https://kevinjil.github.io/Jellyfin.Xtream/repository.json",
		Priority: -36,
	},

	// ── Smart / dynamic playlists ──────────────────────────────────────────────
	// SmartLists: rule-based dynamic playlists and collections, auto-refreshed
	{
		Name:     "SmartLists (动态播放列表)",
		URL:      "https://raw.githubusercontent.com/jyourstone/jellyfin-plugin-manifest/main/manifest.json",
		Priority: -37,
	},

	// ── shemanaev plugin bundle (MyShows + Webhooks + Media Cleaner) ───────────
	// MyShows: Russian TV tracker sync; Media Cleaner: auto-delete played media
	{
		Name:     "shemanaev Bundle (MyShows+MediaCleaner+Webhooks)",
		URL:      "https://raw.githubusercontent.com/shemanaev/jellyfin-plugin-repo/refs/heads/master/manifest.json",
		Priority: -38,
	},

	// ── Douban (Xzonn standalone) ──────────────────────────────────────────────
	// Third Douban implementation, self-hosted manifest on xzonn.top
	{
		Name:     "Douban (Xzonn版)",
		URL:      "https://xzonn.top/JellyfinPluginDouban/manifest.json",
		Priority: -39,
	},

	// ── caryyu Gitee bundle (Open Douban + MaxSubtitle) ───────────────────────
	// Open Douban: open-source Douban metadata; MaxSubtitle: Chinese subtitle download
	{
		Name:     "Open Douban + MaxSubtitle (caryyu)",
		URL:      "https://gitee.com/caryyu/jellyfin-plugin-repo/raw/master/manifest-cn.json",
		Priority: -40,
	},

	// ── Apple Music metadata ───────────────────────────────────────────────────
	// lyarenei: Apple Music artist/album metadata provider (self-hosted manifest)
	{
		Name:     "Apple Music Metadata",
		URL:      "https://repo.xkrivo.net/jellyfin/manifest.json",
		Priority: -41,
	},

	// ── Better instant mix ─────────────────────────────────────────────────────
	// BetterMix: improved Instant Mix algorithm; manifest is platform-specific (linux-x64)
	{
		Name:     "BetterMix (更好的即时混音)",
		URL:      "https://raw.githubusercontent.com/StergiosBinopoulos/jellyfin-plugin-bettermix/refs/heads/main/manifest-linux-x64.json",
		Priority: -42,
	},

	// ── Video quality badges ────────────────────────────────────────────────────
	// Quality Overlay: overlays HDR/DV/codec badges onto poster and backdrop images
	{
		Name:     "Quality Overlay (画质标签覆盖)",
		URL:      "https://raw.githubusercontent.com/obxidion/Jellyfin-Quality-Overlay/main/manifest.json",
		Priority: -43,
	},

	// ── Poster enhancement (NeurekaSoftware) ───────────────────────────────────
	// Better Posters: auto-select best poster per item (different from TheAceOfficials version)
	{
		Name:     "Better Posters (Neureka)",
		URL:      "https://code.neureka.dev/jellyfin/better-posters/raw/branch/master/manifest.json",
		Priority: -44,
	},

	// ── Trailers ───────────────────────────────────────────────────────────────
	// Trailers4Jellyfin: fetch and play trailers from multiple sources
	{
		Name:     "Trailers4Jellyfin",
		URL:      "https://raw.githubusercontent.com/robadieNZ/Trailers4Jellyfin/main/manifest.json",
		Priority: -45,
	},

	// ── Watch together / social ────────────────────────────────────────────────
	// Binge Buddy: watch-party notifications and binge session companion
	{
		Name:     "Binge Buddy (一起追剧)",
		URL:      "https://raw.githubusercontent.com/Cyprien-png/jellyfin-binge-buddy/master/manifest.json",
		Priority: -46,
	},

	// ── Library management ─────────────────────────────────────────────────────
	// Hide Empty Folders: remove empty TV show folders from library view
	{
		Name:     "Hide Empty Folders (隐藏空文件夹)",
		URL:      "https://raw.githubusercontent.com/CapstonPeters/Jellyfin-Hide-Empty-Folders/main/manifest.json",
		Priority: -47,
	},

	// ── Live TV ────────────────────────────────────────────────────────────────
	// Live TV Builder: build and manage live TV playlists/channels from STRM sources
	{
		Name:     "Live TV Builder",
		URL:      "https://raw.githubusercontent.com/obxidion/Live-TV-Builder-Jellyfin-Plugin/main/manifest.json",
		Priority: -48,
	},

	// ── IMDb ratings overlay ───────────────────────────────────────────────────
	// TUIMDB: display IMDb ratings and metadata badges on media items
	{
		Name:     "TUIMDB (IMDb评分标签)",
		URL:      "https://tuimdb.com/jellyfin/manifest.json",
		Priority: -49,
	},

	// ── HTTP authentication ────────────────────────────────────────────────────
	// HttpAuth: protect Jellyfin with HTTP Basic / header-based authentication
	{
		Name:     "HttpAuth",
		URL:      "https://raw.githubusercontent.com/UlysseM/jellyfin-plugin-httpauth/gh-pages/repository.json",
		Priority: -50,
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
