package cache

import (
	"github.com/coocood/freecache"
)

var GlobalCache *freecache.Cache

func InitCache(size int) {
	GlobalCache = freecache.NewCache(size)
}
