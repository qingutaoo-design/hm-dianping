package com.hmdp.entity;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.LocalDateTime;

@Data
@NoArgsConstructor
@AllArgsConstructor
public class RedisData<T> {
    private LocalDateTime expireTime;
    //使用泛型能够自动反序列化为对应的类
    private T data;
}
