package checks

import (
	"math/rand"
	"time"
)

// RandomFailHealthcheck defines a healthcheck that can be executed
type RandomFailHealthcheck struct {
	Name        string
	Description string
	FailRate    int
}

// Execute runs the healthcheck
func (c RandomFailHealthcheck) Execute() Result {
	input := struct {
		FailRate int `json:"failRate"`
	}{
		c.FailRate,
	}

	number := 0
	reason := ""

	switch c.FailRate {
	case 0:
		number = 0
		reason = "Never failing"
		break
	case 1:
		number = 1
		reason = "Always failing"
		break
	default:
		rand.Seed(time.Now().UnixNano())
		max := c.FailRate
		number = rand.Intn(max) + 1
		reason = "Randomly failing"
	}

	output := struct {
		Number int `json:"number"`
	}{
		number,
	}

	if number > 0 && number == c.FailRate {
		return FailWithIO(reason, input, output)
	}

	return PassWithIO(input, output)
}

// Describe returns the description of the healthcheck
func (c RandomFailHealthcheck) Describe() Description {
	return Description{
		Name:        c.Name,
		Description: c.Description,
	}
}
