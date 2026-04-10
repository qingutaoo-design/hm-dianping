package com.hmdp.service.impl;

import cn.hutool.core.util.StrUtil;
import cn.hutool.json.JSONUtil;
import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.baomidou.mybatisplus.core.conditions.query.QueryWrapper;
import com.hmdp.entity.ShopType;
import com.hmdp.mapper.ShopTypeMapper;
import com.hmdp.service.IShopTypeService;
import com.baomidou.mybatisplus.extension.service.impl.ServiceImpl;
import com.hmdp.utils.RedisConstants;
import org.springframework.data.redis.core.RedisCallback;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.stereotype.Service;

import javax.annotation.Resource;
import java.util.List;

/**
 * <p>
 *  服务实现类
 * </p>
 *
 * @author 虎哥
 * @since 2021-12-22
 */
@Service
public class ShopTypeServiceImpl extends ServiceImpl<ShopTypeMapper, ShopType> implements IShopTypeService {

    @Resource
    private StringRedisTemplate stringRedisTemplate;
    @Resource
    private ShopTypeMapper shopTypeMapper;

    /**
     * 查询商铺类型列表
     * @return
     */
    @Override
    public List<ShopType> list() {
        String cacheKey = RedisConstants.CACHE_SHOP_TYPE_KEY + "list";
        String shopType = stringRedisTemplate.opsForValue().get(cacheKey);

        if(StrUtil.isNotBlank(shopType)){
            //如果缓存中有数据，直接返回
            List<ShopType> shopTypes = JSONUtil.toList(shopType, ShopType.class);
            return shopTypes;
        }

        //如果缓存中没有数据，就从数据库中查询
        List<ShopType> shopTypes = shopTypeMapper.selectList(new QueryWrapper<ShopType>().orderByAsc("sort"));

        if (shopTypes == null || shopTypes.isEmpty()) {
            //如果数据库中没有数据，返回null
            return null;
        }
        //如果数据库中有数据，将数据写入缓存，并设置过期时间
        stringRedisTemplate.opsForValue().set(cacheKey, JSONUtil.toJsonStr(shopTypes));
        return shopTypes;
    }
}
