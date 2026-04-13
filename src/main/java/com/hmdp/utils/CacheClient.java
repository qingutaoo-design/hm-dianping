package com.hmdp.utils;

import cn.hutool.core.util.BooleanUtil;
import cn.hutool.core.util.StrUtil;
import cn.hutool.json.JSONObject;
import cn.hutool.json.JSONUtil;
import com.hmdp.entity.RedisData;
import com.hmdp.entity.Shop;
import lombok.extern.slf4j.Slf4j;
import org.springframework.boot.autoconfigure.cache.CacheProperties;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.stereotype.Component;
import org.springframework.stereotype.Service;

import java.time.LocalDateTime;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;
import java.util.function.Function;

@Component
@Slf4j
public class CacheClient {
    private StringRedisTemplate stringRedisTemplate;

    //线程池，线程数量根据业务量调整
    private static final ExecutorService CACHE_REBUILD_EXECUTOR = Executors.newFixedThreadPool(10);

    // 构造器注入
    public CacheClient(StringRedisTemplate stringRedisTemplate){
        this.stringRedisTemplate = stringRedisTemplate;
    }

        //封装一个工具类，专门用来操作redis缓存
        //将任意Java对象序列化为json字符串并保存到redis中
    public void set(String key, Object value, Long time, TimeUnit unit){
        String jsonStr = JSONUtil.toJsonStr(value);
        stringRedisTemplate.opsForValue().set(key,jsonStr,time,unit);
    }

    //设置逻辑过期
    public void setWithLogicExpire(String key, Object value, Long time, TimeUnit unit){
        RedisData redisData = new RedisData();
        redisData.setData(value);
        redisData.setExpireTime(LocalDateTime.now().plusSeconds(unit.toSeconds(time)));
        stringRedisTemplate.opsForValue().set(key,JSONUtil.toJsonStr(redisData));
    }

    //通过存null解决缓存穿透问题
    //Function参数<参数类型，返回值类型>
    //<R,ID>返回值类型，id类型
    public <R,ID> R queryWithPassThrough(String keyPrefix ,
                                         ID id , Class<R> type,
                                         Function<ID,R> dbFallback,
                                         Long time,TimeUnit unit){
        String key = keyPrefix + id;

        String resultJson = stringRedisTemplate.opsForValue().get(key);

        if(StrUtil.isNotBlank(resultJson)){
            return JSONUtil.toBean(resultJson,type);
        }

        if (resultJson != null){
            //返回一个错误信息
            return null;
        }
        R r = dbFallback.apply(id);
        if(r == null){
            stringRedisTemplate.opsForValue().set(key,"",RedisConstants.CACHE_NULL_TTL,TimeUnit.MINUTES);
            return null;
        }

        // 6.存在，写入redis
        this.set(key, r, time, unit);
        return r;
    }

    public <R,ID> R queryWithLogicExpire(String keyPreFix ,
                                         ID id, Class<R> type,
                                         Function<ID,R> dbFallback,
                                         Long time,TimeUnit unit){

        String key = keyPreFix + id;
        String resultJson = stringRedisTemplate.opsForValue().get(key);

        if(StrUtil.isBlank(resultJson)){
            return null;
        }
        RedisData redisData = JSONUtil.toBean(resultJson, RedisData.class);
        R r = JSONUtil.toBean((JSONObject) redisData.getData(), type);
        LocalDateTime expireTime = redisData.getExpireTime();
        if(expireTime.isAfter(LocalDateTime.now())){
            return r;
        }
        //todo 可以优化
        String lockKey = RedisConstants.LOCK_SHOP_KEY + id;
        boolean flag = tryLock(lockKey);

        if(flag){
            //二次检查缓存是否过期
            String doubleCheck = stringRedisTemplate.opsForValue().get(key);
            RedisData redisData1 = JSONUtil.toBean(doubleCheck, RedisData.class);
            LocalDateTime expireTime1 = redisData1.getExpireTime();
            if (expireTime1.isAfter(LocalDateTime.now())){
                //此时还先需要解锁，防止后续的访问依然返回旧值
                unlock(lockKey);
                return JSONUtil.toBean((JSONObject) redisData1.getData(),type);
            }
            CACHE_REBUILD_EXECUTOR.submit(
                    () ->{
                        try {
                            R r1 = dbFallback.apply(id);
                            this.setWithLogicExpire(key,r1,time,unit);
                            //模拟复杂缓存重建过程
                            Thread.sleep(200);
                        } catch (Exception e) {
                            throw new RuntimeException(e);
                        } finally {
                            unlock(lockKey);
                        }
                    }
            );
        }
        //直接返回过期商铺信息
        return r;
    }

    public <R, ID> R queryWithMutex(
            String keyPrefix, ID id, Class<R> type, Function<ID, R> dbFallback, Long time, TimeUnit unit) {
        String key = keyPrefix + id;
        // 1.从redis查询商铺缓存
        String shopJson = stringRedisTemplate.opsForValue().get(key);
        // 2.判断是否存在
        if (StrUtil.isNotBlank(shopJson)) {
            // 3.存在，直接返回
            return JSONUtil.toBean(shopJson, type);
        }
        // 判断命中的是否是空值
        if (shopJson != null) {
            // 返回一个错误信息
            return null;
        }

        // 4.实现缓存重建
        // 4.1.获取互斥锁
        String lockKey = RedisConstants.LOCK_SHOP_KEY + id;
        R r = null;
        try {
            boolean isLock = tryLock(lockKey);
            // 4.2.判断是否获取成功
            if (!isLock) {
                // 4.3.获取锁失败，休眠并重试
                Thread.sleep(50);
                return queryWithMutex(keyPrefix, id, type, dbFallback, time, unit);
            }
            // 4.4.获取锁成功，根据id查询数据库
            r = dbFallback.apply(id);
            // 5.不存在，返回错误
            if (r == null) {
                // 将空值写入redis
                stringRedisTemplate.opsForValue().set(key, "", RedisConstants.CACHE_NULL_TTL, TimeUnit.MINUTES);
                // 返回错误信息
                return null;
            }
            // 6.存在，写入redis
            this.set(key, r, time, unit);
        } catch (InterruptedException e) {
            throw new RuntimeException(e);
        }finally {
            // 7.释放锁
            unlock(lockKey);
        }
        // 8.返回
        return r;
    }

    private boolean tryLock(String lockKey){
        Boolean b = stringRedisTemplate.opsForValue().setIfAbsent(lockKey, "1", 10, TimeUnit.SECONDS);
        return BooleanUtil.isTrue(b);
    }

    private void unlock(String lockKey){
        stringRedisTemplate.delete(lockKey);
    }

}
