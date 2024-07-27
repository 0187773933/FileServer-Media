package routes

import (
	"context"
	fiber "github.com/gofiber/fiber/v2"
	server "github.com/0187773933/GO_SERVER/v1/server"
	// utils "github.com/0187773933/FileServer-Media/v1/utils"
	// circular_set "github.com/0187773933/RedisCircular/v1/set"
)

func LibraryGetEntries( s *server.Server ) fiber.Handler {
	return func( c *fiber.Ctx ) error {
		entries := s.REDIS.SMembers( context.Background() , s.Config.Redis.Prefix + ".LIBRARY" ).Val()
		return c.JSON( fiber.Map{
			"entries": entries ,
		})
	}
}
