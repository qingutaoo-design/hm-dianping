--需要的参数
--1.优惠券id，判断库存是否充足
--2.用户id，需要知道谁下的单，可以用来做一些限制，比如说一个用户只能买一件，或者说这个用户之前买过了，就不能再买了
local voucherId = ARGV[1]
local userId = ARGV[2]
local orderId = ARGV[3]

local stockKey = "seckill:stock:" .. voucherId
local orderKey = "seckill:order:" .. voucherId

--判断库存是否充足
if(tonumber(redis.call('get',stockKey)) <= 0 )then
    return  1
end
--判断用户是否已经购买过了
if(redis.call('sismember',orderKey,userId) == 1)then
    return 2
end
--扣库存
redis.call('incrby',stockKey,-1)
--保存用户
redis.call('sadd',orderKey,userId)
return 0