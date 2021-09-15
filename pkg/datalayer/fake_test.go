package datalayer_test

import (
	"testing"

	"github.com/dollarshaveclub/furan/v2/pkg/datalayer"
	"github.com/dollarshaveclub/furan/v2/pkg/datalayer/testsuite"
)

func TestFakeDBSuite(t *testing.T) {
	testsuite.RunTests(t, func(t *testing.T) datalayer.DataLayer {
		return &datalayer.FakeDataLayer{}
	})
}
