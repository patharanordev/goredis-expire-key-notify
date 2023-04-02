package main

// Ref.
// - SET command options : https://redis.io/commands/set/

import (
	"net/http"

	"github.com/labstack/echo/v4"

	RedisCache "godis/rediscache"
)

func main() {

	e := echo.New()
	RedisCache.NewGoRedis()
	RedisCache.EnableKeyNotify()

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})

	e.POST("/checkout", func(c echo.Context) error {
		req := new(RedisCache.ReqCheckout)

		if err := c.Bind(req); err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}

		res := RedisCache.Checkout(req.Id, req.Expire)
		return c.JSON(res.Status, res)
	})

	e.GET("/checkout/:id", func(c echo.Context) error {
		id := c.Param("id")
		res := RedisCache.GetOrder("checkout", id)
		return c.JSON(res.Status, res)
	})

	e.Logger.Fatal(e.Start(":1323"))
}
