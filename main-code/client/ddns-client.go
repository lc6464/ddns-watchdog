package main

import (
	"ddns/client"
	"ddns/common"
	"flag"
	"fmt"
)

var (
	forcibly = flag.Bool("f", false, "强制检查 DNS 解析记录")
	moreTips = flag.Bool("mt", false, "显示更多的提示")
	version  = flag.Bool("version", false, "查看当前版本")
)

func main() {
	flag.Parse()

	// 加载配置
	conf := client.ClientConf{}
	getErr := common.IsDirExistAndCreate("./conf")
	if getErr != nil {
		fmt.Println(getErr)
		return
	}
	getErr = common.LoadAndUnmarshal("./conf/client.json", &conf)
	if getErr != nil {
		fmt.Println(getErr)
		// 这里不能 return
	}

	saveMark := false
	if conf.WebAddr == "" {
		conf.WebAddr = "https://yzyweb.cn/ddns"
		saveMark = true
	}
	if conf.LatestIP == "" {
		conf.LatestIP = "0.0.0.0"
		saveMark = true
	}
	if saveMark {
		getErr = common.MarshalAndSave(conf, "./conf/client.json")
		if getErr != nil {
			fmt.Println(getErr)
		}
		fmt.Println("请打开客户端配置文件 client.json 启用需要使用的服务\n并重新启动")
		// 需要用户手动设置
		return
	}
	if *version {
		client.CheckLatestVersion(conf)
		return
	}
	if !conf.EnableDdns {
		fmt.Println("请打开客户端配置文件 client.json 启用需要使用的服务\n并重新启动")
	}

	// 对比上一次的 IP
	if conf.WebAddr == "" {
		conf.WebAddr = "https://yzyweb.cn/ddns"
	}
	ipAddr, isIPv6, getErr := client.GetOwnIP(conf.WebAddr)
	if getErr != nil {
		fmt.Println(getErr)
		return
	}
	if ipAddr != conf.LatestIP || *forcibly {
		conf.LatestIP = ipAddr
		conf.IsIPv6 = isIPv6
		fmt.Println("你的公网 IP: ", ipAddr)
		getErr = common.MarshalAndSave(conf, "./conf/client.json")
		if getErr != nil {
			fmt.Println(getErr)
			return
		}
		if conf.EnableDdns {
			if conf.DNSPod {
				getErr = client.DNSPod(ipAddr)
				if getErr != nil {
					fmt.Println(getErr)
					return
				}
			}
		}
	} else {
		if *moreTips {
			fmt.Println("因为最新 IP 和当前文件记录的 IP 相同，所以跳过检查解析记录\n" +
				"若需要强制检查 DNS 解析记录，请添加启动参数 -f")
		}
	}
}