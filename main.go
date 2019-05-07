package main

import (
	"flag"
	"time"

	rp "github.com/ying32/rproxy/librp"
)

var (
	tcpPort     = flag.Int("tcpport", 0, "Socket连接或者监听的端口")
	httpPort    = flag.Int("httpport", 0, "当mode为server时为服务端监听端口，当为mode为client时为转发至本地客户端的端口")
	rpMode      = flag.String("mode", "client", "启动模式，可选为client、server")
	svrAddr     = flag.String("svraddr", "127.0.0.1", "当mode为client时有效，为连接服务器的地址")
	verifyKey   = flag.String("vkey", "", "用作客户端与服务端连接时的校验")
	isHTTPS     = flag.Bool("ishttps", false, "httpPort端口是否只用作HTTPS监听")
	tlsCAFile   = flag.String("tlscafile", "", "当ishttps为true时，所需的CA根证书文件。可为空，根据实际情况确定")
	tlsCertFile = flag.String("tlscertfile", "", "当ishttps为true时，所需求的TLS证书文件")
	tlsKeyFile  = flag.String("tlskeyfile", "", "当ishttps为true时，所需求的TLS密匙文件")
	isZip       = flag.Bool("iszip", false, "是否开启zip压缩")
	configFile  = flag.String("cfgfile", "", "使用指定的配置文件中的参数，此时只有mode参数有效")
	isGUI       = flag.Bool("gui", false, "在客户端中使用GUI替代命令行，只支持mode=client")
)

func main() {
	flag.Parse()

	mode := *rpMode

	rpConfig := new(rp.TRProxyConfig)

	if *configFile != "" {
		err := rp.LoadConfig(*configFile, rpConfig)
		if err != nil {
			rp.Log.EF("加载配置失败：", err)
		}
	} else {
		// 初始填充配置
		rpConfig.TCPPort = *tcpPort
		switch mode {
		case "server":
			rpConfig.Server.HTTPPort = *httpPort

		case "client":
			rpConfig.Client.HTTPPort = *httpPort
			rpConfig.Client.SvrAddr = *svrAddr
		}
		rpConfig.VerifyKey = *verifyKey
		rpConfig.IsHTTPS = *isHTTPS
		rpConfig.TLSCAFile = *tlsCAFile
		rpConfig.TLSCertFile = *tlsCertFile
		rpConfig.TLSKeyFile = *tlsKeyFile
		rpConfig.IsZIP = *isZip
	}

	if rpConfig.VerifyKey == "" {
		rp.Log.EF("必须输入一个验证的key")
	}
	if rpConfig.TCPPort <= 0 || rpConfig.TCPPort >= 65536 {
		rp.Log.EF("请输入正确的tcp端口。")
	}
	rPort := rpConfig.Client.HTTPPort
	if mode == "server" {
		rPort = rpConfig.Server.HTTPPort
	}
	if rPort <= 0 || rPort >= 65536 {
		rp.Log.EF("请输入正确的http端口。")
	}
	if mode != "client" && mode != "server" {
		rp.Log.EF("请输入正确的服务启动模式")
	}
	if mode == "server" && rpConfig.TCPPort == rpConfig.Server.HTTPPort {
		rp.Log.EF("tcp端口与http端口不能为同一个。")
	}
	if rpConfig.IsHTTPS && (rpConfig.TLSCertFile == "" || rpConfig.TLSKeyFile == "") {
		rp.Log.EF("当为HTTPS时，TLS证书不能为空。")
	}

	// 初始配置文件
	rp.SetConfig(rpConfig)

	switch mode {

	case "server":
		rp.Log.I("TCP服务端已启动，端口：", rpConfig.TCPPort)
		if rpConfig.IsHTTPS {
			rp.Log.I("当前HTTP服务为HTTPS")
		}
		rp.Log.I("HTTP(S)服务端已开启，端口：", rpConfig.Server.HTTPPort)
		svr := rp.NewRPServer()
		if err := svr.Start(); err != nil {
			rp.Log.EF(err)
		}
		defer svr.Close()

	case "client":
	retry:
		rp.Log.I("客户端启动，连接服务器：", rpConfig.Client.SvrAddr, "， 端口：", rpConfig.TCPPort)
		if rpConfig.IsHTTPS {
			rp.Log.I("转发至HTTP服务为HTTPS")
		}
		rp.Log.I("转发至本地HTTP(S)端口：", rpConfig.Client.HTTPPort)
		cli := rp.NewRPClient()
		if err := cli.Start(); err != nil {
			rp.Log.E(err)
			// 重连
			rp.Log.I("5秒后重新连接...")
			time.Sleep(time.Second * 5)
			goto retry
		}
		defer cli.Close()

	}
}
