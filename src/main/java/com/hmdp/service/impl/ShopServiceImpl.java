package com.hmdp.service.impl;

import cn.hutool.core.util.StrUtil;
import cn.hutool.json.JSONUtil;
import com.hmdp.dto.Result;
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

        String cacheKey = RedisConstants.CACHE_SHOP_KEY + id;

        //先根据id查询缓存
        String shopInfo = stringRedisTemplate.opsForValue().get(cacheKey);

        if(StrUtil.isNotBlank(shopInfo)) {
            //如果缓存中有数据，直接返回
            log.info("从缓存中查询商铺信息，商铺id：{}", id);
            Shop shop = JSONUtil.toBean(shopInfo, Shop.class);
            return Result.ok(shop);
        }
        //如果缓存中没有数据，根据id查询数据库
        Shop shop = shopMapper.selectById(id);
        if(shop == null) {
            //如果数据库中没有数据，返回错误信息
            return Result.fail("商铺不存在");
        }
        //如果数据库中有数据，将数据写入缓存，并设置过期时间
        stringRedisTemplate.opsForValue().set(cacheKey, JSONUtil.toJsonStr(shop),RedisConstants.CACHE_SHOP_TTL, TimeUnit.MINUTES);
        return Result.ok(shop);
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
