package main

import "time"

type Paste struct {
	Content   string
	ExpiresAt time.Time
}
