# changelog

## [2019-12-04]

- mq增加mq_gpstiming，用于接收mq的gps校时数据，对本地系统进行对时

## [2019-11-11]

- 修正rabbitmq未启动时，状态检查的bug
- sql,redis,etcd,rmq启动时返回是否成功

## [2019-11-09]

- 使用interface传递封装库日志
- 增加CheckUUID中间件，用于获取用户名
- rmq采用服务名自动设置队列，可设置为随机队列名

## [2019-10-23]

- 增加公共变量rootPath
- 隐藏mq和redis的client变量

## [2019-10-08]

- etcd增加etcd_root 配置项，用于设置etcd注册根路径

## [2019-09-11]

- rabbitmq增加ssl支持

## [2019-09-30]

- 增加EraseAllRedis方法
- WriteRedis增加返回error

## [2019-09-09]

- 使用vendor管理gopsu包

## [2019-09-03]

- init
- 增加各个组件可用状态判断
- 增加mq消费者初始化
- https可设置为单项认证
