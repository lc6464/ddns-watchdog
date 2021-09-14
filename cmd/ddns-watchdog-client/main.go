package main

import (
	"ddns-watchdog/internal/client"
	"errors"
	"flag"
	"log"
	"time"
)

var (
	installOption   = flag.Bool("I", false, "安装服务")
	uninstallOption = flag.Bool("U", false, "卸载服务")
	enforcement     = flag.Bool("f", false, "强制检查 DNS 解析记录")
	version         = flag.Bool("v", false, "查看当前版本并检查更新")
	initOption      = flag.String("i", "", "有选择地初始化配置文件，可以组合使用 (例 01)\n"+
		"0 -> "+client.ConfFileName+"\n"+
		"1 -> "+client.DNSPodConfFileName+"\n"+
		"2 -> "+client.AliDNSConfFileName+"\n"+
		"3 -> "+client.CloudflareConfFileName)
	confPath = flag.String("c", "", "指定配置文件路径 (最好是绝对路径)(路径有空格请放在双引号中间)")
)

func main() {
	// 初始化并处理 flag
	exit, err := runFlag()
	if err != nil {
		log.Fatal(err)
	}
	if exit {
		return
	}

	// 加载服务配置
	err = runLoadConf()
	if err != nil {
		log.Fatal(err)
	}

	// 周期循环
	waitCheckDone := make(chan bool, 1)
	if client.Conf.CheckCycleMinutes <= 0 {
		go asyncCheck(waitCheckDone)
		<-waitCheckDone
	} else {
		cycle := time.NewTicker(time.Duration(client.Conf.CheckCycleMinutes) * time.Minute)
		for {
			go asyncCheck(waitCheckDone)
			<-waitCheckDone
			<-cycle.C
		}
	}
}

func runFlag() (exit bool, err error) {
	flag.Parse()
	// 加载自定义配置文件路径
	if *confPath != "" {
		tempStr := *confPath
		if tempStr[len(tempStr)-1:] != "/" {
			tempStr = tempStr + "/"
		}
		client.ConfPath = tempStr
	}

	// 有选择地初始化配置文件
	if *initOption != "" {
		for _, event := range *initOption {
			err = runInitConf(string(event))
			if err != nil {
				return
			}
		}
		exit = true
		return
	}

	// 加载客户端配置
	// 不得不放在这个地方，因为有下面的检查版本和安装 / 卸载服务
	err = client.Conf.LoadConf()
	if err != nil {
		return
	}

	// 检查版本
	if *version {
		client.Conf.CheckLatestVersion()
		exit = true
		return
	}

	// 安装 / 卸载服务
	switch {
	case *installOption:
		err = client.Install()
		if err != nil {
			return
		}
		exit = true
		return
	case *uninstallOption:
		err = client.Uninstall()
		if err != nil {
			return
		}
		exit = true
		return
	}
	return
}

func runInitConf(event string) error {
	switch event {
	case "0":
		msg, err := client.Conf.InitConf()
		if err != nil {
			return err
		}
		log.Println(msg)
	case "1":
		msg, err := client.Dpc.InitConf()
		if err != nil {
			return err
		}
		log.Println(msg)
	case "2":
		msg, err := client.Adc.InitConf()
		if err != nil {
			return err
		}
		log.Println(msg)
	case "3":
		msg, err := client.Cfc.InitConf()
		if err != nil {
			return err
		}
		log.Println(msg)
	default:
		err := errors.New("你初始化了一个寂寞")
		return err
	}
	return nil
}

func runLoadConf() (err error) {
	if client.Conf.Services.DNSPod {
		err = client.Dpc.LoadCOnf()
		if err != nil {
			return
		}
	}
	if client.Conf.Services.AliDNS {
		err = client.Adc.LoadConf()
		if err != nil {
			return
		}
	}
	if client.Conf.Services.Cloudflare {
		err = client.Cfc.LoadConf()
		if err != nil {
			return
		}
	}
	return
}

func asyncCheck(done chan bool) {
	// 获取 IP
	ipv4, ipv6, err := client.GetOwnIP(client.Conf.Enable, client.Conf.APIUrl, client.Conf.NetworkCard)
	if err != nil {
		log.Println(err)
		done <- true
		return
	}

	// 进入更新流程
	if ipv4 != client.Conf.LatestIPv4 || ipv6 != client.Conf.LatestIPv6 || *enforcement {
		if ipv4 != client.Conf.LatestIPv4 {
			client.Conf.LatestIPv4 = ipv4
		}
		if ipv6 != client.Conf.LatestIPv6 {
			client.Conf.LatestIPv6 = ipv6
		}
		servicesCount := 0
		if client.Conf.Services.DNSPod {
			servicesCount++
		}
		if client.Conf.Services.AliDNS {
			servicesCount++
		}
		if client.Conf.Services.Cloudflare {
			servicesCount++
		}
		waitServicesDone := make(chan bool, servicesCount)
		if client.Conf.Services.DNSPod {
			go asyncServiceInterface(ipv4, ipv6, client.Dpc.Run, waitServicesDone)
		}
		if client.Conf.Services.AliDNS {
			go asyncServiceInterface(ipv4, ipv6, client.Adc.Run, waitServicesDone)
		}
		if client.Conf.Services.Cloudflare {
			go asyncServiceInterface(ipv4, ipv6, client.Cfc.Run, waitServicesDone)
		}
		for i := 0; i < servicesCount; i++ {
			<-waitServicesDone
		}
	}
	done <- true
}

func asyncServiceInterface(ipv4, ipv6 string, callback client.AsyncServerCallback, done chan bool) {
	msg, err := callback(client.Conf.Enable, ipv4, ipv6)
	for _, row := range err {
		log.Println(row)
	}
	for _, row := range msg {
		log.Println(row)
	}
	done <- true
}
