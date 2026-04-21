package com.hmdp.service.impl;

import cn.hutool.core.bean.BeanUtil;
import com.baomidou.mybatisplus.core.conditions.query.QueryWrapper;
import com.baomidou.mybatisplus.core.toolkit.BeanUtils;
import com.hmdp.dto.Result;
import com.hmdp.dto.UserDTO;
import com.hmdp.entity.Follow;
import com.hmdp.entity.User;
import com.hmdp.mapper.FollowMapper;
import com.hmdp.service.IFollowService;
import com.baomidou.mybatisplus.extension.service.impl.ServiceImpl;
import com.hmdp.service.IUserService;
import com.hmdp.utils.UserHolder;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.stereotype.Service;

import javax.annotation.Resource;

import java.util.List;
import java.util.Set;
import java.util.stream.Collectors;

import static com.baomidou.mybatisplus.core.toolkit.Wrappers.query;

/**
 * <p>
 *  服务实现类
 * </p>
 *
 * @author 虎哥
 * @since 2021-12-22
 */
@Service
public class FollowServiceImpl extends ServiceImpl<FollowMapper, Follow> implements IFollowService {

    @Resource
    private StringRedisTemplate stringRedisTemplate;

    @Resource
    private IUserService userService;

    @Override
    public Result follow(Long followUserId, Boolean isFollow) {
            // 1.获取登录用户
            Long userId = UserHolder.getUser().getId();
            String key = "follows:" + userId;
            // 1.判断到底是关注还是取关
            if (isFollow) {
                // 2.关注，新增数据
                Follow follow = new Follow();
                follow.setUserId(userId);
                follow.setFollowUserId(followUserId);
                boolean isSuccess = save(follow);
                stringRedisTemplate.opsForSet().add(key, followUserId.toString());
            } else {
                // 3.取关，删除 delete from tb_follow where user_id = ? and follow_user_id = ?
                remove(new QueryWrapper<Follow>()
                        .eq("user_id", userId).eq("follow_user_id", followUserId));
                stringRedisTemplate.opsForSet().remove(key, followUserId.toString());
            }
            return Result.ok();
        }


    @Override
    public Result isFollow(Long followUserId) {
        // 1.获取登录用户
        Long userId = UserHolder.getUser().getId();
        if(userId == null){
            return Result.fail("请先登录");
        }
        // 2.查询是否关注 select count(*) from tb_follow where user_id = ? and follow_user_id = ?
        Integer count = query().eq("user_id", userId).eq("follow_user_id", followUserId).count();
        // 3.判断
        return Result.ok(count > 0);
    }

    @Override
    public Result followCommons(Long id) {
        Long currentId = UserHolder.getUser().getId();
        String key1 = "follows:" + currentId;
        String key2 = "follows:" + id;
        Set<String> followCommon = stringRedisTemplate.opsForSet().intersect(key2, key1);
        List<Long> collect = followCommon.stream().map(Long::valueOf).collect(Collectors.toList());
        List<UserDTO> userList = collect.stream()
                .map(userId -> BeanUtil.copyProperties(userService.query().eq("id", userId).one(),UserDTO.class))
                .collect(Collectors.toList());
        return Result.ok(userList);
    }
}
