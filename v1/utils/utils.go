package utils

import (
	"fmt"
	// "time"
	// "runtime"
	filepath "path/filepath"
	"strings"
	"strconv"
	"io/ioutil"
	"net/url"
	"context"
	"gopkg.in/yaml.v3"
	redis "github.com/redis/go-redis/v9"
	slug "github.com/gosimple/slug"
	types "github.com/0187773933/FileServer-Media/v1/types"
	// fiber_cookie "github.com/gofiber/fiber/v2/middleware/encryptcookie"
	// encryption "github.com/0187773933/encryption/v1/encryption"
	circular_set "github.com/0187773933/RedisCircular/v1/set"
	server "github.com/0187773933/GO_SERVER/v1/server"
)

func IsURL( input string ) ( result bool ) {
	result = false
	parsed_url , err := url.Parse( input )
	fmt.Println( parsed_url , parsed_url.Scheme )
	if err == nil {
		if parsed_url.Scheme == "http" || parsed_url.Scheme == "https" {
			result = true
		}
	}
	return
}

type Entry struct {
	Name string `yaml:"-"`
	Extension string  `yaml:"extension,omitempty"`
	ID string `yaml:"id,omitempty"`
	Length int `yaml:"length,omitempty"`
	Path string `yaml:"path,omitempty"`
	Items []*Entry `yaml:"items,omitempty"`
}

func ( e *Entry ) UnmarshalYAML( value *yaml.Node ) error {
	if value.Kind == yaml.MappingNode {
		for i := 0; i < len( value.Content ); i += 2 {
			keyNode := value.Content[ i ]
			valueNode := value.Content[ ( i + 1 ) ]
			if keyNode.Value == "extension" {
				e.Extension = valueNode.Value
			} else if keyNode.Value == "id" {
				e.ID = valueNode.Value
			} else if keyNode.Value == "length" {
				fmt.Sscanf( valueNode.Value , "%d" , &e.Length )
			} else if keyNode.Value == "path" {
				e.Path = valueNode.Value
			} else {
				e.Name = keyNode.Value
				var items []*Entry
				if err := valueNode.Decode( &items ); err != nil {
					return err
				}
				e.Items = items
			}
		}
	} else if value.Kind == yaml.SequenceNode {
		for _ , itemNode := range value.Content {
			var item Entry
			if err := itemNode.Decode( &item ); err != nil {
				return err
			}
			e.Items = append( e.Items , &item )
		}
	}
	return nil
}

func FlattenEntries( entries []*Entry , flattened *[]Entry ) {
	for _, entry := range entries {
		*flattened = append( *flattened , *entry )
		if entry.Items != nil {
			FlattenEntries( entry.Items , flattened )
		}
	}
}

func GetFlattenedEntries( yaml_file_path string ) ( result []Entry ) {
	fmt.Println( "Reading YAML file:" , yaml_file_path )
	data , err := ioutil.ReadFile( yaml_file_path )
	if err != nil { fmt.Printf( "Error reading YAML file: %v\n" , err ); return }
	var entries []*Entry
	err = yaml.Unmarshal( data , &entries );
	if err != nil { fmt.Printf( "Failed to Unmarshal YAML: %v\n" , err ); return }
	FlattenEntries( entries , &result )
	return
}

func RedisGetBool( ctx context.Context , client *redis.Client , key string ) ( result bool ) {
	result_str := client.Get( ctx , key ).Val()
	result , _ = strconv.ParseBool( result_str )
	return
}

func DeleteKeysWithPattern( ctx context.Context , db *redis.Client , pattern string ) error {
	var cursor uint64
	var keys []string
	var err error
	for {
		keys, cursor, err = db.Scan( ctx , cursor , pattern , 0 ).Result()
		if err != nil { return err }
		if len( keys ) > 0 {
			if _ , err := db.Del( ctx , keys... ).Result(); err != nil {
				return err
			}
		}
		if cursor == 0 {
			break
		}
	}
	return nil
}

func ImportLibrarySaveFilesInToRedis( s *server.Server ) {
	var ctx = context.Background()
	// defer redis_connection.FlushDB( ctx ).Err()
	files , _ := ioutil.ReadDir( s.Config.SaveFilesPath )
	for _ , file := range files {

		if file.IsDir() { continue }
		if strings.HasSuffix( file.Name() , ".yaml" ) == false { continue }
		file_path := filepath.Join( s.Config.SaveFilesPath , file.Name() )
		library_entries := GetFlattenedEntries( file_path )
		total_entries := len( library_entries )
		if total_entries < 1 { continue }

		// global_circular_key := fmt.Sprintf( "%s.%s", s.Config.Redis.Prefix , library_entry.RedisKey )
		file_name := strings.TrimSuffix( file.Name() , filepath.Ext( file.Name() ) )
		file_name_slug := slug.Make( file_name )
		global_circular_key := fmt.Sprintf( "%s.%s" , s.Config.Redis.Prefix , file_name_slug )

		s.REDIS.SAdd( ctx , s.Config.Redis.Prefix + ".LIBRARY" , file_name_slug )

		// Force Reset
		redis_reset := true
		if redis_reset == true {
			// fmt.Println( "resetting global circular key:" , global_circular_key , file_path , total_entries )
			s.REDIS.Del( ctx , global_circular_key )
			// db.Del( ctx, global_circular_key + ".INDEX" )
		}

		for index, entry := range library_entries {
			if entry.Path == "" {
				continue
			}
			if entry.ID == "" {
				continue
			}
			fmt.Printf( "index: %d, name: %s, path: %s, id: %s\n" , index , entry.Name , entry.Path , entry.ID )
			global_entry_key := fmt.Sprintf( "%s.%s" , s.Config.Redis.Prefix , entry.ID )
			// Minimum need the path, could json blob store here instead
			s.REDIS.Set( ctx , global_entry_key , entry.Path , 0 )
			// So these are setting up "sessions"
			// a "session" here is just an ephemeral copy of the circular set's index tracking
			fmt.Println( "adding" , global_circular_key , entry.ID )
			circular_set.Add( s.REDIS , global_circular_key , entry.ID )
		}
		fmt.Println( "done" )
	}
	// for _ , file := range files {
	// 	fmt.Println( file.Name() )
	// }
}

func GetMediaHTML( params types.GetMediaHTMLParams ) ( html string ) {
	var html_file []byte
	if params.Extension == "mp4" || params.Extension == "webm" {
		html_file , _ = ioutil.ReadFile( "./v1/html/video.html" )
	} else if params.Extension == "mp3" || params.Extension == "wav" || params.Extension == "ogg" {
		html_file , _ = ioutil.ReadFile( "./v1/html/audio.html" )
	}
	html = string( html_file )
	html = strings.ReplaceAll(html, "{{SESSION_KEY}}", params.SessionKey)
	html = strings.ReplaceAll(html, "{{FILES_PREFIX}}", params.FilesURLPrefix)
	html = strings.ReplaceAll(html, "{{LIBRARY_KEY}}", params.LibraryKey)
	html = strings.ReplaceAll(html, "{{SESSION_ID}}", params.SessionID)
	html = strings.ReplaceAll(html, "{{TIME_STR}}", params.TimeStr)
	html = strings.ReplaceAll(html, "{{UUID}}", params.NextID)
	html = strings.ReplaceAll(html, "{{EXTENSION}}", params.Extension)
	html = strings.ReplaceAll(html, "{{READY_URL}}", params.ReadyURL)
	html = strings.ReplaceAll(html, "{{MEDIA_TYPE}}", params.Type)
	return
}

func GetYouTubePlaylistHTML( params types.GetYouTubePlaylistParams ) ( html string ) {
	html_file , _ := ioutil.ReadFile( "./v1/html/youtube-playlist.html" )
	html = string( html_file )
	html = strings.ReplaceAll(html, "{{SESSION_KEY}}", params.SessionKey)
	html = strings.ReplaceAll(html, "{{LIBRARY_KEY}}", params.LibraryKey)
	html = strings.ReplaceAll(html, "{{PLAYLIST_ID}}", params.PlaylistID)
	html = strings.ReplaceAll(html, "{{SESSION_ID}}", params.SessionID)
	html = strings.ReplaceAll(html, "{{TIME}}", params.Time)
	html = strings.ReplaceAll(html, "{{INDEX}}", params.Index)
	html = strings.ReplaceAll(html, "{{READY_URL}}", params.ReadyURL)
	html = strings.ReplaceAll(html, "{{TYPE}}", params.Type)
	return
}