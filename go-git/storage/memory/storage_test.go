package memory

import (
	"testing"

	"github.com/devtron-labs/git-sensor/go-git/storage/test"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type StorageSuite struct {
	test.BaseStorageSuite
}

var _ = Suite(&StorageSuite{})

func (s *StorageSuite) SetUpTest(c *C) {
	s.BaseStorageSuite = test.NewBaseStorageSuite(NewStorage())
}
