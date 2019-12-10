package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestShouldCheckUpdate(t *testing.T) {
	factory := FromFile(nil, nil, time.Millisecond*500)
	cache, _ := factory("")
	file := cache.(*file)

	require.True(t, file.shouldCheckUpdate())

	time.Sleep(time.Millisecond * 200)

	require.False(t, file.shouldCheckUpdate())

	time.Sleep(time.Millisecond * 310)

	require.True(t, file.shouldCheckUpdate())

	time.Sleep(time.Millisecond * 500)

	require.True(t, file.shouldCheckUpdate())
}
