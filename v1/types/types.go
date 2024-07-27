package types

type LibraryEntry struct {
	FilePath string `yaml:"file_path"`
	RedisKey string `yaml:"redis_key"`
}

type UpdatePositionRequest struct {
	LibraryKey   string `json:"library_key"`
	SessionID   string `json:"session_id"`
	YouTubePlaylistID string `json:"youtube_playlist_id"`
	YouTubePlaylistIndex int `json:"youtube_playlist_index"`
	Type string `json:"type"`
	Title string `json:"title"`
	UUID  string `json:"uuid"`
	Position int `json:"position"`
	Duration int `json:"duration"`
	Finished bool `json:"finished"`
	ReadyURL string `json:"ready_url"`
}

type Library map[string]LibraryEntry

type GetMediaHTMLParams struct {
	SessionKey string `json:"session_key"`
	FilesURLPrefix string `json:"files_url_prefix"`
	LibraryKey string `json:"library_key"`
	SessionID string `json:"session_id"`
	TimeStr string `json:"time_str"`
	NextID string `json:"next_id"`
	Extension string `json:"extension"`
	ReadyURL string `json:"ready_url"`
	Type string `json:"type"`
}

type GetYouTubePlaylistParams struct {
	SessionKey string `json:"session_key"`
	LibraryKey string `json:"library_key"`
	PlaylistID string `json:"playlist_id"`
	ListID string `json:"library_key"`
	SessionID string `json:"session_id"`
	ReadyURL string `json:"ready_url"`
	Type string `json:"type"`
	Index string `json:"index"`
	Time string `json:"time"`
}