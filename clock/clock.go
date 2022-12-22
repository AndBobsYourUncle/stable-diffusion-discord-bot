package clock

import "time"

//go:generate mockgen -destination=mock/mock.go -package=mock_clock -source=clock.go

type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}

func NewClock() Clock {
	return &realClock{}
}
