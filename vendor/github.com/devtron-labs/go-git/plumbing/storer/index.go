package storer

import "github.com/devtron-labs/go-git/plumbing/format/index"

// IndexStorer generic storage of index.Index
type IndexStorer interface {
	SetIndex(*index.Index) error
	Index() (*index.Index, error)
}
