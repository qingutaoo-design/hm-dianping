package com.hmdp.service.impl;

import com.baomidou.mybatisplus.core.conditions.Wrapper;
import com.baomidou.mybatisplus.core.conditions.query.QueryWrapper;
import com.baomidou.mybatisplus.core.conditions.update.Update;
import com.baomidou.mybatisplus.core.conditions.update.UpdateWrapper;
import com.hmdp.dto.Result;
import com.hmdp.entity.SeckillVoucher;
import com.hmdp.entity.Voucher;
import com.hmdp.entity.VoucherOrder;
import com.hmdp.mapper.SeckillVoucherMapper;
import com.hmdp.mapper.VoucherMapper;
import com.hmdp.mapper.VoucherOrderMapper;
import com.hmdp.service.ISeckillVoucherService;
import com.hmdp.service.IVoucherOrderService;
import com.baomidou.mybatisplus.extension.service.impl.ServiceImpl;
import com.hmdp.utils.RedisIdWorker;
import com.hmdp.utils.SimpleRedisLock;
import com.hmdp.utils.UserHolder;
import org.springframework.aop.framework.AopContext;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.lang.Nullable;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.LocalDateTime;

/**
 * <p>
 *  服务实现类
 * </p>
 *
 * @author 虎哥
 * @since 2021-12-22
 */
@Service
public class VoucherOrderServiceImpl extends ServiceImpl<VoucherOrderMapper, VoucherOrder> implements IVoucherOrderService {

    @Autowired
    VoucherOrderMapper voucherOrderMapper;

    @Autowired
    VoucherMapper  voucherMapper;

    @Autowired
    RedisIdWorker redisIdWorker;

    @Autowired
    SeckillVoucherMapper seckillVoucherMapper;

    @Autowired
    ISeckillVoucherService seckillVoucherService;

    @Autowired
    StringRedisTemplate stringRedisTemplate;

    /**
     * 下单秒杀优惠券
     * @param voucherId
     * @return
     */
    @Override
    public Result orderSeckillVoucher(Long voucherId) {
        SeckillVoucher seckillVoucher = seckillVoucherMapper.selectById(voucherId);

        //判断活动是否还没开始
        if(seckillVoucher.getBeginTime().isAfter(LocalDateTime.now())){
            return Result.fail("活动还没开始!");
        }
        //判断活动是否结束
        if(seckillVoucher.getEndTime().isBefore(LocalDateTime.now())){
            return Result.fail("活动已经结束!");
        }
        //判断是否还有库存
        if(seckillVoucher.getStock() < 1){
            return Result.fail("优惠券已抢光！");
        }
        //活动开始，还没结束，且有库存



        //扣库存
//        //方法一：
//        UpdateWrapper<SeckillVoucher> updateWrapper = new UpdateWrapper<>();
//        SeckillVoucher seckillVoucher1 = new SeckillVoucher();
//        seckillVoucher1.setStock(seckillVoucher.getStock() - 1);
//        updateWrapper.eq("voucher_id", voucherId)
//                .gt("stock", 0);
//        seckillVoucherMapper.update(seckillVoucher1, updateWrapper);

//        //方法二
//        seckillVoucherMapper.update(null, new UpdateWrapper<SeckillVoucher>()
//                .setSql("stock = stock - 1")
//                .eq("voucher_id", voucherId)
//                .gt("stock", 0));
//

        //todo 这里可以做二次验证，判断该用户是否已经抢过了


        SimpleRedisLock simpleRedisLock = new SimpleRedisLock("order:" + UserHolder.getUser().getId(), stringRedisTemplate);

        if(!simpleRedisLock.tryLock(120)) {
            //获取锁失败，返回错误信息
            return Result.fail("你已经抢过了！");
        }
        try {
            //这里需要使用代理技术，因为事务注解只能在被代理对象的方法上生效，而不能在当前对象的方法上生效，所以需要获取当前对象的代理对象来调用方法
            IVoucherOrderService proxy = (IVoucherOrderService) AopContext.currentProxy();
            return proxy.createVoucherOrder(voucherId);
        } finally {
            simpleRedisLock.unlock();
        }

    }


    /**
     * 实现一人一单
     * @param voucherId
     * @return
     */
    @Transactional
    public   Result createVoucherOrder(Long voucherId) {

        //添加一人一单功能
        int count = query().eq("user_id", UserHolder.getUser().getId())
                .eq("voucher_id", voucherId).count();
        if(count > 0){
            return Result.fail("你已经抢过了！");
        }
        //下订单
        long orderId = redisIdWorker.nextId("order");
        //方法三,标准答案
        //修改优惠券库存
        boolean success = seckillVoucherService.update().setSql("stock = stock - 1")
                .eq("voucher_id", voucherId)
                //乐观锁解决超卖问题，更新时判断库存是否大于0
                .gt("stock", 0).update();

        if (!success) {
            //扣减库存
            return Result.fail("库存不足！");
        }

        //创建订单
        VoucherOrder voucherOrder = new VoucherOrder();
        voucherOrder.setId(orderId);
        voucherOrder.setUserId(UserHolder.getUser().getId());
        voucherOrder.setVoucherId(voucherId);
        save(voucherOrder);
        return Result.ok(orderId);
    }
}
