package main

import (
	"os"
	"fmt"
	"time"
	"os/signal"
	"syscall"
	"strings"
	"context"
	bolt "github.com/boltdb/bolt"
	redis "github.com/redis/go-redis/v9"
	logger "github.com/0187773933/Logger/v1/logger"
	server_utils "github.com/0187773933/GO_SERVER/v1/utils"
	utils "github.com/0187773933/FileServer-Media/v1/utils"
	server "github.com/0187773933/GO_SERVER/v1/server"
	fiber "github.com/gofiber/fiber/v2"
	cors "github.com/gofiber/fiber/v2/middleware/cors"
	routes "github.com/0187773933/FileServer-Media/v1/routes"
)

var s server.Server
var DB *bolt.DB
var REDIS *redis.Client

func SetupCloseHandler() {
	c := make( chan os.Signal )
	signal.Notify( c , os.Interrupt , syscall.SIGTERM , syscall.SIGINT )
	go func() {
		<-c
		logger.Log.Println( "\r- Ctrl+C pressed in Terminal" )
		DB.Close()
		REDIS.Close()
		logger.Log.Printf( "Shutting Down %s Server" , s.Config.Name )
		s.FiberApp.Shutdown()
		logger.CloseDB()
		os.Exit( 0 )
	}()
}

func main() {

	// start
	config := server_utils.GetConfig()
	// server_utils.GenerateNewKeysWrite( &config )
	defer server_utils.SetupStackTraceReport()
	logger.New( &config.Log )
	DB , _ = bolt.Open( config.Bolt.Path , 0600 , &bolt.Options{ Timeout: ( 3 * time.Second ) } )
	s = server.New( &config , logger.Log , DB )

	REDIS = redis.NewClient( &redis.Options{
		Addr: fmt.Sprintf( "%s:%s" , config.Redis.Host , config.Redis.Port ) ,
		Password: config.Redis.Password ,
		DB: config.Redis.Number ,
	})
	_ , err := REDIS.Ping( context.Background() ).Result()
	if err != nil { panic( err ) }
	s.REDIS = REDIS

	// custom
	utils.ImportLibrarySaveFilesInToRedis( &s )
	allow_origins_string := strings.Join( config.AllowOrigins , "," )
	s.STORE[ "session_key" ] = s.ConfigGenericGet( "creds" , "session_key" ).( string )
	s.STORE[ "google_key" ] = s.ConfigGenericGet( "creds" , "google_key" ).( string )
	s.FiberApp.Use( cors.New( cors.Config{
		AllowOrigins: allow_origins_string ,
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS" ,
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, k" ,
	}))
	s.FiberApp.Use( func( c *fiber.Ctx ) error {
		c.Set( "Cache-Control" , "no-store, no-cache, must-revalidate, proxy-revalidate" )
		c.Set( "Pragma" , "no-cache" )
		c.Set( "Expires" , "0" )
		return c.Next()
	})

	// wait
	routes.SetupPublicRoutes( &s )
	routes.SetupAdminRoutes( &s )
	SetupCloseHandler()
	s.Start()
}