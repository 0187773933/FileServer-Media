package routes

import (
	"fmt"
	"context"
	// uuid "github.com/satori/go.uuid"
	fiber "github.com/gofiber/fiber/v2"
	server "github.com/0187773933/GO_SERVER/v1/server"
	utils "github.com/0187773933/FileServer-Media/v1/utils"
	types "github.com/0187773933/FileServer-Media/v1/types"
	circular_set "github.com/0187773933/RedisCircular/v1/set"
	"github.com/gosimple/slug"
	"reflect"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)


func fetchPlaylistItems(service *youtube.Service, playlistID, pageToken string, allItems *[]*youtube.PlaylistItem) {
	call := service.PlaylistItems.List([]string{"snippet"}).PlaylistId(playlistID).MaxResults(50)
	if pageToken != "" {
		call = call.PageToken(pageToken)
	}

	response, err := call.Do()
	if err != nil {
		fmt.Printf("Error making API call: %v", err)
	}
	fmt.Println( response )
	fmt.Println( reflect.TypeOf( response ) )
	*allItems = append(*allItems, response.Items...)

	if response.NextPageToken != "" {
		fetchPlaylistItems(service, playlistID, response.NextPageToken, allItems)
	}
}

func getPlaylistDetails( service *youtube.Service , playlistID string ) ( result *youtube.PlaylistSnippet ) {
	call := service.Playlists.List([]string{"snippet"}).Id(playlistID)
	response, err := call.Do()
	if err != nil {
		return
	}

	if len(response.Items) == 0 {
		fmt.Errorf("no playlist found with ID %s", playlistID)
		return
	}
	result = response.Items[0].Snippet
	return
}

type YouTubePlaylist struct {
	Name string `json:"name"`
	Items []*youtube.PlaylistItem `json:"items"`
}
func YouTubeGetVideosInPlaylist( api_key string , playlist_id string ) ( result YouTubePlaylist ) {
	ctx := context.Background()
	service , err := youtube.NewService( ctx , option.WithAPIKey( api_key ) )
	if err != nil {
		fmt.Printf( "Error creating YouTube service: %v" , err )
	}
	var allItems []*youtube.PlaylistItem
	details := getPlaylistDetails( service , playlist_id )
	fetchPlaylistItems( service , playlist_id , "" , &allItems )
	result.Name = slug.Make( details.Title )
	result.Items = allItems
	return
}

func YouTubeAddPlaylist( s *server.Server ) fiber.Handler {
	return func( c *fiber.Ctx ) error {
		var ctx = context.Background()
		playlist_id := c.Params( "playlist_id" )
		playlist := YouTubeGetVideosInPlaylist( s.STORE[ "google_key" ] , playlist_id )
		s.REDIS.SAdd( ctx , s.Config.Redis.Prefix + ".LIBRARY" , playlist.Name )
		s.REDIS.Set( ctx , s.Config.Redis.Prefix + ".LIBRARY." + playlist.Name , playlist_id , 0 )
		global_circular_key := fmt.Sprintf( "%s.YOUTUBE.PLAYLISTS.%s" , s.Config.Redis.Prefix , playlist_id )
		s.REDIS.Del( ctx , global_circular_key ) // force reset
		s.REDIS.Del( ctx , global_circular_key + ".INDEX" )
		for _ , item := range playlist.Items {
			id := item.Snippet.ResourceId.VideoId
			// title := item.Snippet.Title
			// uri := fmt.Sprintf( "youtube::%s" , id )
			// x_uuid := uuid.NewV4().String()
			// fmt.Printf( "Title: %s, Video ID: %s\n" , title , id )
			// global_entry_key := fmt.Sprintf( "%s.%s" , s.Config.Redis.Prefix , x_uuid )
			// s.REDIS.Set( ctx , global_entry_key , uri , 0 )
			// So these are setting up "sessions"
			// a "session" here is just an ephemeral copy of the circular set's index tracking
			// fmt.Println( "adding" , global_circular_key , x_uuid , uri )
			circular_set.Add( s.REDIS , global_circular_key , id )
		}
		return c.JSON( fiber.Map{
			"playlist_id": playlist_id ,
			"playlist": playlist ,
		})
	}
}

func YouTubeSessionHTMLPlayer( s *server.Server ) fiber.Handler {
	return func( c *fiber.Ctx ) error {
		var ctx = context.Background()
		library_key := c.Params( "library_key" )
		playlist_id := s.REDIS.Get( ctx , s.Config.Redis.Prefix + ".LIBRARY." + library_key ).Val()
		session_id := c.Params( "session_id" )
		ready_url := c.Query( "ready_url" )
		session_key := fmt.Sprintf( "%s.SESSIONS.%s.%s" , s.Config.Redis.Prefix , library_key , session_id )
		session_key_index_key := fmt.Sprintf( "%s.INDEX" , session_key )
		session_key_time_key := fmt.Sprintf( "%s.TIME" , session_key )

		current_index := s.REDIS.Get( ctx , session_key_index_key ).Val()
		current_time := s.REDIS.Get( ctx , session_key_time_key ).Val()

		options := types.GetYouTubePlaylistParams{
			SessionKey: s.STORE[ "session_key" ] ,
			LibraryKey: library_key ,
			PlaylistID: playlist_id ,
			SessionID: session_id ,
			Time: current_time ,
			Index: current_index ,
			ReadyURL: ready_url ,
		}
		html := utils.GetYouTubePlaylistHTML( options )
		c.Set( "Content-Type" , "text/html" )
		return c.SendString( html )
	}
}