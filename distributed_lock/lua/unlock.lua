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

