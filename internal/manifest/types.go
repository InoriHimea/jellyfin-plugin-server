package manifest

// Catalog is the top-level structure of a Jellyfin plugin manifest.json.
type Catalog []Plugin

type Plugin struct {
	GUID        string    `json:"guid"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Overview    string    `json:"overview"`
	Owner       string    `json:"owner"`
	Category    string    `json:"category"`
	ImageURL    string    `json:"imageUrl,omitempty"`
	Versions    []Version `json:"versions"`
}

type Version struct {
	Version   string `json:"version"`
	ChangeLog string `json:"changelog"`
	TargetABI string `json:"targetAbi"`
	SourceURL string `json:"sourceUrl"`
	Checksum  string `json:"checksum"`
	Timestamp string `json:"timestamp"`
}
