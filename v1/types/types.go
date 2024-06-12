package types

type LibraryEntry struct {
	FilePath string `yaml:"file_path"`
	RedisKey string `yaml:"redis_key"`
}

type UpdatePositionRequest struct {
	LibraryKey   string `json:"library_key"`
	SessionID   string `json:"session_id"`
	UUID  string `json:"uuid"`
	Position int `json:"position"`
	Duration int `json:"duration"`
	Finished bool `json:"finished"`
	ReadyURL string `json:"ready_url"`
}

type ConfigFile struct {
	ServerName string `yaml:"server_name"`
	ServerBaseUrl string `yaml:"server_base_url"`
	ServerLiveUrl string `yaml:"server_live_url"`
	ServerPrivateUrl string `yaml:"server_private_url"`
	ServerPublicUrl string `yaml:"server_public_url"`
	ServerPort string `yaml:"server_port"`
	ServerAPIKey string `yaml:"server_api_key"`
	ServerLoginUrlPrefix string `yaml:"server_login_url_prefix"`
	ServerCookieName string `yaml:"server_cookie_name"`
	ServerCookieSecret string `yaml:"server_cookie_secret"`
	ServerCookieAdminSecretMessage string `yaml:"server_cookie_admin_secret_message"`
	ServerCookieSecretMessage string `yaml:"server_cookie_secret_message"`
	AdminUsername string `yaml:"admin_username"`
	AdminPassword string `yaml:"admin_password"`
	TimeZone string `yaml:"time_zone"`
	SaveFilesPath string `yaml:"save_files_path"`
	BoltDBPath string `yaml:"bolt_db_path"`
	EncryptionKey string `yaml:"encryption_key"`
	RedisAddress string `yaml:"redis_address"`
	RedisDBNumber int `yaml:"redis_db_number"`
	RedisPassword string `yaml:"redis_password"`
	AllowOrigins string `yaml:"allow_origins"`
	FilesURLPrefix string `yaml:"files_url_prefix"`
	SessionKey string `yaml:"session_key"`
	LibraryGlobalRedisKey string `yaml:"library_global_redis_key"`
	Library map[string]LibraryEntry `yaml:"library"`
}