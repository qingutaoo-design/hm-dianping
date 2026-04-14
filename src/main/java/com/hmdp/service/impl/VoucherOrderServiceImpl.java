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
import com.hmdp.utils.UserHolder;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

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
        if(seckillVoucher.getStock() <= 0){
            return Result.fail("优惠券已抢光！");
        }
        //活动开始，还没结束，且有库存
        //下订单
        long orderId = redisIdWorker.nextId("order");

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
        //方法三,标准答案
        seckillVoucherService.update().setSql("stock = stock - 1")
                .eq("voucher_id", voucherId)
                .gt("stock", 0).update();

        //创建订单
        VoucherOrder voucherOrder = new VoucherOrder();
        voucherOrder.setId(orderId);
        voucherOrder.setUserId(UserHolder.getUser().getId());
        voucherOrder.setVoucherId(voucherId);
        save(voucherOrder);

        return Result.ok(orderId);
    }
}
