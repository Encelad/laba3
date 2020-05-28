package main

import (
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type MySuite struct{}

var _ = Suite(&MySuite{})

func (s *MySuite) TestBalancer(c *C) {
	// TODO: Реалізуйте юніт-тест для балансувальникка.
	availableAnyServer := false
	testTraffic := [3]int{999, 15, 440}
	testHealth := [3]bool{true, true, true}
	res1, _, availableAnyServer := testBalancer(testTraffic, testHealth)
	c.Assert(1, Equals, res1)
	c.Assert(true, Equals, availableAnyServer)
	testHealth[1] = false
	res2, _, availableAnyServer := testBalancer(testTraffic, testHealth)
	c.Assert(2, Equals, res2)
	c.Assert(true, Equals, availableAnyServer)
	testHealth[0] = false
	testHealth[2] = false
	res3, _, availableAnyServer := testBalancer(testTraffic, testHealth)
	c.Assert(0, Equals, res3)
	c.Assert(false, Equals, availableAnyServer)
}
