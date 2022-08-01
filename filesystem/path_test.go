package filesystem

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathParents(t *testing.T) {
	assert.Equal(t, []Path{"/foo/bar/baz", "/foo/bar", "/foo", "/"}, Path("/foo/bar/baz/1").Parents())
	assert.Equal(t, []Path{"foo/bar/baz", "foo/bar", "foo", "."}, Path("foo/bar/baz/1").Parents())
}
