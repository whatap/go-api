#  Go Monitoring  <whatap.io>

WhaTap Go Application Monitoring provides the monitoring service for the Go applications.

* It supports the web framework. It collects the data such as web transaction URL, response time, TPS information, error message, etc.
* It continuously collects data from the Go Runtime package. It also collects data including memory, goroutine, and GC related data.
* When multiple API/RPC services are called in the MSA environment, the call relationships are collected through link tracing.

WhaTap's Application Monitoring can monitor application in real time without reproducing failures.

* [document](https://docs.whatap.io/en/golang/introduction)

![scrrenshot](https://docs.whatap.io/assets/images/golang_system-8da823abb548e3c11b54bfc48ec7d9bb.png)


##  Agent Installation

####  Configuring the Go library

Add the github.com/whatap/go-api package to the Go application source code.

* [example](https://github.com/whatap/go-api-example)

```

go get github.com/whatap/go-api@latest

```

Configure for initialization and shutdown with the trace.Init() and trace.Shutdown() functions. Set the startup and end for transactions with the trace.Start() and trace.End() functions.

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

###  Download agent

An agent must be installed on the same server to forward data from the monitored application server through TCP communication and to transfer the data to the WhaTap collection server. The agent can be installed using the package.

**Note**
**The agent works as the default 127.0.0.1:6600 TCP server. It receives data from the Go application and forwards the data to the WhaTap collection server via the outbound 6600 port.**


1 Install the WhaTap repository.
2 Install the whatap-agent Linux package (yum, apt-get).
3 Run the whatap-agent service.

####  RedHat/CentOS

#####  Register the package repository

```
$  sudo  rpm  -Uvh  http://repo.whatap.io/centos/5/noarch/whatap-repo-1.0-1.noarch.rpm
```

#####  Install the package

```
$  sudo  yum  install  whatap-agent
```

####  Debian/Ubuntu

#####  Register the package repository

```
$  wget  http://repo.whatap.io/debian/release.gpg  -O  -|sudo  apt-key  add  -
$  wget  http://repo.whatap.io/debian/whatap-repo_1.0_all.deb
$  sudo  dpkg  -i  whatap-repo_1.0_all.deb
$  sudo  apt-get  update
```

#####  Install the package

```
$  sudo  apt-get  install  whatap-agent
```
  
####  Amazon  Linux

#####  Register the package repository

```
$  sudo  rpm  --import  http://repo.whatap.io/centos/release.gpg  
$  echo  "[whatap]"  |  sudo  tee  /etc/yum.repos.d/whatap.repo  >  /dev/null  
$  echo  "name=whatap  packages  for  enterprise  linux"  |  sudo  tee  -a  /etc/yum.repos.d/whatap.repo  >  /dev/null  
$  echo  "baseurl=http://repo.whatap.io/centos/latest/\$basearch"  |  sudo  tee  -a  /etc/yum.repos.d/whatap.repo  >  /dev/null
$  echo  "enabled=1"  |  sudo  tee  -a  /etc/yum.repos.d/whatap.repo  >  /dev/null  
$  echo  "gpgcheck=0"  |  sudo  tee  -a  /etc/yum.repos.d/whatap.repo  >  /dev/null
```

####  Install the package

```
$  sudo  yum  install  whatap-agent
```  

#### Alpine Linux

[whatap-agent.tar.gz]After downloading the file (https://s3.ap-northeast-2.amazonaws.com/repo.whatap.io/alpine/x86_64/whatap-agent.tar.gz), unzip the file based on the "/" directory. Create the monitoring file in the /usr/whatap/agent path.

```

$ wget https://s3.ap-northeast-2.amazonaws.com/repo.whatap.io/alpine/x86_64/whatap-agent.tar.gz
$ tar -xvzf whatap-agent.tar.gz -C /

```

#### Dockerfile

```

FROM golang:1.20

# install whatap-agent (x64)
RUN wget https://s3.ap-northeast-2.amazonaws.com/repo.whatap.io/alpine/x86_64/whatap-agent.tar.gz
RUN tar -xvzf whatap-agent.tar.gz -C /

# Create whatap.conf. You can copy it or attach it as a volume.
RUN echo "accesskey=aaasd-23432-123-" >> /app/whatap.conf
RUN echo "whatap.server.host=1.1.1.1/2.2.2.2" >> /app/whatap.conf

# Whatap/go-api library must be imported into the user application.
WORKDIR /app
ENV WHATAP_HOME /app
RUN go mod tidy
RUN go mod download -x
RUN go build -o ./app app.go

# Run whatap-agent and the user application together with the Docker execution command.
CMD ['sh', '-c', '/usr/whatap/agent/whatap-agent start && /app/app']

```

### Agent setting

Default settings
Execute the following commands in order to set the **access key** and **collection server IP** in whatap.conf.

* Create the whatap.conf file in the path of the application startup script.
* If the WHATAP_HOME environment variable has not been set, the path of the application startup script is recognized as the one of the whatap.conf file.

```

# Creation of whatap.conf in the script running path
$ echo "license={Access Key}" >> ./whatap.conf
$ echo "whatap.server.host={Collection Server IP}" >> ./whatap.conf
$ echo "app_name={Application Name}" >> ./whatap.conf

# Run application
./app

```

* license: Enter the access key.
* whatap.server.host: Enter the collection server IP address.
* app_name: Enter the application name. Set the user as a string.

**Note**
app_name is a component to determine the agent name. For more information, see the following.


#### Setting the **WHATAP_HOME** variable

You can set the whatap.conf path as the **WHATAP_HOME** variable. Create the **WHATAP_HOME** path first.

```
# Set the whatap.conf path after setting the WHATAP_HOME path.
mkdir ./whatap_home
echo "license={Access Key}" >> ./whatap_home/whatap.conf
echo "whatap.server.host={Collection Server IP}" >> ./whatap_home/whatap.conf
echo "app_name={Application Name}" >> ./whatap_home/whatap.conf

# Run the application.
WHATAP_HOME=./whatap_home ./app

```


#### Setting the agent names for each process

If one whatap.conf file is shared by the applications running in multiple processes, any changes may not be reflected. It is recommended to set a whatap.conf for each process.

To avoid duplicate agent names, you can add a string to each agent name for identification. The value set with the app_name option is added to the beginning of the agent name.

You can avoid duplicate names of agents run with the same command and the same instance.

```

# Set the whatap.conf path after setting the WHATAP_HOME path.
mkdir ./whatap_home
echo "license={Access Key}" >> ./whatap_home/whatap.conf
echo "whatap.server.host={Collection Server IP}" >> ./whatap_home/whatap.conf
echo "app_name={Application Name-1}" >> ./whatap_home/whatap.conf

# Run the application.
WHATAP_HOME=./whatap_home ./app 

# Set the whatap.conf path after setting the WHATAP_HOME path.
mkdir ./whatap_home1
echo "license={Access Key}" >> ./whatap_home1/whatap.conf
echo "whatap.server.host={Collection Server IP}" >> ./whatap_home1/whatap.conf
echo "app_name={Application Name-2}" >> ./whatap_home1/whatap.conf

# Run the application
WHATAP_HOME=./whatap_home1 ./app

```
