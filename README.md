## Very fast customized Caching Server in Go - CacheGo

- Capable of +200K read/write/delete cycles per second
- With custom auto-expiration for records set globally or per record
- UDP-based communication

## Format

- Prefix||Suffix||TimeOut||< MaxRecordsPerPrefix >
- Or, Prefix||Suffix||< MaxRecordsPerPrefix >
