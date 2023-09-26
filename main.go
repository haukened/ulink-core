/*
 * Created Date: Tue Sep 26 2023
 * Author: David Haukeness david@hauken.us
 * Copyright (c) 2023 David Haukeness
 * Distributed under the terms of the GNU GENERAL PUBLIC LICENSE Version 3.0
 */

package main

import (
	"embed"
	_ "embed"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/monitor"
)

//go:embed dist/* static/*
var embedFS embed.FS

func main() {
	// create a new instance of a fiber http server. See https://docs.gofiber.io/
	app := fiber.New()

	// set up a favicon handler (before logger!) so we don't log all the 404 requests
	app.Use(favicon.New(favicon.ConfigDefault))

	// set up logging
	app.Use(logger.New(logger.Config{
		Next:          nil,
		Done:          nil,
		Format:        "[${time}] [${ip}] ${status} - ${latency} ${method} ${path}\n",
		TimeFormat:    "15:04:05",
		TimeZone:      "Local",
		TimeInterval:  500 * time.Millisecond,
		Output:        os.Stdout,
		DisableColors: false,
	}))

	// set up CORS
	app.Use(cors.New(cors.Config{
		// this needs to be changed later when parameters are added
		AllowOrigins: "*",
		// only allow basic headers to reduce attack surface
		AllowHeaders: "Origin, Content-Type, Accept",
		// only allow HTTP GET method to reduce attack surface
		AllowMethods: fiber.MethodGet,
	}))

	/*
		lets put on our helmet:
		helmet.ConfigDefault = Config{
			XSSProtection:             "0",
			ContentTypeNosniff:        "nosniff",
			XFrameOptions:             "SAMEORIGIN",
			ReferrerPolicy:            "no-referrer",
			CrossOriginEmbedderPolicy: "require-corp",
			CrossOriginOpenerPolicy:   "same-origin",
			CrossOriginResourcePolicy: "same-origin",
			OriginAgentCluster:        "?1",
			XDNSPrefetchControl:       "off",
			XDownloadOptions:          "noopen",
			XPermittedCrossDomain:     "none",
		}
	*/
	app.Use(helmet.New(helmet.ConfigDefault))

	// set up static files
	app.Use(filesystem.New(filesystem.Config{
		Root:       http.FS(embedFS),
		PathPrefix: "static",
		Browse:     false,
		Index:      "/index.html",
		MaxAge:     0,
	}))

	// set up monitoring
	app.Get("/api/monitor", isAllowedMetrics, monitor.New(
		monitor.Config{
			Title:   "µLink Metrics",
			Refresh: 3 * time.Second,
			// serve copies of these locally so we don't have to have internet
			FontURL:    "/roboto.css",
			ChartJsURL: "/Chart.bundle.min.js",
		},
	))

	// set up a rate limiter for everything defined after this block
	// this exempts static files and monitoring
	app.Use(limiter.New(limiter.Config{
		// change this when parameters are added
		Max:        1,
		Expiration: 1 * time.Second,
		// key on the source IP, use X-Forwarded-For if we detect a reverse proxy
		KeyGenerator: func(c *fiber.Ctx) string {
			if len(c.IPs()) == 0 {
				// not behind a reverse proxy, so use real IP
				return c.IP()
			} else {
				// behind a reverse proxy, so use the first address in X-Forwarded-For
				return c.IPs()[0]
			}
		},
		// no fancy page here, just a 429
		LimitReached: func(c *fiber.Ctx) error {
			return fiber.ErrTooManyRequests
		},
		SkipFailedRequests:     false,
		SkipSuccessfulRequests: false,
		LimiterMiddleware:      limiter.FixedWindow{},
		Next: func(c *fiber.Ctx) bool {
			path := c.Path()
			if path == "/api/monitor" {
				return true
			}
			return false
		},
	}))

	// set up handlers
	app.Get("/hello", handleHello)

	// serve the application
	app.Listen(":3000")
}

func isAllowedMetrics(c *fiber.Ctx) error {
	switch ip := c.IP(); ip {
	case "127.0.0.1":
		return c.Next()
	case "localhost":
		return c.Next()
	default:
		return fiber.ErrUnauthorized
	}
}
