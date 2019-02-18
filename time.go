package main

import (
	"strconv"
	"time"
)

type Seconds time.Duration

func (s Seconds) String() string {
	return strconv.FormatInt(int64(time.Duration(s)/time.Second), 10)
}
