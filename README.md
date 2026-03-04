# GOLFU
In-memory cache system
- plug your cold storage on it to eventually persist all setted data
- prevent some resources to be removed from the cache by implementing [Evictable.CanBeEvicted](element/element.go) on the element
- auto eviction by read count (LFU)
- retrieves all cache-miss from the cold storage provided and sets new elements both in cold storage and cache
## Module tour
### [storage module](storage) 
definition of the cached storage and cold storage
### [element module](element)
any Indexable is storable, any Evictable can be kept explicitely in memory
### [coldstorages module](coldstorages)
file based implementation of a cold storage using gob for encoding

## notes
Don't use in production. <br>
Feel free to submit a PR or put a comment or whatever if you find a bug or wanna improve it somehow

## release
increase version in [README.md](README.md):
Given a version number MAJOR.MINOR.PATCH, increment the:

- MAJOR version when you make incompatible API changes
- MINOR version when you add functionality in a backward compatible manner
- PATCH version when you make backward compatible bug fixes <br>

`go mod tidy`<br>
`go test ./...`<br>
`git commit -m "what I just did - vX.Y.Z"`<br>
`git tag vX.Y.Z`<br>
`git push origin vX.Y.Z`<br>
`GOPROXY=proxy.golang.org go list -m github.com/JGpGH/golfu@vX.Y.Z`

## current version
v0.8.4


     /\
    ( /   @ @    ()
     \  __| |__  /
      -/   "   \-
     /-|       |-\
    / /-\     /-\ \
     / /-`---'-\ \     crab of luck
      /         \
