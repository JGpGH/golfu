# GOLFU
In-memory cache system
- plug your cold storage on it to eventually persist all setted data
- auto eviction by read count (LFU); only evicts persisted data
- retrieves all cache-miss from the cold storage
- "eventual" persistency (async if you will) allows for shorter set operation
- you will see everything you need to implement or use and a couple helpers func & struct in /storage
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
