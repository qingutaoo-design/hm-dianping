package com.hmdp.utils;

import cn.hutool.core.lang.UUID;
import cn.hutool.core.util.BooleanUtil;
import com.hmdp.dto.Result;
import org.springframework.core.io.ClassPathResource;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.core.script.DefaultRedisScript;

import java.util.Collection;
import java.util.Collections;
import java.util.concurrent.TimeUnit;

public class SimpleRedisLock implements ILock{

    private String name;

    private StringRedisTemplate stringRedisTemplate;

    private static final String KEY_PREFIX = "lock:";

    //lua脚本，保证原子性
    //为了防止多次读取脚本导致性能问题，我们将脚本编译成对象，放在成员变量当中
    //脚本内容：如果锁的值和当前线程的标识相同，则删除锁
    private static final DefaultRedisScript<Long> UNLOCK_SCRIPT ;
    static {
        UNLOCK_SCRIPT = new DefaultRedisScript<>();
        UNLOCK_SCRIPT.setLocation(new ClassPathResource("unlock.lua"));
        UNLOCK_SCRIPT.setResultType(Long.class);
    }

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
//        //解决误删问题
//        String currentValue = stringRedisTemplate.opsForValue().get(KEY_PREFIX + name);
//        String value = ID_PREFIX + Thread.currentThread().getId();
//        if(value.equals(currentValue)){
//            stringRedisTemplate.delete(KEY_PREFIX + name);
//        }

        long threadId = Thread.currentThread().getId();
        String value = ID_PREFIX + threadId;

        //使用lua脚本删除锁，保证原子性
        stringRedisTemplate.execute(UNLOCK_SCRIPT, Collections.singletonList(KEY_PREFIX + name),value);

    }
}
