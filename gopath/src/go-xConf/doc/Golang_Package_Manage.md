## Golang包管理 ##


<table class="table-bordered table-striped">
	<tr>
		<td colspan="4">
		1、go get
		</td>
	</tr>

   <tr>
      <td>
         优点
      </td>
      <td colspan="3">
        1、使用简单<br>
		2、go tool 原生支持
      </td>
   </tr>
   <tr>
      <td>
         缺点
      </td>
      <td colspan="3">
        1、缺乏明确显示的版本。团队开发容易导入不一样的版本<br>
		2、第三方包没有内容安全审计，很容易引入代码 Bug<br>
		3、依赖的完整性无法校验，程序编译时无法保障百分百成功<br>
		4、不能方便地隔离不同项目的环境
      </td>
   </tr>

</table>

<table class="table-bordered table-striped">
	<tr>
		<td colspan="4">
		2、vendor
		</td>
	</tr>

   <tr>
      <td>
         优点
      </td>
      <td colspan="3">
        1、可靠、稳定<br>
		2、任何时间点编出的程序都是一致的<br>
      </td>
   </tr>
   <tr>
      <td>
         缺点
      </td>
      <td colspan="3">
        1、升级依赖库不方便<br>
		2、修复依赖库的bug困难<br>
		3、版本控制困难<br>
      </td>
   </tr>

</table>

<table class="table-bordered table-striped">
	<tr>
		<td colspan="4">
		3、使用其他各种包管理(Glide | godep | govendor)
		</td>
	</tr>

   <tr>
      <td>
         优点
      </td>
      <td colspan="3">
        1、可保证依赖包的版本<br>
		2、任何时间点编出的程序都是一致的<br>
		3、每个包都可以采用独立的包，很方便控制包的版本<br>
      </td>
   </tr>
   <tr>
      <td>
         缺点
      </td>
      <td colspan="3">
        1、使用上有一定门槛限制<br>
		2、需要在众多良莠不齐的包管理中选出适合自己的<br>
      </td>
   </tr>

</table>


<table class="table-bordered table-striped">
	<tr>
		<td colspan="4">
		4、git submodule
		</td>
	</tr>

   <tr>
      <td>
         优点
      </td>
      <td colspan="3">
        1、一定程度上保证依赖包的版本<br>
		2、任何时间点编出的程序都是一致的<br>
      </td>
   </tr>
   <tr>
      <td>
         缺点
      </td>
      <td colspan="3">
        1、git submodule通过下载到项目的指定目录下，作为开源项目，就算通过git submodule下载<br>
		到了指定的包，但是集成者还是无法正确的引入他们所需要的包，还需要设置goPath才能找到,<br>
		同时，如果在他所设置的gopath之前的path存在相关的package，就会优先使用之前的package，<br>
		也无法做到相关的包的版本控制，同时，还增加了集成的复杂度。<br>
      </td>
   </tr>

</table>


1、目前网上一些小的开源库采用的为：go get获取依赖的库

例如：

    https://github.com/outbrain/zookeepercli  build.sh

2、github上大型go项目大多数还都是原生的golang，采用系统的库

    seelog zookeeper等包

3、使用包管理的github项目


    https://github.com/aws/aws-sdk-go
	
	https://github.com/jpillora/chisel 

    https://github.com/fatih/color 

	https://github.com/zquestz/s/ 

如果想要作为开源项目提供出去，我认为采用第三种方案比较合适：

1、可保证我们依赖的package准确。

2、同时方便版本的更新的。

3、各个项目之间可以相互独立，互不影响。

-----------------
## 针对golang的包管理工具的选择##

几乎所有的包管理工具都是针对vendor的优化。

### vendor ###

官方语言特性，这个特性在 1.5 版本作为实验特性被添加，1.6 中默认被启用，1.7 移除变量加入标准中。

当我们使用go的一些命令时,例如go build 或者 go install，go首先会检查依赖库是否存在于./vendor/文件下，如果存在就是用它，如果在vendor目录下没有找到依赖库，则会去$GOPATH/src/目录下。

所以一下情况也是可以通过编译的。

A公司开发的PackageA 使用到了github的PackageG，同时将其开源出去。

B公司使用了A公司开源出去的PackageA，同时也使用了github上的PackageG（但是与A公司使用的为不同的版本）

只要A公司把PackageG存放在./Vendor目录下，B公司无论如何存放他们所使用的的PackageG(vendor 或者 GOPATH下)，都不会存在冲突。都可以正常使用。

#### 原理： ####

将源码拷贝到当前工程的vendor目录下，这样打包当前的工程代码到任意机器的$GOPATH/src下都可以通过编译，避免出现由于项目代码外部依赖过多，在迁移后，需要多次go get依赖包，防止通过go get重新拉取的依赖包的版本可能与工程开发时使用的不一致，导致编译错误问题。

#### 缺陷： ####
无法精确的引用外部包并进行版本控制，不能指定某一个特定版本的外部包，只是在当初选择依赖包时，将其拷贝到vendor目录下，但是一旦外部包需要升级,vendor下的代码不会自动跟着升级。

vendor没有相关元文件记录引用包的版本信息，缺少元文件的信息，对于以后外部包升级、调研产生很大的问题，无法评估升级带来的风险；

为了改进vendor的缺陷，github上很多包管理软件对此进行了优化。



以下为github上golangTop1000的项目中所使用到的包管理统计情况。

所使用的分析数据来源以下网站

    https://github.com/blindpirate/report-of-build-tools-for-java-and-golang

**包管理排行**

|--|--|--|
|Tool Name|Reference Count|URL|
|godep|119| [https://github.com/tools/godep](https://github.com/tools/godep)
|govendor|65|[https://github.com/kardianos/govendor](https://github.com/kardianos/govendor)
|glide|64|[https://github.com/Masterminds/glide](https://github.com/Masterminds/glide)
|gvt|25|
|submodule|8|
|gpm/johnny-deps|7|
|trash	|7|
|glock	|5|
|gom	|4|
|gopack	|3|
|gopm	|3|
|gvend	|2|
|goop	|1|


我们根据热度选择最火的**TOP3**，选取godep，govendor，glide三种包管理，来进行详细的分析比对


### godep ###

#### 使用说明 ####
	
	go get github.com/tools/godep

	godep save //godep会保存依赖关系到Godeps/Godeps.json 依赖文件会保存到./vendor/下
	//老版本的go会保存在./Godeps/_workspace/下

	//所支持的命令
	save     list and copy dependencies into Godeps
    go       run the go tool with saved dependencies
    get      download and install packages with specified dependencies
    path     print GOPATH for dependency code
    restore  check out listed dependency versions in GOPATH
    update   update selected packages or the go version
    diff     shows the diff between current and previously saved set of dependencies
    version  show version info


通过对Godeps.json文件进行版本管理，即可管理整个项目的第三方包的依赖信息。

#### 添加新包 ####

1. go get 把新增的第三方包get到GOPATH的src目录下，然后再执行godep save
2. godep get 同样是把第三方包get到GOPATH的src下，然后再执行godep save

通过实验，godep只是把第三方包进行单独到依赖管理，而新增到第三包还是会被get到GOPATH中；

在这样的情况下，如果多个项目同时使用第三方依赖包的不同版本时，显然不能满足。


----


### govendor ###

vendor的升级版，相对于govendor具有以下优势，


- 可以平滑的将现有非vendor项目转换为vendor项目

	    govendor init
	    govendor add  inport_out_packagename


- 会生成一个元数据文件，记录项目工程依赖的外部包，以及其版本信息，方便以后持续更新和维护

		vendor/vendor.json


- 提供命令查看整个工程的依赖关系

		goverdor --list 
		goverdor --list -v

#### 使用方式 ####

    # Setup your project.初始化项目
    cd "my project in GOPATH"
    govendor init
    
    # Add existing GOPATH files to vendor.初始化GOPATH
    govendor add +external
    
    # View your work.
    govendor list
    
    # Look at what is using a package
    govendor list -v fmt
    
    # Specify a specific version or revision to fetch
    govendor fetch golang.org/x/net/context@a4bbce9fcae005b22ae5443f6af064d80a6f5a55
    govendor fetch golang.org/x/net/context@v1   # Get latest v1.*.* tag or branch.
    govendor fetch golang.org/x/net/context@=v1  # Get the tag or branch named "v1".
    
    # Update a package to latest, given any prior version constraint
    govendor fetch golang.org/x/net/context
    
    # Format your repository only
    govendor fmt +local
    
    # Build everything in your repository only
    govendor install +local
    
    # Test your repository only
    govendor test +local

用govendor fetch <url1> <url2>新增的第三方包直接被get到根目录的vendor文件夹下,不会进入到GOPATH下，所以不会与其它的项目混用第三方包，完美避免多个项目同用同一个第三方包的不同版本问题。

govendor get 与go get功能相识，但是能直接下载依赖包到vendor文件中。

只需要对vendor/vendor.json进行版本控制，即可对第三包依赖关系进行控制。


------

### glide ###

使用方法
	
	//glide 所支持的命令

    COMMANDS:
	     create, init			Initialize a new project, creating a glide.yaml file
	     config-wizard, cw		Wizard that makes optional suggestions to improve config in a glide.yaml file.
	     get					Install one or more packages into `vendor/` and add dependency to glide.yaml.
	     remove, rm				Remove a package from the glide.yaml file, and regenerate the lock file.
	     import					Import files from other dependency management systems.
	     name					Print the name of this project.
	     novendor, nv			List all non-vendor paths in a directory.
	     rebuild				Rebuild ('go build') the dependencies
	     install, i				Install a project's dependencies
	     update, up				Update a project's dependencies
	     tree					(Deprecated) Tree prints the dependencies of this project as a tree.
	     list					List prints all dependencies that the present code references.
	     info					Info prints information about this project
	     cache-clear, cc		Clears the Glide cache.
	     about					Learn about Glide
	     mirror					Manage mirrors
	     help, h				Shows a list of commands or help for one command
    
    GLOBAL OPTIONS:
       --yaml value, -y value	Set a YAML configuration file. (default: "glide.yaml")
       --quiet, -q				Quiet (no info or debug messages)
       --debug					Print debug verbose informational messages
       --home value				The location of Glide files (default: "/home/majortom/.glide") [$GLIDE_HOME]
       --tmp value				The temp directory to use. Defaults to systems temp [$GLIDE_TMP]
       --no-color				Turn off colored output for log messages
       --help, -h				show help
       --version, -v			print the version


glide 通过glide create或glide init命令初始化第三方包管理，会在项目根目录下生成一个glide.yaml，这个文件记录用到的第三方包的依赖关系，支持编辑修改。

glide通过glide install, 会把所有缺少的第三方包都下载到vendor文件夹下，并且会在glide.yaml中添加所有依赖的第三方包名称，在glide.lock文件中记录具体的版本管理信息。

### 总结 ###

首先，godep,govendor,glide 三种工具都可以很好的进行包管理。

#### 命令行支持上 ####

govendor,glide提供的可操作命令更丰富，godep则较弱。

#### 文件结构上 ####

1. godep 会在根目录生成Godeps和vendor两个文件夹; 
2. govendor把所有信息都生成在vendor目录下;
3. glide 会在根目录下生成glide.yaml, glide.lock文件及vendor目录; 


#### 简洁 ####

从简洁度和尽量不污染项目来看，govendor最优，glide次之。

#### 安装使用上 ####

godep, govendor, glide 都提供go get 第三方包的命令。

但是glide的glide提供多平台上工具的直接下载和使用。十分方便，并且支持直接把第三方包get到本项目的vendor目录下，并且glide提供的便捷命令也丰富。

#### 多项目支持上 ####
多项目支持上，主要是指，多个项目引用同一个依赖包的不同版本的情况。

1. glide和govendor都可以很好的支持多项目公用不同的版本依赖库问题
2. godep 由于下载包到GOPATH中，所以对于多项目引用同一依赖包的不同版本不太友好。

**综上所述：**

我个人的意见为：

1. 在实际生产的环境中使用govendor, 项目文档中更简洁;
2. 尝鲜可以使用glide，在试验项目中推荐试用glide, 更方便。

### 参考文章： ###


[1、https://github.com/golang/go/wiki/PackageManagementTools](https://github.com/golang/go/wiki/PackageManagementTools)

[2、https://github.com/blindpirate/report-of-build-tools-for-java-and-golang](https://github.com/blindpirate/report-of-build-tools-for-java-and-golang)

[3、http://stackoverflow.com/questions/37237036/how-should-i-use-vendor-in-go-1-6](http://stackoverflow.com/questions/37237036/how-should-i-use-vendor-in-go-1-6)
