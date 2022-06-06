package distributed_lock

// 加锁lua脚本
const lockLuaScript = `
-- lock
-- KEYS: [lockKey, goRoutineId]
-- ARGV: [reentrantCount, expiredTime]
if redis.call("exists", KEYS[1]) == 0 then
    redis.call("hset", KEYS[1], KEYS[2], ARGV[1])
    redis.call("pexpire", KEYS[1], ARGV[2])
    return 0;
else
    if redis.call("hexists", KEYS[1],KEYS[2]) == 0 then
        return redis.call("pttl", KEYS[1]);
    else
        redis.call("hincrby", KEYS[1], KEYS[2], 1)
        redis.call("pexpire", KEYS[1], ARGV[2])
        return 0;
    end
end
`

// 释放锁脚本
const unlockLuaScript = `
-- unlock
-- KEYS: [lockKey, goRoutineId]
-- ARGV: [reentrantCount, expiredTime]
if redis.call("exists", KEYS[1]) > 0 then
    if redis.call("hexists", KEYS[1], KEYS[2]) > 0 then
        if redis.call("hincrby", KEYS[1], KEYS[2], -1) == 0 then
            redis.call("del", KEYS[1]);
            -- 发布锁释放的消息
            redis.call("publish", ARGV[1], KEYS[2])
        end
        return 1
    end
end
return 0
`
