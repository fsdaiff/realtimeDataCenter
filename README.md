#实时数据总线服务
##需求
1. 减少多进程实时数据传递的耦合性，提供消息传递路由。
2. 提供一次通知即可发布到所有关注该数据的进程中。
3. 提供统一的数据描述，使得各个进程不用重复描述大型数据表。
4. 多种语言客户端。
5. 可靠数据传递。
6. 可以应用于嵌入式级别硬件。

##项目概述
1. 作为嵌入式设备微服务架构的实时数据传递中心节点，减少各个业务进程的耦合，提高通用性。
2. 提供简单的通知者和接收者两类接口，简化通讯逻辑。在此基础上可以实现生产者+消费者模型通信方式和长轮询式通讯方式。
3. 提供实时数据的完整描述和各逻辑层服务启动时同步。
4. 提供socket接口和http+restfulapi接口。
5. 以数据为中心的通信，以集合论的思路传递传递数据，减少传输中不必要的数据，可以灵活地传输数据点或数据块甚至数据总集。
6. 在生产者+消费者通信模式中，为消费者提供带时间戳的内存缓冲，以便实时数据的同步。同时提供非阻塞的消息读取。
7. 每一次传递都带有身份标识。
8. 提供数据总集类型保护，每个进程维护数据总集中的某一块，也就是说每一个数据块最好只有一个写入者。
##基本运行流程
1. 初始化时，从config.json中导入json格式的数据总集模板，该树状数据总集包含通信中的所有已知数据。
2. 启动数据总集维护线程，负责接收数据写入，并通知关注该数据读取者，读取数据的值。并提供api。
3. 启动http驱动和socket-server通信协程。

##数据定位
按路径的方式访问数据总集中某个数据点或数据块
如/{根节点名称}/{一级容器节点名称}/{二级容器节点(数组名)}/{数组序号}/{三级级容器节点名称或者子节点名称}

##一次生产者+消费者模式数据传递的实现。
1. 用`访问者名称+访问路径`组合成的全局唯一标识，向总线服务申请监听某路径中的所有变化。（http接口在第一次访问时自动创建）
2. 当生产者改变该路径内的任意一点时，总线服务更新数据总集，并以json格式推送数据到各个消费者。（http是长轮询模式）

##启动建议
启动时所有业务进程第一步要获取数据总集，避免数据不一致。
由于各个进程启动顺序是随机的，数据总集中的数据块从无效状态转为有效状态也是随机的，建议在每个数据块中加入锁标识，避免其他进程由于数据无效导致主逻辑异常。
在自己关注数据块变为有效前不要进入业务主逻辑。

##其他
json表在数据量和层级比较大时，不好维护。所以制作了[excel2json](http://github.com/fsdaiff/excel2json)工具，利用excel维护数据。