#  와탭  Golang  모니터링  서비스  <whatap.io>

Golang  애플리케이션에  대한  모니터링  서비스를  제공합니다.

![scrrenshot](https://img.whatap.io/media/images/golang_system.png)


##  에이전트  설치  방식  개요

####  Golang  라이브러리
Golang  애플리케이션  소스코드에  whatap/go-api  를  추가하고  배포합니다.

```  
go get -u github.com/whatap/go-api
```

```
import  (
	"github.com/whatap/go-api/trace"
)

func  main(){
	trace.Init(nil)
	//It  must  be  executed  before  closing  the  app.
	defer  trace.Shutdown()
	
	...  
}
```

###  에이전트  설치

대상  애플리케이션에서  UDP  통신으로  데이터를  전달하고,  와탭  수집서버로  데이터를  전송하기  위해서는  같은  서버에  에이전트를  설치해야  합니다.

설치  방식은  패키지  설치로  가능합니다.

*  와탭  저장소(Repository)를  설치합니다.
*  whatap-agent  리눅스  패키지를(yum,  apt-get)  설치합니다.

에이전트는  whatap-agent  서비스(Service)로  실행됩니다.

####  RedHat/CentOS

#####  패키지  저장소(Repository)  등록

와탭  저장소(Repository)를  등록합니다.
```
$  sudo  rpm  -Uvh  http://repo.whatap.io/centos/5/noarch/whatap-repo-1.0-1.noarch.rpm
```
 #####  패키지  설치

아래  명령어를  통해  패키지를  설치합니다.

```
$  sudo  yum  install  whatap-agent
```

####  Debian/Ubuntu
#####  패키지  저장소(Repository)  등록

와탭  저장소(Repository)를  등록합니다.

```
$  wget  http://repo.whatap.io/debian/release.gpg  -O  -|sudo  apt-key  add  -
$  wget  http://repo.whatap.io/debian/whatap-repo_1.0_all.deb
$  sudo  dpkg  -i  whatap-repo_1.0_all.deb
$  sudo  apt-get  update
```

#####  패키지  설치

```
$  sudo  apt-get  install  whatap-agent
```
  
####  Amazon  Linux
#####  패키지  저장소(Repository)  등록

와탭  저장소(Repository)를  등록합니다.
```
$  sudo  rpm  --import  http://repo.whatap.io/centos/release.gpg  
$  echo  "[whatap]"  |  sudo  tee  /etc/yum.repos.d/whatap.repo  >  /dev/null  
$  echo  "name=whatap  packages  for  enterprise  linux"  |  sudo  tee  -a  /etc/yum.repos.d/whatap.repo  >  /dev/null  
$  echo  "baseurl=http://repo.whatap.io/centos/latest/\$basearch"  |  sudo  tee  -a  /etc/yum.repos.d/whatap.repo  >  /dev/null
$  echo  "enabled=1"  |  sudo  tee  -a  /etc/yum.repos.d/whatap.repo  >  /dev/null  
$  echo  "gpgcheck=0"  |  sudo  tee  -a  /etc/yum.repos.d/whatap.repo  >  /dev/null
```

####  패키지  설치

```
$  sudo  yum  install  whatap-agent
```  

####  라이센스  및  수집서버  설정

음 명령어를 차례로 실행해 *whatap.conf* 파일에 **액세스 키**와 **수집 서버 IP 주소** 등을 설정하세요.

- 애플리케이션의 시작 스크립트 경로에 *whatap.conf* 파일을 생성하세요.
- `WHATAP_HOME` 환경 변수를 설정하지 않으면 애플리케이션 시작 스크립트의 경로를 *whatap.conf* 파일 경로로 인식합니다.
/usr/whatap/agent  이하에  whatap.con다f  파일에  라이센스와  수집서버  정보를  설정합니다.  

```

# 스크립트 실행 경로에 whatap.conf 파일 생성
echo "license={액세스 키}" >> ./whatap.conf
echo "whatap.server.host={수집 서버 IP}" >> ./whatap.conf
echo "app_name={애플리케이션 이름}" >> ./whatap.conf

# 애플리케이션 실행
./app  

```