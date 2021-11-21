package headscale

import (
	"io/ioutil"
	"os"
	"testing"

	"gopkg.in/check.v1"
	"inet.af/netaddr"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&Suite{})

type Suite struct{}

var tmpDir string
var h Headscale

func (s *Suite) SetUpTest(c *check.C) {
	s.ResetDB(c)
}

func (s *Suite) TearDownTest(c *check.C) {
	os.RemoveAll(tmpDir)
}

func (s *Suite) ResetDB(c *check.C) {
	if len(tmpDir) != 0 {
		os.RemoveAll(tmpDir)
	}
	var err error
	tmpDir, err = ioutil.TempDir("", "autoygg-client-test")
	if err != nil {
		c.Fatal(err)
	}
	cfg := Config{
		IPPrefixes: []netaddr.IPPrefix{
			netaddr.MustParseIPPrefix("10.27.0.0/23"),
		},
	}

	h = Headscale{
		cfg:      cfg,
		dbType:   "sqlite3",
		dbString: tmpDir + "/headscale_test.db",
	}
	err = h.initDB()
	if err != nil {
		c.Fatal(err)
	}
	db, err := h.openDB()
	if err != nil {
		c.Fatal(err)
	}
	h.db = db
}
