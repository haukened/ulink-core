/*
 * Created Date: Tue Sep 26 2023
 * Author: David Haukeness david@hauken.us
 * Copyright (c) 2023 David Haukeness
 * Distributed under the terms of the GNU GENERAL PUBLIC LICENSE Version 3.0
 */

package main

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

func handleHello(c *fiber.Ctx) error {
	ips := c.IPs()
	if len(ips) == 0 {
		return c.SendString(c.IP())
	}
	return c.SendString(strings.Join(c.IPs(), " "))
}
