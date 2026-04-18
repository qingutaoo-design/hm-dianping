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
import org.redisson.api.RLock;
import org.redisson.api.RedissonClient;
import org.springframework.aop.framework.AopContext;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.core.io.ClassPathResource;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.core.script.DefaultRedisScript;
import org.springframework.lang.Nullable;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import javax.annotation.PostConstruct;
import javax.annotation.Resource;
import java.time.LocalDateTime;
import java.util.Collection;
import java.util.Collections;
import java.util.concurrent.ArrayBlockingQueue;
import java.util.concurrent.BlockingQueue;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

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

    @Resource
    private RedissonClient redissonClient;

    private BlockingQueue<VoucherOrder> orderTasks = new ArrayBlockingQueue<>(1024 * 1024);

    private static final ExecutorService SECKILL_ORDER_EXECUTOR = Executors.newSingleThreadExecutor();


    IVoucherOrderService proxy;
    @PostConstruct
    public void init(){
        SECKILL_ORDER_EXECUTOR.submit(new VoucherOrderHandler());
    }

    private class VoucherOrderHandler implements Runnable{

        @Override
        public void run() {
            while(true){
                //从队列中获取订单信息
                try {
                    VoucherOrder voucherOrder = orderTasks.take();
                    //创建订单
                    //异步下单导致无法获取到代理对象，因为代理对象也是基于原线程创建的，所以只能通过成员变量获取代理对象
                    //也是因为异步下单，导致无法获取到UserHolder中的用户信息，因为UserHolder中的用户信息是基于ThreadLocal存储的
                    //可以根据订单信息中的用户id查询用户信息
                    proxy.createVoucherOrder(voucherOrder);
                } catch (Exception e) {
                    log.error("处理订单异常", e);
                }
            }
        }
    }


    private static final DefaultRedisScript<Long> SECKILL_SCRIPT ;
    static {
        SECKILL_SCRIPT = new DefaultRedisScript<>();
        SECKILL_SCRIPT.setLocation(new ClassPathResource("seckill.lua"));
        SECKILL_SCRIPT.setResultType(Long.class);
    }

    /**
     * 下单秒杀优惠券
     * @param voucherId
     * @return
     */
    @Override
    public Result orderSeckillVoucher(Long voucherId) {

        Long result = stringRedisTemplate.execute(SECKILL_SCRIPT,
                Collections.EMPTY_LIST,
                voucherId.toString(), UserHolder.getUser().getId().toString());
        //返回结果为0，说明抢到了
        //返回结果为1，说明库存不足
        //返回结果为2，说明用户已经抢过了
        if(result == 1L){
            return Result.fail("库存不足！");
        } else if(result == 2L){
            return Result.fail("你已经抢过了！");
        }

        //创建订单
        long orderId = redisIdWorker.nextId("order");

        VoucherOrder voucherOrder = new VoucherOrder();
        voucherOrder.setId(orderId);
        voucherOrder.setUserId(UserHolder.getUser().getId());
        voucherOrder.setVoucherId(voucherId);


        //抢到了，将订单信息放入阻塞队列
        orderTasks.add(voucherOrder);
        //赋值代理对象
        proxy = (IVoucherOrderService) AopContext.currentProxy();
        //异步下单

        //抢到了，返回订单id
        return Result.ok(orderId);
    }

    /**
     * 下单秒杀优惠券
     * @param voucherId
     * @return
     */
//    @Override
//    public Result orderSeckillVoucher(Long voucherId) {
//        SeckillVoucher seckillVoucher = seckillVoucherMapper.selectById(voucherId);
//
//        //判断活动是否还没开始
//        if(seckillVoucher.getBeginTime().isAfter(LocalDateTime.now())){
//            return Result.fail("活动还没开始!");
//        }
//        //判断活动是否结束
//        if(seckillVoucher.getEndTime().isBefore(LocalDateTime.now())){
//            return Result.fail("活动已经结束!");
//        }
//        //判断是否还有库存
//        if(seckillVoucher.getStock() < 1){
//            return Result.fail("优惠券已抢光！");
//        }
//        //活动开始，还没结束，且有库存
//


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

////        //添加分布式锁，解决一人多单问题
////        SimpleRedisLock simpleRedisLock = new SimpleRedisLock("order:" + UserHolder.getUser().getId() , stringRedisTemplate);
//        //利用redisson实现分布式锁
//        //可重入锁，锁的名字是lock:order:用户id
//        RLock lock = redissonClient.getLock("lock:order:" + UserHolder.getUser().getId());
//        //尝试获取锁,无参数表示获取锁失败立即返回，true表示获取锁成功，false表示获取锁失败
//        if(!lock.tryLock()) {
//            //获取锁失败，返回错误信息
//            return Result.fail("你已经抢过了！");
//        }
//        try {
//            //这里需要使用代理技术，因为事务注解只能在被代理对象的方法上生效，而不能在当前对象的方法上生效，所以需要获取当前对象的代理对象来调用方法
//            IVoucherOrderService proxy = (IVoucherOrderService) AopContext.currentProxy();
//            return proxy.createVoucherOrder(voucherId);
//        } finally {
//            lock.unlock();
//        }
//    }



    /**
     * 实现一人一单
     * @param voucherOrder
     * @return
     */
    @Transactional
    public  void createVoucherOrder(VoucherOrder voucherOrder) {

         Long voucherId = voucherOrder.getVoucherId();
        Long userId = voucherOrder.getUserId();

        //添加一人一单功能
        int count = query().eq("user_id", userId)
                .eq("voucher_id", voucherId).count();
        if(count > 0){
            log.error("你已经抢过了！");
        }
        //方法三,标准答案
        //修改优惠券库存
        boolean success = seckillVoucherService.update().setSql("stock = stock - 1")
                .eq("voucher_id", voucherId)
                //乐观锁解决超卖问题，更新时判断库存是否大于0
                .gt("stock", 0).update();

        if (!success) {
            log.error("库存不足！");
        }
        //保存订单
        save(voucherOrder);
    }
//    /**
//     * 实现一人一单
//     * @param voucherId
//     * @return
//     */
//    @Transactional
//    public   Result createVoucherOrder(Long voucherId) {
//
//        //添加一人一单功能
//        int count = query().eq("user_id", UserHolder.getUser().getId())
//                .eq("voucher_id", voucherId).count();
//        if(count > 0){
//            return Result.fail("你已经抢过了！");
//        }
//        //下订单
//        long orderId = redisIdWorker.nextId("order");
//        //方法三,标准答案
//        //修改优惠券库存
//        boolean success = seckillVoucherService.update().setSql("stock = stock - 1")
//                .eq("voucher_id", voucherId)
//                //乐观锁解决超卖问题，更新时判断库存是否大于0
//                .gt("stock", 0).update();
//
//        if (!success) {
//            //扣减库存
//            return Result.fail("库存不足！");
//        }
//
//        //创建订单
//        VoucherOrder voucherOrder = new VoucherOrder();
//        voucherOrder.setId(orderId);
//        voucherOrder.setUserId(UserHolder.getUser().getId());
//        voucherOrder.setVoucherId(voucherId);
//        save(voucherOrder);
//        return Result.ok(orderId);
//    }
}
