package routes

import (
	"fmt"
	// "strconv"
	"time"
	"context"
	"path/filepath"
	fiber "github.com/gofiber/fiber/v2"
	server "github.com/0187773933/GO_SERVER/v1/server"
	rate_limiter "github.com/gofiber/fiber/v2/middleware/limiter"
	utils "github.com/0187773933/FileServer-Media/v1/utils"
	types "github.com/0187773933/FileServer-Media/v1/types"
	circular_set "github.com/0187773933/RedisCircular/v1/set"
	// logger "github.com/0187773933/Logger/v1/logger"
)

func PublicMaxedOut( c *fiber.Ctx ) error {
	ip_address := c.IP()
	log_message := fmt.Sprintf( "%s === %s === %s === PUBLIC RATE LIMIT REACHED !!!" , ip_address , c.Method() , c.Path() );
	fmt.Println( log_message )
	c.Set( "Content-Type" , "text/html" )
	return c.SendString( "<html><h1>loading ...</h1><script>setTimeout(function(){ window.location.reload(1); }, 6000);</script></html>" )
}
var PublicLimter = rate_limiter.New( rate_limiter.Config{
	Max: 3 ,
	Expiration: 1 * time.Second ,
	KeyGenerator: func( c *fiber.Ctx ) string {
		return c.Get( "x-forwarded-for" )
	} ,
	LimitReached: PublicMaxedOut ,
	LimiterMiddleware: rate_limiter.SlidingWindow{} ,
})

var UUIDFileLimter = rate_limiter.New( rate_limiter.Config{
	Max: 10 ,
	Expiration: 1 * time.Second ,
	KeyGenerator: func( c *fiber.Ctx ) string {
		return c.Get( "x-forwarded-for" )
	} ,
	LimitReached: PublicMaxedOut ,
	LimiterMiddleware: rate_limiter.SlidingWindow{} ,
})

// https://github.com/gofiber/fiber/blob/0592e01382d2dd011980b7687023957f63025f7b/ctx.go#L74
// https://github.com/gofiber/fiber/blob/v2/ctx.go#L1732
func ServeFile( s *server.Server ) fiber.Handler {
	return func( c *fiber.Ctx ) error {
		uuid := c.Params( "uuid" )
		var ctx = context.Background()
		global_entry_key := fmt.Sprintf( "%s.%s" , s.Config.Redis.Prefix , uuid )
		path , err := s.REDIS.Get( ctx , global_entry_key ).Result()
		range_header := c.Get( "Range" )
		fmt.Println( "serving file - uuid:" , uuid , "global entry key:" , global_entry_key , "path:" , path , "range:" , range_header )
		if err != nil {
			fmt.Println( "error:" , err )
			return c.Status( fiber.StatusNotFound ).SendString( "file not found" )
		}
		if utils.IsURL( path ) {
			fmt.Println( "detected url , sending that" )
			return c.Redirect( path , fiber.StatusFound )
		}
		return c.SendFile( path , false )
	}
}

func UpdatePosition( s *server.Server ) fiber.Handler {
	return func( c *fiber.Ctx ) error {
		session_key_header := c.Get( "k" )
		if session_key_header != s.STORE[ "session_key" ] {
			return c.Status( fiber.StatusUnauthorized ).SendString( "unauthorized" )
		}
		var req types.UpdatePositionRequest
		if err := c.BodyParser( &req ); err != nil {
			fmt.Println( err )
			return c.Status( fiber.StatusBadRequest ).SendString( "invalid request" )
		}
		var ctx = context.Background()

		// if youtube
		if req.Type == "youtube-playlist" {
			session_key := fmt.Sprintf( "%s.SESSIONS.%s.%s" , s.Config.Redis.Prefix , req.LibraryKey , req.SessionID )
			session_key_index_key := fmt.Sprintf( "%s.INDEX" , session_key )
			session_key_time_key := fmt.Sprintf( "%s.TIME" , session_key )
			s.REDIS.Set( ctx , session_key_index_key , req.YouTubePlaylistIndex , 0 )
			s.REDIS.Set( ctx , session_key_time_key , req.Position , 0 )
			return c.Status( fiber.StatusOK ).SendString( "ok" )
		} else { // local1
			session_key := fmt.Sprintf( "%s.SESSIONS.%s.%s" , s.Config.Redis.Prefix , req.LibraryKey , req.SessionID )
			session_key_index_key := fmt.Sprintf( "%s.INDEX" , session_key )
			session_time_key := fmt.Sprintf( "%s.%s.TIME" , session_key , req.UUID )
			global_key := fmt.Sprintf( "%s.%s" , s.Config.Redis.Prefix , req.LibraryKey )
			global_key_index := fmt.Sprintf( "%s.INDEX" , global_key )
			result := fiber.Map{
				"result": true ,
			}
			if req.Finished {
				session_finished_key := fmt.Sprintf( "%s.%s.FINISHED" , session_key , req.UUID )
				s.REDIS.Set( ctx, session_finished_key , true , 0 )


				// 2.) Set Global Version of Session Clone to Sessions Current Index
				session_index := s.REDIS.Get( ctx , session_key_index_key ).Val()
				if session_index == "" {
					fmt.Println( "New Session , Setting Index to 0" )
					session_index = "0"
					s.REDIS.Set( ctx , session_key_index_key , session_index , 0 )
				}
				s.REDIS.Set( ctx , global_key_index , session_index , 0 )

				// 3.) Get Next Global Version
				next_global_version := circular_set.Next( s.REDIS, global_key )
				next_id := next_global_version
				global_version_index := s.REDIS.Get( ctx, global_key_index ).Val()
				// fmt.Println( "new index" , global_version_index )
				s.REDIS.Set( ctx , session_key_index_key , global_version_index , 0 )

				// Reset FINISHED status
				finished_key := fmt.Sprintf( "%s.%s.FINISHED" , session_key , next_id )
				s.REDIS.Set( ctx , finished_key , false , 0 )

				path_key := fmt.Sprintf( "%s.%s" , s.Config.Redis.Prefix , next_id )
				path , _ := s.REDIS.Get( ctx , path_key ).Result()
				extension_test := filepath.Ext( path )
				if extension_test == "" {
					return c.JSON( result )
				}
				result[ "next_uuid" ] = next_id
				result[ "next_extension" ] = extension_test[ 1: ]
			}
			fmt.Println( session_time_key , req )
			s.REDIS.Set( ctx , session_time_key , req.Position , 0 )
			return c.JSON( result )
		}
	}
}
