### Added/Changed
1. Added type definitions for Guild struct(s).
2. Created basic shard, discord gateway connection logic.

### Planned
1. Make shard support whole range of instructions like reconnect, resume session, etc.
2. Add ability to spawn multiple shards + have auto sharding as option (ask API for number of shards).
3. Add support for v2 components.
4. Optimize memory layout of all structs.
5. Hide struct fields that are supposed to not be used by outside code.