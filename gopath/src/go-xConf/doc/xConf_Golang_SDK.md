## xConf Golang SDK##

配置中心SDK包含NewService，Init，Fini，Register，Deregister，GetCfg，GetSrvNodes 6个方法，实现服务发现和配置更新的功能，两种功能可独立使用，互不影响。
下面，从这六个方法，依次介绍入手

### 实例创建 ###

    func NewService(srv string, tags []string, addr string) (*Service, error)

配置中心服务的实例获取，传入的参数为：srv需要集成的服务名，tags为服务的标签，addr为本机的ip：port。返回值为配置中心实例和err。

如果srv为空，返回错误；

tags目前可以为空；

如果addr为空时，默认获取第一块网卡的**ip：xxxx**，请确保addr传入的值正确，因为后面服务注册和feedback都会使用

----------

### 配置中心初始化 ###

    func (s *Service) Init(zkAddr []string, requestAddr []string) (err error)

配置服务初始化，传入参数为zookeeper的地址，xConf-web服务的地址。返回err。

目前，集成者保证传入zk地址的准确性，SDK保证传入地址的连接。

xConf-web地址，目前支持多个地址的传入，并且在一个地址有错的情况下，选择下一个地址的简单策略。

若返回err不为nil，表示初始化错误

----------

### 服务注册 ###

    func (s *Service) Register() (err error) 

根据在NewService中传入的srv（服务名称）和addr（服务ip：port）注册相关的服务；

若返回err不为nil，表示服务注册失败。

-----------

### 服务下线 ###

	`(s *Service) Deregister() (err error) `

服务下线操作，从zookeeper相关节点中去除该注册节点。

若返回err不为nil，表示服务下线失败

----------

### 获取配置 ###

    func (s *Service) GetCfg(cfgHandler CfgHandler) (result map[string][]byte, err error)

    // config update Handler
	type CfgHandler interface {
		HandlerCfgUpdate(cfgMsg map[string][]byte, err error) error
	}

配置更新获取，集成方实现HandlerCfgUpdate方法。

传入回调函数，可实现动态配置的更新回调，调用接口返回的result为全部的配置内容，包括动态的静态的配置文件。

随后动态配置文件的修改、更新会通过回调函数通知

注：**在web上传的文件名不可以一样（动态、静态配置文件），否则会出现配置文件丢失的情况。**

--------

### 服务发现 ###

    func (s *Service) GetSrvNodes(srv string, tags []string, srvHandler SrvHandler) (result []string, err error)

    type SrvHandler interface {
	//TODO add srv and tags
		HandlerSvrUpdate(srvMsg SrvMessage, err error) error
	}

服务发现接口，传入srv（服务名称），tags，回调函数，首次会获取到当前的服务节点ip：port；

随后，如果srv下节点有变化时，会通过用户传入的回调函数通知用户。

-------
### 逆初始化 ###

    func (s *Service) Fini()error

配置中心逆初始化，释放资源，断开zookeeper Conn


## SDK设计细节 ##

### dumpfile & log ###

dumpfile存储在 ./xConf/config/srv(项目名) 

log存储在./xConf/log/srv（项目名）下

按照srv独立分开，便于后期问题查找

### xConfDebug选项 ###

在集成sdk时，一般只打普通的info和error日志；

如果想查看更多细节的日志，则可以在命令行的最后一个参数键入xConfDebug；

例如集成配置中心的IM项目

    ./IM xConfDebug

通过最后一个xConfDebug参数，控制输出到文件的日志等级（包含trace，debug，info，error等全部信息）