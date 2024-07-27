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
	s.FiberApp.Post( "/update_position" , UpdatePosition( s ) )
	prefix := s.FiberApp.Group( s.Config.URLS.Prefix )
	prefix.Use( PublicLimter )
	prefix.Get( "/:uuid.:ext" , ServeFile( s ) )

	// youtube
	youtube := prefix.Group( "/youtube" )
	youtube.Get( "/:library_key/:session_id" , YouTubeSessionHTMLPlayer( s ) )

	// local library
	library := prefix.Group( "/library" )
	library.Get( "/get/entries" , LibraryGetEntries( s ) )
	// library-session
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