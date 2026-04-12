package com.hmdp;

import com.hmdp.service.impl.ShopServiceImpl;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;

import java.math.BigDecimal;

@SpringBootTest
class HmDianPingApplicationTests {

    @Autowired
    ShopServiceImpl service;

    @Test
    public void test(){
        service.rebuildCacheWithLogicExpire(1L,10L);
    }

}
