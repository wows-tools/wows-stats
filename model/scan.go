package model

import (
	"time"
)

type Scan struct {
	StartDate    time.Time
	EndDate      time.Time
	ApiCallCount int
}
