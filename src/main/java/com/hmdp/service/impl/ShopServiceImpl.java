package com.hmdp.service.impl;

import cn.hutool.core.util.BooleanUtil;
import cn.hutool.core.util.StrUtil;
import cn.hutool.json.JSONObject;
import cn.hutool.json.JSONUtil;
import com.hmdp.dto.Result;
import com.hmdp.entity.RedisData;
import com.hmdp.entity.Shop;
import com.hmdp.mapper.ShopMapper;
import com.hmdp.service.IShopService;
import com.baomidou.mybatisplus.extension.service.impl.ServiceImpl;
import com.hmdp.utils.RedisConstants;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import javax.annotation.Resource;
import java.time.LocalDateTime;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;

/**
 * <p>
 *  服务实现类
 * </p>
 *
 * @author 虎哥
 * @since 2021-12-22
 */
@Slf4j
@Service
public class ShopServiceImpl extends ServiceImpl<ShopMapper, Shop> implements IShopService {

    @Autowired
    private ShopMapper shopMapper;
    @Resource
    private StringRedisTemplate stringRedisTemplate;

    @Override
    public Result getById(Long id) {
////        查询商品信息
////        解决缓存穿透（直接返回空值）
//       Shop shop = dealWithPassThrough(id);

//        //解决缓存击穿（互斥锁）
//        Shop shop = queryWithMutex(id);

        //解决缓存击穿（逻辑过期）

        Shop shop = queryWithLogicExpire(id);

        if(shop == null){
            return Result.fail("商铺信息不存在");
        }
        return Result.ok(shop);
    }

    private static final ExecutorService CACHE_REBUILD_EXECUTOR = Executors.newFixedThreadPool(10);

    public Shop queryWithLogicExpire(Long id){
        String cacheKey = RedisConstants.CACHE_SHOP_KEY + id;

        //先根据id查询缓存
        String cacheShop = stringRedisTemplate.opsForValue().get(cacheKey);

        //若shopInfo为空字符串，isNotBlank方法返回false
        if (StrUtil.isBlank(cacheShop)) {
            //如果缓存没有数据，直接返回空
            return null;
        }

        //如果命中了缓存，判断时间是否超时
        RedisData redisData = JSONUtil.toBean(cacheShop, RedisData.class);
        //旧数据,两次转换得来
        Shop shop = JSONUtil.toBean((JSONObject) redisData.getData(), Shop.class);
        //如果没超时，直接返还数据
        if(redisData.getExpireTime().isAfter(LocalDateTime.now())){
            return shop;
        }
        //如果超时，重建缓存，二次检验拿锁线程的数据是否可以直接返回
        String lockKey = RedisConstants.LOCK_SHOP_KEY + id;
        if(tryLock(lockKey)){
            if(redisData.getExpireTime().isAfter(LocalDateTime.now())){
                return shop;
            }
            CACHE_REBUILD_EXECUTOR.submit( () ->{
                        try{
                            //重建缓存
                            this.rebuildCacheWithLogicExpire(id,20L);
                        }catch (Exception e){
                            throw new RuntimeException(e);
                        }finally {
                            delLock(lockKey);
                        }
                    }

            );
        }

        return shop;
    }



    public void rebuildCacheWithLogicExpire(Long id , Long time){

        String key = RedisConstants.CACHE_SHOP_KEY + id;
        LocalDateTime expireTime = LocalDateTime.now().plusSeconds(time);

        RedisData redisData = new RedisData();
        redisData.setData(shopMapper.selectById(id));
        redisData.setExpireTime(expireTime);

        String jsonStr = JSONUtil.toJsonStr(redisData);
        log.info("重建缓存");
        stringRedisTemplate.opsForValue().set(key,jsonStr);


    }


    public Shop queryWithMutex(Long id) {
        String cacheShopKey = RedisConstants.CACHE_SHOP_KEY + id;

        //先根据id查询缓存
        String cacheShop1 = stringRedisTemplate.opsForValue().get(cacheShopKey);
        //若shopInfo为空字符串，isNotBlank方法返回false
        if (StrUtil.isNotBlank(cacheShop1)) {
            //如果缓存中有数据，直接返回
            log.info("从缓存中查询商铺信息，商铺id：{}", id);
            Shop shop = JSONUtil.toBean(cacheShop1, Shop.class);
            return shop;
        }

        if (cacheShop1 != null) {
            //如果缓存中有空字符串，说明数据库中没有数据，直接返回错误信息
            return null;
        }
        //如果缓存中没有数据,先获取锁，防止直接打在数据库上
        String lockKey = RedisConstants.LOCK_SHOP_KEY + id;
        Shop shop = null;

        try {
            //没获取到锁执行递归
            //进行本线程是否获取到锁进行判断,若没有获取到锁，则休眠之后再次去尝试获取
            boolean flag = tryLock(lockKey);
            if(!flag){
                Thread.sleep(50);
                return queryWithMutex(id);
            }
            log.info("拿到锁");
            //拿到锁的应该二次检查缓存中是否有了数据
            String cacheShop2 = stringRedisTemplate.opsForValue().get(cacheShopKey);
            if (StrUtil.isNotBlank(cacheShop2)) {
                //如果缓存中有数据，直接返回
                log.info("从缓存中查询商铺信息:{}",id);
                Shop checkShop = JSONUtil.toBean(cacheShop2, Shop.class);
                return checkShop;
            }

            //没有从缓存中拿到商铺信息，则从数据库中拿
            shop = shopMapper.selectById(id);
            log.info("操作数据库：{}",shop);
            //模拟复杂数据库查询
            Thread.sleep(100);

            if (shop == null) {
                //如果数据库中没有数据，返回错误信息,并将空字符串写入缓存，并设置过期时间
                stringRedisTemplate.opsForValue().set(cacheShopKey, "", RedisConstants.CACHE_NULL_TTL, TimeUnit.MINUTES);
                return null;
            }
            //如果数据库中有数据，将数据写入缓存，并设置过期时间
            stringRedisTemplate.opsForValue().set(cacheShopKey, JSONUtil.toJsonStr(shop), RedisConstants.CACHE_SHOP_TTL, TimeUnit.MINUTES);
        } catch (InterruptedException e) {
            throw new RuntimeException(e);
        }finally{
            delLock(lockKey);
        }

        return shop;
    }




    private boolean tryLock(String key){
        log.info("获取锁:{}",key);
        Boolean flag = stringRedisTemplate.opsForValue().setIfAbsent( key, "1", 10, TimeUnit.SECONDS);
        //防止flag为空，在拆箱时会报错
        return BooleanUtil.isTrue(flag);
    }
    private void delLock(String key){
        log.info("解锁:{}",key);
        stringRedisTemplate.delete( key);
    }

    //解决缓存穿透（直接返回null）
    public Shop dealWithPassThrough(Long id){
        String cacheKey = RedisConstants.CACHE_SHOP_KEY + id;

        //先根据id查询缓存
        String cacheShop = stringRedisTemplate.opsForValue().get(cacheKey);

        //若shopInfo为空字符串，isNotBlank方法返回false
        if (StrUtil.isNotBlank(cacheShop)) {
            //如果缓存中有数据，直接返回
            log.info("从缓存中查询商铺信息，商铺id：{}", id);
            Shop shop = JSONUtil.toBean(cacheShop, Shop.class);
            return shop;
        }
        if (cacheShop != null) {
            //如果缓存中有空字符串，说明数据库中没有数据，直接返回错误信息
            return null;
        }

        //缓存中的商铺信息为空，从数据库中取数据
        Shop shop = shopMapper.selectById(id);
            if (shop == null) {
                //如果数据库中没有数据，返回错误信息,并将空字符串写入缓存，并设置过期时间
                stringRedisTemplate.opsForValue().set(cacheKey, "", RedisConstants.CACHE_NULL_TTL, TimeUnit.MINUTES);
                return null;
            }
            //如果数据库中有数据，将数据写入缓存，并设置过期时间
            stringRedisTemplate.opsForValue().set(cacheKey, JSONUtil.toJsonStr(shop), RedisConstants.CACHE_SHOP_TTL, TimeUnit.MINUTES);
        return shop;
    }

    @Override
    //为了保证数据库和缓存同时操作，应该添加事务
    @Transactional
    public Result update(Shop shop) {
        Long id = shop.getId();//获取商铺id

        if(id == null) {
            //如果商铺id不存在，直接返回
            return Result.fail("商铺id不能为空");
        }

        //先更新数据库，再删除缓存。这样安全性更高
        updateById(shop);
        //删除缓存
        stringRedisTemplate.delete(RedisConstants.CACHE_SHOP_KEY + shop.getId());
        return Result.ok();
    }


}
