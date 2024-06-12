package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	redis "github.com/redis/go-redis/v9"
	"github.com/0187773933/FileServer-Media/v1/types"
	"github.com/0187773933/FileServer-Media/v1/utils"
	circular_set "github.com/0187773933/RedisCircular/v1/set"
)

var db *redis.Client

func setup_db( config *types.ConfigFile ) {
	fmt.Println( "setting up database connection..." )
	db = redis.NewClient( &redis.Options{
		Addr:     config.RedisAddress,
		Password: config.RedisPassword,
		DB:       config.RedisDBNumber,
	})
	var ctx = context.Background()
	ping_result, err := db.Ping( ctx ).Result()
	fmt.Printf( "db connected: ping = %s\n", ping_result )
	if err != nil {
		panic( err )
	}
}

func main() {

	defer utils.SetupStackTraceReport()

	// 1.) Load Config
	// utils.GenerateNewKeys()
	fmt.Println( "loading config..." )
	var config_file_path string
	if len( os.Args ) > 1 {
		config_file_path, _ = filepath.Abs( os.Args[ 1 ] )
	} else {
		config_file_path, _ = filepath.Abs( "./config.yaml" )
		if _, err := os.Stat( config_file_path ); os.IsNotExist( err ) {
			config_file_path, _ = filepath.Abs( "./SAVE_FILES/config.yaml" )
			if _, err := os.Stat( config_file_path ); os.IsNotExist( err ) {
				panic( "config file not found" )
			}
		}
	}
	fmt.Println( "config file path:", config_file_path )
	config := utils.ParseConfig( config_file_path )
	fmt.Println( "config loaded:", config )

	// 2.) Connect Redis DB
	fmt.Println( "connecting to redis..." )
	setup_db( &config )

	// 3.) Read-In Local Library Store Files, and Store them In Redis
	fmt.Println( "reading and storing local library files in redis..." )
	for _, library_entry := range config.Library {
		flattened_library_entries := utils.GetFlattenedEntries( library_entry.FilePath )
		var ctx = context.Background()
		global_circular_key := fmt.Sprintf( "%s.%s", config.LibraryGlobalRedisKey, library_entry.RedisKey )

		// Force Reset
		fmt.Println( "resetting global circular key:", global_circular_key )
		db.Del( ctx, global_circular_key )
		// db.Del( ctx, global_circular_key + ".INDEX" )

		for index, entry := range flattened_library_entries {
			if entry.Path == "" {
				continue
			}
			if entry.ID == "" {
				continue
			}
			fmt.Printf( "index: %d, name: %s, path: %s, id: %s\n", index, entry.Name, entry.Path, entry.ID )
			global_entry_key := fmt.Sprintf( "%s.%s", config.LibraryGlobalRedisKey, entry.ID )

			// Minimum need the path, could json blob store here instead
			db.Set( ctx, global_entry_key, entry.Path, 0 )
			// So these are setting up "sessions"
			// a "session" here is just an ephemeral copy of the circular set's index tracking
			fmt.Println( "adding" , global_circular_key, entry.ID )
			circular_set.Add( db, global_circular_key, entry.ID )
		}
	}

	// 4.) Start Server
	fmt.Println( "starting server..." )
	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: config.AllowOrigins ,
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, k",
	}))
	// Original route to serve files
	match_url := fmt.Sprintf( "/%s/:uuid.:ext", config.FilesURLPrefix )
	app.Get( match_url, func( c *fiber.Ctx ) error {
		uuid := c.Params( "uuid" )
		var ctx = context.Background()
		global_entry_key := fmt.Sprintf( "%s.%s", config.LibraryGlobalRedisKey, uuid )
		path, err := db.Get( ctx, global_entry_key ).Result()
		fmt.Println( "serving file - uuid:", uuid, "global entry key:", global_entry_key, "path:", path )
		if err != nil {
			return c.Status( fiber.StatusNotFound ).SendString( "file not found" )
		}
		return c.SendFile( path, false )
	})

	app.Get( "/:library_key/:session_id/reset", func( c *fiber.Ctx ) error {
		library_key := c.Params( "library_key" )
		session_id := c.Params( "session_id" )
		var ctx = context.Background()
		session_key := fmt.Sprintf( "%s.SESSIONS.%s.%s*", config.LibraryGlobalRedisKey, library_key, session_id )
		fmt.Println( "resetting session - library key:", library_key, "session id:", session_id, "session key:", session_key )
		utils.DeleteKeysWithPattern( ctx, db, session_key )
		return c.SendStatus( fiber.StatusOK )
	})

	app.Get( "/:library_key/:session_id/total" , func( c *fiber.Ctx ) error {
		var ctx = context.Background()
		library_key := c.Params( "library_key" )
		session_id := c.Params( "session_id" )
		global_key := fmt.Sprintf( "%s.%s" , config.LibraryGlobalRedisKey , library_key )
		total , _ := db.ZCard( ctx , global_key ).Result()
		return c.JSON( fiber.Map{ "library_key": library_key , "session_id": session_id , "total": total } )
	})

	app.Get( "/:library_key/:session_id/index" , func( c *fiber.Ctx ) error {
		var ctx = context.Background()
		library_key := c.Params( "library_key" )
		session_id := c.Params( "session_id" )
		session_key := fmt.Sprintf( "%s.SESSIONS.%s.%s" , config.LibraryGlobalRedisKey , library_key , session_id )
		session_key_index := fmt.Sprintf( "%s.INDEX" , session_key )
		session_index := db.Get( ctx , session_key_index ).Val()
		return c.JSON( fiber.Map{ "library_key": library_key , "session_id": session_id , "index": session_index } )
	})

	app.Get( "/:library_key/:session_id/set/index/:index" , func( c *fiber.Ctx ) error {
		var ctx = context.Background()
		library_key := c.Params( "library_key" )
		session_id := c.Params( "session_id" )
		index := c.Params( "index" )
		session_key := fmt.Sprintf( "%s.SESSIONS.%s.%s" , config.LibraryGlobalRedisKey , library_key , session_id )
		session_key_index := fmt.Sprintf( "%s.INDEX" , session_key )
		db.Set( ctx , session_key_index , index , 0 )
		return c.JSON( fiber.Map{ "library_key": library_key , "session_id": session_id , "index": index } )
	})

	// Endpoint to update playback position
	app.Post( "/update_position", func( c *fiber.Ctx ) error {
		session_key_header := c.Get( "k" )
		if session_key_header != config.SessionKey {
			return c.Status( fiber.StatusUnauthorized ).SendString( "unauthorized" )
		}
		var req types.UpdatePositionRequest
		if err := c.BodyParser( &req ); err != nil {
			return c.Status( fiber.StatusBadRequest ).SendString( "invalid request" )
		}
		var ctx = context.Background()
		session_key := fmt.Sprintf( "%s.SESSIONS.%s.%s", config.LibraryGlobalRedisKey, req.LibraryKey, req.SessionID )
		session_time_key := fmt.Sprintf( "%s.%s.TIME", session_key, req.UUID )
		if req.Finished {
			session_finished_key := fmt.Sprintf( "%s.%s.FINISHED", session_key, req.UUID )
			db.Set( ctx, session_finished_key, true, 0 )
		}
		fmt.Println( session_time_key, req )
		db.Set( ctx, session_time_key, req.Position, 0 )
		return c.SendStatus( fiber.StatusOK )
	})

	// Endpoint to fetch playback position
	app.Get( "/position" , func( c *fiber.Ctx ) error {
		// TODO
		return c.JSON( fiber.Map{ "position": 0 } )
	})

	app.Get( "/:library_key/:session_id/previous", func( c *fiber.Ctx ) error {
		var ctx = context.Background()
		library_key := c.Params( "library_key" )
		session_id := c.Params( "session_id" )
		ready_url := c.Query( "ready_url" )
		session_key := fmt.Sprintf( "%s.SESSIONS.%s.%s", config.LibraryGlobalRedisKey, library_key, session_id )
		global_key := fmt.Sprintf( "%s.%s", config.LibraryGlobalRedisKey, library_key )
		global_key_index := fmt.Sprintf( "%s.INDEX", global_key )
		session_key_index_key := fmt.Sprintf( "%s.INDEX", session_key )

		fmt.Println( "previous - library key:", library_key, "session id:", session_id, "session key:", session_key, "global key:", global_key , "ready url:" , ready_url )

		// 2.) Set Global Version of Session Clone to Sessions Current Index
		session_index := db.Get( ctx, session_key_index_key ).Val()
		if session_index == "" {
			session_index = "0"
			db.Set( ctx, session_key_index_key, session_index, 0 )
		}
		fmt.Println( "session index:", session_index )
		db.Set( ctx, global_key_index, session_index, 0 )

		// 3.) Get Previous Global Version
		previous_global_version := circular_set.Previous( db, global_key )
		previous_id := previous_global_version
		global_version_index := db.Get( ctx, global_key_index ).Val()
		fmt.Println( "new index" , global_version_index )
		db.Set( ctx, session_key_index_key, global_version_index, 0 )

		// Reset FINISHED status
		finished_key := fmt.Sprintf( "%s.%s.FINISHED", session_key, previous_id )
		db.Set( ctx, finished_key, false, 0 )

		fmt.Println( "previous id:", previous_id )

		path_key := fmt.Sprintf( "%s.%s", config.LibraryGlobalRedisKey, previous_id )
		path, _ := db.Get( ctx, path_key ).Result()
		extension := filepath.Ext( path )[1:]

		html := utils.GetMediaHTML( config.SessionKey, config.FilesURLPrefix, library_key, session_id, "0", previous_id, extension , ready_url )
		c.Type( "html" )
		return c.SendString( html )
	} )

	app.Get( "/:library_key/:session_id/next", func( c *fiber.Ctx ) error {
		var ctx = context.Background()
		library_key := c.Params( "library_key" )
		session_id := c.Params( "session_id" )
		ready_url := c.Query( "ready_url" )
		session_key := fmt.Sprintf( "%s.SESSIONS.%s.%s", config.LibraryGlobalRedisKey, library_key, session_id )
		global_key := fmt.Sprintf( "%s.%s", config.LibraryGlobalRedisKey, library_key )
		global_key_index := fmt.Sprintf( "%s.INDEX", global_key )
		session_key_index_key := fmt.Sprintf( "%s.INDEX", session_key )

		fmt.Println( "next - library key:", library_key, "session id:", session_id, "session key:", session_key, "global key:", global_key , "ready url:" , ready_url )

		// 2.) Set Global Version of Session Clone to Sessions Current Index
		session_index := db.Get( ctx, session_key_index_key ).Val()
		if session_index == "" {
			session_index = "0"
			db.Set( ctx, session_key_index_key, session_index, 0 )
		}
		fmt.Println( "session index:", session_index )
		db.Set( ctx, global_key_index, session_index, 0 )

		// 3.) Get Next Global Version
		next_global_version := circular_set.Next( db, global_key )
		next_id := next_global_version
		global_version_index := db.Get( ctx, global_key_index ).Val()
		fmt.Println( "new index" , global_version_index )
		db.Set( ctx, session_key_index_key, global_version_index, 0 )

		// Reset FINISHED status
		finished_key := fmt.Sprintf( "%s.%s.FINISHED", session_key, next_id )
		db.Set( ctx, finished_key, false, 0 )

		fmt.Println( "next id:", next_id )

		path_key := fmt.Sprintf( "%s.%s", config.LibraryGlobalRedisKey, next_id )
		path, _ := db.Get( ctx, path_key ).Result()
		extension := filepath.Ext( path )[1:]

		html := utils.GetMediaHTML( config.SessionKey, config.FilesURLPrefix, library_key, session_id, "0", next_id, extension , ready_url )
		c.Type( "html" )
		return c.SendString( html )
	})


	// Serve HTML player
	app.Get( "/:library_key/:session_id" , func( c *fiber.Ctx ) error {

		var ctx = context.Background()

		// 1.) Setup
		library_key := c.Params( "library_key" )
		session_id := c.Params( "session_id" )
		ready_url := c.Query( "ready_url" )
		session_key := fmt.Sprintf( "%s.SESSIONS.%s.%s", config.LibraryGlobalRedisKey, library_key, session_id )
		global_key := fmt.Sprintf( "%s.%s", config.LibraryGlobalRedisKey, library_key )
		global_key_index := fmt.Sprintf( "%s.INDEX", global_key )
		session_key_index_key := fmt.Sprintf( "%s.INDEX", session_key )

		fmt.Println( "serve html player - library key:", library_key, "session id:", session_id, "session key:", session_key, "global key:", global_key , "ready url:" , ready_url )

		// 2.) Set Global Version of Session Clone to Sessions Current Index
		session_index := db.Get( ctx, session_key_index_key ).Val()
		if session_index == "" {
			session_index = "0"
			db.Set( ctx, session_key_index_key, session_index, 0 )
		}
		db.Set( ctx, global_key_index, session_index, 0 )

		// 3.) Get Current Global Version
		current_global_version := circular_set.Current( db, global_key )
		next_id := current_global_version
		next_index := session_index

		// 4.) If Current Global Version is Finished, Get Next Global Version and Update Session Index
		finished_key := fmt.Sprintf( "%s.%s.FINISHED", session_key, next_id )
		finished := utils.RedisGetBool( ctx, db, finished_key )
		time_str := "0"
		if finished {
			fmt.Println( "finished:", finished )
			next_id = circular_set.Next( db, global_key )
			next_index = db.Get( ctx, global_key_index ).Val()
			db.Set( ctx, session_key_index_key, next_index, 0 )
		} else {
			session_time_key := fmt.Sprintf( "%s.%s.TIME", session_key, next_id )
			time_str, _ = db.Get( ctx, session_time_key ).Result()
		}

		fmt.Println( "next id:", next_id )
		fmt.Println( "next index:", next_index )
		fmt.Println( "time:", time_str )

		path_key := fmt.Sprintf( "%s.%s", config.LibraryGlobalRedisKey, next_id )
		path, _ := db.Get( ctx, path_key ).Result()
		extension := filepath.Ext( path )[1:]

		html := utils.GetMediaHTML( config.SessionKey, config.FilesURLPrefix, library_key, session_id, time_str, next_id, extension , ready_url )
		c.Type( "html" )
		return c.SendString( html )
	})

	app.Get( "/:library_key/:session_id/:index" , func( c *fiber.Ctx ) error {
		var ctx = context.Background()
		library_key := c.Params( "library_key" )
		session_id := c.Params( "session_id" )
		ready_url := c.Query( "ready_url" )
		index := c.Params( "index" )
		global_key := fmt.Sprintf( "%s.%s", config.LibraryGlobalRedisKey, library_key )
		global_key_index := fmt.Sprintf( "%s.INDEX" , global_key )

		db.Set( ctx, global_key_index , index , 0 )
		current_global_version := circular_set.Current( db, global_key )

		time_str := "0"
		fmt.Println( "loading id:", current_global_version )
		fmt.Println( "loading index:", index )
		fmt.Println( "loading time:", time_str )

		path_key := fmt.Sprintf( "%s.%s", config.LibraryGlobalRedisKey, current_global_version )
		path, _ := db.Get( ctx, path_key ).Result()
		extension := filepath.Ext( path )[1:]

		html := utils.GetMediaHTML( config.SessionKey, config.FilesURLPrefix, library_key, session_id, time_str, current_global_version, extension , ready_url )
		c.Type( "html" )
		return c.SendString( html )
	})


	log.Fatal( app.Listen( fmt.Sprintf( ":%s", config.ServerPort ) ) )
}
