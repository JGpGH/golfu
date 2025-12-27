# GOLFU
In-memory cache system
- plug your cold storage on it to eventually persist all setted data
- prevent some resources to be trashed with the SetOptions.CanBeTrashed
- auto eviction by read count (LFU); only evicts persisted data
- retrieves all cache-miss from the cold storage
- you will see everything you need to implement or use and a couple helpers func & struct in the [storage module](storage/storage.go)
## ctx
Don't use in production lol <br>
Feel free to submit a PR or put a comment or whatever if you find a bug or wanna improve it somehow

     /\
    ( /   @ @    ()
     \  __| |__  /
      -/   "   \-
     /-|       |-\
    / /-\     /-\ \
     / /-`---'-\ \     ascii crab
      /         \
