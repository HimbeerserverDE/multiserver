package main

import (
	"math"
	"time"
)

var uptime time.Time

// Uptime reports how long the program has been running
func Uptime() float64 {
	return math.Floor(time.Since(uptime).Seconds())
}

func init() {
	uptime = time.Now()
}
