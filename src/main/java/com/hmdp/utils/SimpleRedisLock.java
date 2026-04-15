package com.hmdp.utils;

import cn.hutool.core.lang.UUID;
import cn.hutool.core.util.BooleanUtil;
import com.hmdp.dto.Result;
import org.springframework.data.redis.core.StringRedisTemplate;

import java.util.concurrent.TimeUnit;

public class SimpleRedisLock implements ILock{

    private String name;

    private StringRedisTemplate stringRedisTemplate;

    private static final String KEY_PREFIX = "lock:";

    public SimpleRedisLock (String name, StringRedisTemplate stringRedisTemplate) {
        this.name = name;
        this.stringRedisTemplate = stringRedisTemplate;
    }

     private static final String ID_PREFIX = UUID.randomUUID().toString(true) + "-";

    @Override
    public boolean tryLock(long timeoutSec) {
        //通过给value值添加上uuid区分jvm，threadid区分线程，解决误删问题

        long threadId = Thread.currentThread().getId();
        String value = ID_PREFIX + threadId;

        //尝试获取锁
        Boolean success = stringRedisTemplate.opsForValue().setIfAbsent(KEY_PREFIX + name, value , timeoutSec, TimeUnit.SECONDS);

        return BooleanUtil.isTrue(success);
    }

    @Override
    public void unlock() {
        //解决误删问题
        String currentValue = stringRedisTemplate.opsForValue().get(KEY_PREFIX + name);
        String value = ID_PREFIX + Thread.currentThread().getId();
        if(value.equals(currentValue)){
            stringRedisTemplate.delete(KEY_PREFIX + name);
        }
    }
}
