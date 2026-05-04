package com.hmdp.service.impl;

import cn.hutool.core.bean.BeanUtil;
import cn.hutool.json.JSONUtil;
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
import org.apache.kafka.clients.producer.ProducerRecord;
import org.redisson.api.RLock;
import org.redisson.api.RedissonClient;
import org.springframework.aop.framework.AopContext;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.core.io.ClassPathResource;
import org.springframework.data.redis.connection.stream.*;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.core.script.DefaultRedisScript;
import org.springframework.kafka.annotation.KafkaListener;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.kafka.support.Acknowledgment;
import org.springframework.kafka.support.SendResult;
import org.springframework.lang.Nullable;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.util.concurrent.ListenableFuture;

import javax.annotation.PostConstruct;
import javax.annotation.PreDestroy;
import javax.annotation.Resource;
import java.time.Duration;
import java.time.LocalDateTime;
import java.util.Collection;
import java.util.Collections;
import java.util.List;
import java.util.Map;
import java.util.concurrent.*;

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

    // 注入自身对象，解决方法内部调用事务失效的问题
    @Autowired
    @org.springframework.context.annotation.Lazy
    private IVoucherOrderService voucherOrderService;


//    private static final ExecutorService SECKILL_ORDER_EXECUTOR = Executors.newSingleThreadExecutor();
//
//    private volatile boolean running = true;

    //代理对象实现写数据库操作
    IVoucherOrderService proxy;
//    @PostConstruct
//    public void init(){
//        SECKILL_ORDER_EXECUTOR.submit(new VoucherOrderHandler());
//    }
//
//    @PreDestroy
//    public void destroy() {
//        running = false;
//        SECKILL_ORDER_EXECUTOR.shutdown();
//        try {
//            if (!SECKILL_ORDER_EXECUTOR.awaitTermination(5, TimeUnit.SECONDS)) {
//                SECKILL_ORDER_EXECUTOR.shutdownNow();
//            }
//        } catch (InterruptedException e) {
//            SECKILL_ORDER_EXECUTOR.shutdownNow();
//            Thread.currentThread().interrupt();
//        }
//    }

//    private class VoucherOrderHandler implements Runnable{
//
//        @Override
//        public void run() {
//            //优雅的关闭线程池，等待线程池中的任务执行完毕后再关闭线程池
//            while(running){
//                try {
//                    //从消息队列中获取订单信息  XREADGROUP GROUP group consumer [COUNT count] [BLOCK milliseconds] [NOACK] STREAMS key [key ...] ID [ID ...]
//                    List<MapRecord<String, Object, Object>> read = stringRedisTemplate.opsForStream().read(Consumer.from("g1", "c1"),
//                            StreamReadOptions.empty().count(1).block(Duration.ofSeconds(2)),
//                            StreamOffset.create("stream.orders", ReadOffset.lastConsumed())
//                    );
//                    //判断是否获取到订单信息
//                    if(read == null || read.isEmpty()){
//                        //没有获取到订单信息，继续下一次循环
//                        continue;
//                    }
//                    //获取到订单信息
//                    Map<Object, Object> value = read.get(0).getValue();
//                    VoucherOrder voucherOrder = BeanUtil.fillBeanWithMap(value, new VoucherOrder(), true);
//
//
//                    //创建订单
//                    //异步下单导致无法获取到代理对象，因为代理对象也是基于原线程创建的，所以只能通过成员变量获取代理对象
//                    //也是因为异步下单，导致无法获取到UserHolder中的用户信息，因为UserHolder中的用户信息是基于ThreadLocal存储的
//                    //可以根据订单信息中的用户id查询用户信息
//                    proxy.createVoucherOrder(voucherOrder);
//                    //ACK确认消息
//                    stringRedisTemplate.opsForStream().acknowledge("g1", "stream.orders", read.get(0).getId());
//                } catch (Exception e) {
//                    handlePendingList();
//                    log.error("处理订单异常", e);
//                }
//            }
//        }
//
//        private void handlePendingList() {
//            while(running){
//                try {
//                    //从pending-list中获取订单信息  XREADGROUP GROUP group consumer [COUNT count] [BLOCK milliseconds] [NOACK] STREAMS key [key ...] ID [ID ...]
//                    //区别在于偏移量，pending-list中的订单信息的偏移量是ReadOffset.from("0")，而不是ReadOffset.lastConsumed()
//                    List<MapRecord<String, Object, Object>> read = stringRedisTemplate.opsForStream().read(Consumer.from("g1", "c1"),
//                            StreamReadOptions.empty().count(1).block(Duration.ofSeconds(2)),
//                            StreamOffset.create("stream.orders", ReadOffset.from("0"))
//                    );
//                    //判断是否获取到订单信息
//                    if(read == null || read.isEmpty()){
//                        //没有获取到订单信息，继续下一次循环
//                        continue;
//                    }
//                    //获取到订单信息
//                    Map<Object, Object> value = read.get(0).getValue();
//                    VoucherOrder voucherOrder = BeanUtil.fillBeanWithMap(value, new VoucherOrder(), true);
//
//                    //创建订单
//                    //异步下单导致无法获取到代理对象，因为代理对象也是基于原线程创建的，所以只能通过成员变量获取代理对象
//                    //也是因为异步下单，导致无法获取到UserHolder中的用户信息，因为UserHolder中的用户信息是基于ThreadLocal存储的
//                    //可以根据订单信息中的用户id查询用户信息
//                    proxy.createVoucherOrder(voucherOrder);
//                    //ACK确认消息
//                    stringRedisTemplate.opsForStream().acknowledge("g1", "stream.orders", read.get(0).getId());
//                } catch (Exception e) {
//                    log.error("处理订单异常", e);
//                    //如果处理pending-list中的订单信息出现异常，继续下一次循环，避免阻塞
//                    try {
//                        Thread.sleep(20);
//                    } catch (InterruptedException ex) {
//                        ex.printStackTrace();
//                    }
//
//                }
//            }
//        }
//    }

    @Resource
    private KafkaTemplate<String,String> kafkaTemplate;
    //从kafka消息队列中获取订单信息
    //可在配置文件中解耦，为了方便测试，直接写在代码当中
    @KafkaListener(groupId = "my-kafka-group" , topics = "kafka-orders",concurrency = "3")
    void onEvents(String event, Acknowledgment ack){
        System.out.println("消费者线程id: " + Thread.currentThread().getId() + "，消费订单消息: " + event);
        VoucherOrder voucherOrder = JSONUtil.toBean(event, VoucherOrder.class);
        // 使用注入的代理对象调用，保证 @Transactional 生效
        voucherOrderService.createVoucherOrder(voucherOrder);
        ack.acknowledge();
    }


    private static final DefaultRedisScript<Long> SECKILL_SCRIPT ;
    static {
        SECKILL_SCRIPT = new DefaultRedisScript<>();
        SECKILL_SCRIPT.setLocation(new ClassPathResource("seckill.lua"));
        SECKILL_SCRIPT.setResultType(Long.class);
    }


//    使用kafka生产者将订单信息发送到kafka消息队列中
    //使用kafka的回调函数异步获取发送结果，发送成功后返回订单id，发送失败后返回错误信息
    @Override
    public Result orderSeckillVoucher(Long voucherId) {
        //创建订单id
        long orderId = redisIdWorker.nextId("order");

        //执行lua脚本，判断用户是否下单成功，并且将订单信息保存到redis的stream消息队列当中
        Long result = stringRedisTemplate.execute(SECKILL_SCRIPT,
                Collections.EMPTY_LIST,
                //添加订单id
                voucherId.toString(), UserHolder.getUser().getId().toString(),String.valueOf(orderId));
        //返回结果为0，说明抢到了
        //返回结果为1，说明库存不足
        //返回结果为2，说明用户已经抢过了
        if(result == 1L){
            return Result.fail("库存不足！");
        } else if(result == 2L){
            return Result.fail("你已经抢过了！");
        }

        //此时断定抢到了,将消息放入kafka消息队列中
        //使用回调函数不阻塞当前线程，异步获取发送结果，发送成功后返回订单id，发送失败后返回错误信息

        VoucherOrder voucherOrder = VoucherOrder.builder()
                .voucherId(voucherId)
                .userId(UserHolder.getUser().getId()).id(orderId).build();
        String voucherOrderJSON = JSONUtil.toJsonStr(voucherOrder);

        ListenableFuture<SendResult<String, String>> future = kafkaTemplate.send("kafka-orders", voucherOrderJSON);
         future.completable()
                .thenAccept(producerResult -> {
                    ProducerRecord<String, String> producerRecord = producerResult.getProducerRecord();
                    System.out.println("生产者发送结果:" + producerRecord.toString());
                })
                .exceptionally(ex -> {
                    log.error("Kafka发送订单消息失败", ex);
                    return null;
                });

        //抢到了，将订单信息放入消息队列当中
        //抢到了，返回订单id
        return Result.ok(orderId);
    }


//    //从redis的stream消息队列中进行异步下单
//    @Override
//    public Result orderSeckillVoucher(Long voucherId) {
//        //创建订单id
//        long orderId = redisIdWorker.nextId("order");
//
//        //执行lua脚本，判断用户是否下单成功，并且将订单信息保存到redis的stream消息队列当中
//        Long result = stringRedisTemplate.execute(SECKILL_SCRIPT,
//                Collections.EMPTY_LIST,
//                //添加订单id
//                voucherId.toString(), UserHolder.getUser().getId().toString(),String.valueOf(orderId));
//        //返回结果为0，说明抢到了
//        //返回结果为1，说明库存不足
//        //返回结果为2，说明用户已经抢过了
//        if(result == 1L){
//            return Result.fail("库存不足！");
//        } else if(result == 2L){
//            return Result.fail("你已经抢过了！");
//        }
//
//
//        //抢到了，将订单信息放入消息队列当中
//
//        //赋值代理对象
//        proxy = (IVoucherOrderService) AopContext.currentProxy();
//        //异步下单
//
//        //抢到了，返回订单id
//        return Result.ok(orderId);
//    }


//    /**
//     * 下单秒杀优惠券
//     * @param voucherId
//     * @return
//     */

//    private BlockingQueue<VoucherOrder> orderTasks = new ArrayBlockingQueue<>(1024 * 1024);
//    @Override
//    public Result orderSeckillVoucher(Long voucherId) {
//
//        Long result = stringRedisTemplate.execute(SECKILL_SCRIPT,
//                Collections.EMPTY_LIST,
//                voucherId.toString(), UserHolder.getUser().getId().toString());
//        //返回结果为0，说明抢到了
//        //返回结果为1，说明库存不足
//        //返回结果为2，说明用户已经抢过了
//        if(result == 1L){
//            return Result.fail("库存不足！");
//        } else if(result == 2L){
//            return Result.fail("你已经抢过了！");
//        }
//
//        //创建订单
//        long orderId = redisIdWorker.nextId("order");
//
//        VoucherOrder voucherOrder = new VoucherOrder();
//        voucherOrder.setId(orderId);
//        voucherOrder.setUserId(UserHolder.getUser().getId());
//        voucherOrder.setVoucherId(voucherId);
//
//
//        //抢到了，将订单信息放入阻塞队列
//        orderTasks.add(voucherOrder);
//        //赋值代理对象
//        proxy = (IVoucherOrderService) AopContext.currentProxy();
//        //异步下单
//
//        //抢到了，返回订单id
//        return Result.ok(orderId);
//    }

//    /**
//     * 下单秒杀优惠券
//     * @param voucherId
//     * @return
//     */
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
