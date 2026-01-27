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

     /\
    ( /   @ @    ()
     \  __| |__  /
      -/   "   \-
     /-|       |-\
    / /-\     /-\ \
     / /-`---'-\ \     crab of luck
      /         \
