package routes

import (
	"fmt"
	// "strconv"
	"strings"
	"time"
	"context"
	"path/filepath"
	// net_url "net/url"
	// bolt_api "github.com/bolts.db/bolt"
	// encryption "github.com/0187773933/encryption/v1/encryption"
	redis "github.com/redis/go-redis/v9"
	fiber "github.com/gofiber/fiber/v2"
	server "github.com/0187773933/GO_SERVER/v1/server"
	// rate_limiter "github.com/gofiber/fiber/v2/middleware/limiter"
	types "github.com/0187773933/FileServer-Media/v1/types"
	utils "github.com/0187773933/FileServer-Media/v1/utils"
	circular_set "github.com/0187773933/RedisCircular/v1/set"
	// types "github.com/0187773933/BLANK_SERVER/v1/types"
	// logger "github.com/0187773933/Logger/v1/logger"
)

func SetDebounceFlag( db *redis.Client , ctx context.Context , key string , ttl time.Duration ) bool {
	if db.SetNX( ctx , key , true , ttl ).Val() {
		return true
	}
	return false
}

func SessionReset( s *server.Server ) fiber.Handler {
	return func( c *fiber.Ctx ) error {
		library_key := c.Params( "library_key" )
		session_id := c.Params( "session_id" )
		var ctx = context.Background()
		session_key := fmt.Sprintf( "%s.SESSIONS.%s.%s*", s.Config.Redis.Prefix, library_key, session_id )
		fmt.Println( "resetting session - library key:", library_key, "session id:", session_id, "session key:", session_key )
		utils.DeleteKeysWithPattern( ctx , s.REDIS , session_key )
		return c.SendStatus( fiber.StatusOK )
	}
}

func SessionTotal( s *server.Server ) fiber.Handler {
	return func( c *fiber.Ctx ) error {
		var ctx = context.Background()
		library_key := c.Params( "library_key" )
		session_id := c.Params( "session_id" )
		global_key := fmt.Sprintf( "%s.%s" , s.Config.Redis.Prefix , library_key )
		total , _ := s.REDIS.ZCard( ctx , global_key ).Result()
		return c.JSON( fiber.Map{ "library_key": library_key , "session_id": session_id , "total": total } )
	}
}

func SessionIndex( s *server.Server ) fiber.Handler {
	return func( c *fiber.Ctx ) error {
		var ctx = context.Background()
		library_key := c.Params( "library_key" )
		session_id := c.Params( "session_id" )
		session_key := fmt.Sprintf( "%s.SESSIONS.%s.%s" , s.Config.Redis.Prefix , library_key , session_id )
		session_key_index := fmt.Sprintf( "%s.INDEX" , session_key )
		session_index := s.REDIS.Get( ctx , session_key_index ).Val()
		return c.JSON( fiber.Map{ "library_key": library_key , "session_id": session_id , "index": session_index } )
	}
}

func SessionSetIndex( s *server.Server ) fiber.Handler {
	return func( c *fiber.Ctx ) error {
		var ctx = context.Background()
		library_key := c.Params( "library_key" )
		session_id := c.Params( "session_id" )
		index := c.Params( "index" )
		session_key := fmt.Sprintf( "%s.SESSIONS.%s.%s" , s.Config.Redis.Prefix , library_key , session_id )
		session_key_index := fmt.Sprintf( "%s.INDEX" , session_key )
		s.REDIS.Set( ctx , session_key_index , index , 0 )
		return c.JSON( fiber.Map{ "library_key": library_key , "session_id": session_id , "index": index } )
	}
}

func SessionPrevious( s *server.Server ) fiber.Handler {
	return func( c *fiber.Ctx ) error {
		var ctx = context.Background()
		library_key := c.Params( "library_key" )
		session_id := c.Params( "session_id" )
		ready_url := c.Query( "ready_url" )
		session_key := fmt.Sprintf( "%s.SESSIONS.%s.%s" , s.Config.Redis.Prefix , library_key , session_id )
		global_key := fmt.Sprintf( "%s.%s" , s.Config.Redis.Prefix , library_key )
		global_key_index := fmt.Sprintf( "%s.INDEX" , global_key )
		session_key_index_key := fmt.Sprintf( "%s.INDEX" , session_key )

		// fmt.Println( "next - library key:", library_key, "session id:", session_id, "session key:", session_key, "global key:", global_key , "ready url:" , ready_url )
		debounce_key := fmt.Sprintf( "debounce:%s:%s:previous" , library_key , session_id )

		// Set debounce flag with 1 second TTL
		if !SetDebounceFlag( s.REDIS , ctx , debounce_key , 1*time.Second ) {
			return c.Status( fiber.StatusTooManyRequests ).SendString( "Too many requests" )
		}

		// 2.) Set Global Version of Session Clone to Sessions Current Index
		session_index := s.REDIS.Get( ctx, session_key_index_key ).Val()
		if session_index == "" {
			fmt.Println( "New Session , Setting Index to 0" )
			session_index = "0"
			s.REDIS.Set( ctx , session_key_index_key, session_index, 0 )
		}
		// fmt.Println( "setting global index:" , session_index )
		s.REDIS.Set( ctx , global_key_index , session_index, 0 )

		// 3.) Get Next Global Version
		next_global_version := circular_set.Previous( s.REDIS, global_key )
		next_id := next_global_version
		global_version_index := s.REDIS.Get( ctx , global_key_index ).Val()
		// fmt.Println( "new index" , global_version_index )
		s.REDIS.Set( ctx , session_key_index_key , global_version_index , 0 )

		// Reset FINISHED status
		finished_key := fmt.Sprintf( "%s.%s.FINISHED" , session_key , next_id )
		s.REDIS.Set( ctx , finished_key , false , 0 )

		return c.Redirect( fmt.Sprintf( "/%s/%s/%s?ready_url=%s" , s.Config.URLS.Prefix , library_key , session_id , ready_url ) )
	}
}

func SessionNext( s *server.Server ) fiber.Handler {
	return func( c *fiber.Ctx ) error {
		var ctx = context.Background()
		library_key := c.Params( "library_key" )
		session_id := c.Params( "session_id" )
		ready_url := c.Query( "ready_url" )
		session_key := fmt.Sprintf( "%s.SESSIONS.%s.%s" , s.Config.Redis.Prefix, library_key, session_id )
		global_key := fmt.Sprintf( "%s.%s" , s.Config.Redis.Prefix, library_key )
		global_key_index := fmt.Sprintf( "%s.INDEX" , global_key )
		session_key_index_key := fmt.Sprintf( "%s.INDEX" , session_key )

		// fmt.Println( "next - library key:", library_key, "session id:", session_id, "session key:", session_key, "global key:", global_key , "ready url:" , ready_url )
		debounce_key := fmt.Sprintf( "debounce:%s:%s:next" , library_key , session_id )

		// Set debounce flag with 1 second TTL
		if !SetDebounceFlag( s.REDIS , ctx , debounce_key , 1*time.Second ) {
			return c.Status( fiber.StatusTooManyRequests ).SendString( "Too many requests" )
		}

		// 2.) Set Global Version of Session Clone to Sessions Current Index
		session_index := s.REDIS.Get( ctx , session_key_index_key ).Val()
		if session_index == "" {
			fmt.Println( "New Session , Setting Index to 0" )
			session_index = "0"
			s.REDIS.Set( ctx , session_key_index_key , session_index , 0 )
		}
		// fmt.Println( "setting global index:" , session_index )
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

		return c.Redirect( fmt.Sprintf( "/%s/%s/%s?ready_url=%s" , s.Config.URLS.Prefix , library_key , session_id , ready_url ) )

	}
}

func SessionHTMLPlayer( s *server.Server ) fiber.Handler {
	return func( c *fiber.Ctx ) error {
		var ctx = context.Background()

		// 1.) Setup
		library_key := c.Params( "library_key" )
		session_id := c.Params( "session_id" )
		ready_url := c.Query( "ready_url" )
		session_key := fmt.Sprintf( "%s.SESSIONS.%s.%s" , s.Config.Redis.Prefix , library_key , session_id )
		global_key := fmt.Sprintf( "%s.%s" , s.Config.Redis.Prefix , library_key )
		global_key_index := fmt.Sprintf( "%s.INDEX" , global_key )
		session_key_index_key := fmt.Sprintf( "%s.INDEX" , session_key )

		// fmt.Println( "serve html player - library key:", library_key, "session id:", session_id, "session key:", session_key, "global key:", global_key , "ready url:" , ready_url )

		debounce_key := fmt.Sprintf( "debounce:%s:%s:now-playing", library_key, session_id )

		// Set debounce flag with 1 second TTL
		if !SetDebounceFlag( s.REDIS , ctx , debounce_key , 1*time.Second ) {
			return c.Status( fiber.StatusTooManyRequests ).SendString( "Too many requests" )
		}

		// 2.) Set Global Version of Session Clone to Sessions Current Index
		session_index := s.REDIS.Get( ctx , session_key_index_key ).Val()
		// fmt.Println( "Session Index" , session_index )
		if session_index == "" {
			fmt.Println( "New Session , Setting Index to 0" )
			session_index = "0"
			s.REDIS.Set( ctx, session_key_index_key,session_index , 0 )
		}
		s.REDIS.Set( ctx , global_key_index , session_index , 0 )

		// 3.) Get Current Global Version
		current_global_version := circular_set.Current( s.REDIS , global_key )
		next_id := current_global_version
		next_index := session_index

		// 4.) If Current Global Version is Finished, Get Next Global Version and Update Session Index
		finished_key := fmt.Sprintf( "%s.%s.FINISHED" , session_key , next_id )
		finished := utils.RedisGetBool( ctx , s.REDIS , finished_key )
		time_str := "0"
		if finished {
			fmt.Println( "finished:", finished )
			next_id = circular_set.Next( s.REDIS , global_key )
			next_index = s.REDIS.Get( ctx , global_key_index ).Val()
			s.REDIS.Set( ctx , session_key_index_key , next_index , 0 )
		} else {
			session_time_key := fmt.Sprintf( "%s.%s.TIME" , session_key , next_id )
			time_str , _ = s.REDIS.Get( ctx , session_time_key ).Result()
		}

		path_key := fmt.Sprintf( "%s.%s" , s.Config.Redis.Prefix , next_id )
		path , path_err := s.REDIS.Get( ctx , path_key ).Result()

		if path_err != nil {
			return c.SendStatus( fiber.StatusNotFound )
		}
		if path == "" {
			return c.SendStatus( fiber.StatusNotFound )
		}

		var html string
		if strings.HasPrefix( path , "youtube::" ) {
			id := strings.Split( path , "::" )[ 1 ]
			// just send entire playlist
			fmt.Println( "youtube id:" , id )
		} else if strings.HasPrefix( path , "twitch::" ) {
			id := strings.Split( path , "::" )[ 1 ]
			fmt.Println( "twitch id:" , id )
		} else {
			extension_test := filepath.Ext( path )
			if extension_test == "" {
				c.Type( "html" )
				return c.SendString( "asdf" )
			}
			extension := extension_test[ 1: ]
			fmt.Println( "" )
			fmt.Println( "id:" , next_id )
			fmt.Println( "index:" , next_index )
			fmt.Println( "time:" , time_str )
			fmt.Println( "path:" , path )
			fmt.Println( "extension:" , extension )
			options := types.GetMediaHTMLParams{
				SessionKey: s.STORE[ "session_key" ] ,
				FilesURLPrefix: s.Config.URLS.Prefix ,
				LibraryKey: library_key ,
				SessionID: session_id ,
				TimeStr: time_str ,
				NextID: current_global_version ,
				Extension: extension ,
				ReadyURL: ready_url ,
			}
			html = utils.GetMediaHTML( options )
		}
		c.Type( "html" )
		return c.SendString( html )
	}
}

func SessionHTMLPlayerAtIndex( s *server.Server ) fiber.Handler {
	return func( c *fiber.Ctx ) error {
		var ctx = context.Background()
		library_key := c.Params( "library_key" )
		session_id := c.Params( "session_id" )
		ready_url := c.Query( "ready_url" )
		index := c.Params( "index" )
		global_key := fmt.Sprintf( "%s.%s" , s.Config.Redis.Prefix , library_key )
		global_key_index := fmt.Sprintf( "%s.INDEX" , global_key )

		s.REDIS.Set( ctx, global_key_index , index , 0 )
		current_global_version := circular_set.Current( s.REDIS , global_key )

		time_str := "0"
		fmt.Println( "loading id:" , current_global_version )
		fmt.Println( "loading index:" , index )
		fmt.Println( "loading time:" , time_str )

		path_key := fmt.Sprintf( "%s.%s" , s.Config.Redis.Prefix , current_global_version )
		path, _ := s.REDIS.Get( ctx , path_key ).Result()
		extension := filepath.Ext( path )[ 1: ]

		options := types.GetMediaHTMLParams{
			SessionKey: s.STORE[ "session_key" ] ,
			FilesURLPrefix: s.Config.URLS.Prefix ,
			LibraryKey: library_key ,
			SessionID: session_id ,
			TimeStr: time_str ,
			NextID: current_global_version ,
			Extension: extension ,
			ReadyURL: ready_url ,
		}

		html := utils.GetMediaHTML( options )
		c.Type( "html" )
		return c.SendString( html )
	}
}