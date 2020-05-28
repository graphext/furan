package datalayer_test

import (
	"testing"

	"github.com/dollarshaveclub/furan/lib/datalayer"
	"github.com/dollarshaveclub/furan/lib/datalayer/testsuite"
)

func TestFakeDBSuite(t *testing.T) {
	testsuite.RunTests(t, func() datalayer.DataLayer {
		return &datalayer.FakeDataLayer{}
	})
}
