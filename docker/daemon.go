// +build daemon

package main

import (
	"log"

	"github.com/docker/docker/builtins"
	"github.com/docker/docker/daemon"
	_ "github.com/docker/docker/daemon/execdriver/lxc"
	_ "github.com/docker/docker/daemon/execdriver/native"
	"github.com/docker/docker/dockerversion"
	"github.com/docker/docker/engine"
	flag "github.com/docker/docker/pkg/mflag"
	"github.com/docker/docker/pkg/signal"
)

const CanDaemon = true

var (
	daemonCfg = &daemon.Config{}
)
//关于Docker Daemon所需要的config配置信息均已经初始化完毕
func init() {
	daemonCfg.InstallFlags()
}

func mainDaemon() {
	/*
	第一个步骤即flag参数的检查。具体而言，即当docker命令经过flag参数解析之后，判断剩余的参数是否为0。
	若为0，则说明Docker Daemon的启动命令无误，正常运行；
	若不为0，则说明在启动Docker Daemon的时候，传入了多余的参数，此时会输出错误提示，并退出运行程序。
	 */
	if flag.NArg() != 0 {
		flag.Usage()
		return
	}
	eng := engine.New()   //创建engine	
	signal.Trap(eng.Shutdown)
	// Load builtins
	if err := builtins.Register(eng); err != nil {
		log.Fatal(err)
	}

	// load the daemon in the background so we can immediately start
	// the http api so that connections don't fail while the daemon
	// is booting
	go func() {
		d, err := daemon.NewDaemon(daemonCfg, eng)
		if err != nil {
			log.Fatal(err)
		}
		if err := d.Install(eng); err != nil {
			log.Fatal(err)
		}
		// after the daemon is done setting up we can tell the api to start
		// accepting connections
		if err := eng.Job("acceptconnections").Run(); err != nil {
			log.Fatal(err)
		}
	}()
	// TODO actually have a resolved graphdriver to show?
	log.Printf("docker daemon: %s %s; execdriver: %s; graphdriver: %s",
		dockerversion.VERSION,
		dockerversion.GITCOMMIT,
		daemonCfg.ExecDriver,
		daemonCfg.GraphDriver,
	)

	// Serve api
	job := eng.Job("serveapi", flHosts...)
	job.SetenvBool("Logging", true)
	job.SetenvBool("EnableCors", *flEnableCors)
	job.Setenv("Version", dockerversion.VERSION)
	job.Setenv("SocketGroup", *flSocketGroup)

	job.SetenvBool("Tls", *flTls)
	job.SetenvBool("TlsVerify", *flTlsVerify)
	job.Setenv("TlsCa", *flCa)
	job.Setenv("TlsCert", *flCert)
	job.Setenv("TlsKey", *flKey)
	job.SetenvBool("BufferRequests", true)
	if err := job.Run(); err != nil {
		log.Fatal(err)
	}
}
