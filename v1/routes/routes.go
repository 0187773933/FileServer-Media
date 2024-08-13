package routes

import (
	fiber "github.com/gofiber/fiber/v2"
	server "github.com/0187773933/GO_SERVER/v1/server"
)

func SetupPublicRoutes( s *server.Server ) {
	s.FiberApp.Get( "/" , PublicLimter , func( c *fiber.Ctx ) error {
		return c.JSON( fiber.Map{
			"result": true ,
			"url": "/" ,
		})
	})
	s.FiberApp.Get( "/twitch" , func( c *fiber.Ctx ) error {
		c.Set( "Content-Type" , "text/html" )
		return c.SendFile( "./v1/html/twitch.html" )
	})
	s.FiberApp.Post( "/update_position" , PublicLimter , UpdatePosition( s ) )
	prefix := s.FiberApp.Group( s.Config.URLS.Prefix )
	prefix.Get( "/:uuid.:ext" , UUIDFileLimter , ServeFile( s ) )

	// youtube
	youtube := prefix.Group( "/youtube" )
	youtube.Get( "/:library_key/:session_id" , YouTubeSessionHTMLPlayer( s ) )

	// local library
	library := prefix.Group( "/library" )
	library.Get( "/get/entries" , LibraryGetEntries( s ) )
	// library-session
	prefix.Use( PublicLimter )
	prefix.Get( "/:library_key/:session_id/reset" , SessionReset( s ) )
	prefix.Get( "/:library_key/:session_id/total" , SessionTotal( s ) )
	prefix.Get( "/:library_key/:session_id/index" , SessionIndex( s ) )
	prefix.Get( "/:library_key/:session_id/set/index/:index" , SessionSetIndex( s ) )
	prefix.Get( "/:library_key/:session_id/previous" , SessionPrevious( s ) )
	prefix.Get( "/:library_key/:session_id/next" , SessionNext( s ) )
	prefix.Get( "/:library_key/:session_id" , SessionHTMLPlayer( s ) ) // HTML Player
	prefix.Get( "/:library_key/:session_id/:index" , SessionHTMLPlayerAtIndex( s ) ) // HTML Player at Session Index ?
}

func SetupAdminRoutes( s *server.Server ) {
	admin := s.FiberApp.Group( s.Config.URLS.AdminPrefix )
	admin.Use( s.ValidateAdminMW )
	admin.Get( "/add/youtube/playlist/:playlist_id" , YouTubeAddPlaylist( s ) )
}