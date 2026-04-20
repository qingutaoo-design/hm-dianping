package com.hmdp.config;


import com.hmdp.entity.AliOssProperties;
import com.hmdp.utils.AliOssUtil;
import lombok.extern.slf4j.Slf4j;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
@Slf4j
public class CommonConfig {
    /**
     * 创建AliOssUtil对象，并将其注册到Spring容器中
     * @param aliOssProperties
     * @return
     */
    @Bean
    public AliOssUtil aliOssUtil(AliOssProperties aliOssProperties) {
        log.info("正在创建AliOssUtil对象...");
        return new AliOssUtil(aliOssProperties.getEndpoint(), aliOssProperties.getAccessKeyId(), aliOssProperties.getAccessKeySecret(), aliOssProperties.getBucketName());
    }
}
