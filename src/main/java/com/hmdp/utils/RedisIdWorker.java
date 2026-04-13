package com.hmdp.utils;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.stereotype.Component;

import javax.annotation.Resource;
import java.time.LocalDateTime;
import java.time.ZoneOffset;
import java.time.format.DateTimeFormatter;

@Component
public class RedisIdWorker {

    @Resource
    StringRedisTemplate stringRedisTemplate;
    /**
     * 开始时间戳
     */
    private static final long BEGIN_TIMESTAMP = 1767225600;
    /**
     * 序列号的位数
     */
    private static final int COUNT_BITS = 32;

    /**
     * 获取全局唯一id
     * @param keyPrefix
     * @return
     */
    public long nextId(String keyPrefix){
        //计算时间戳的差值，减小数值大小
        long currentEpoch = LocalDateTime.now().toEpochSecond(ZoneOffset.UTC);
        long timeStamp = currentEpoch - BEGIN_TIMESTAMP;

        //获取序列号（redis自增）
        // 先获取key
        String date = LocalDateTime.now().format(DateTimeFormatter.ofPattern("yyyy:MM:dd"));
        Long increment = stringRedisTemplate.opsForValue().increment("icr:" + keyPrefix + ":" + date);


        //拼接id
        return timeStamp << COUNT_BITS | increment;
    }

    /**
     * 获取2026-1-1 0：0：0 的秒级时间戳
     * @param args
     */
//    public static void main(String[] args) {
//        LocalDateTime localDateTime = LocalDateTime.of(2026, 1, 1, 0, 0, 0);
//
//        long epochSecond = localDateTime.toEpochSecond(ZoneOffset.UTC);
//        System.out.println(epochSecond);//1767225600
//
//    }

}
