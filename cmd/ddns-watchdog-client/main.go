package main

import (
	"ddns-watchdog/internal/client"
	"ddns-watchdog/internal/common"
	"errors"
	"flag"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"
)

var (
	installOption   = flag.Bool("I", false, "安装服务并退出")
	uninstallOption = flag.Bool("U", false, "卸载服务并退出")
	enforcement     = flag.Bool("f", false, "强制检查 DNS 解析记录")
	version         = flag.Bool("v", false, "查看当前版本并检查更新后退出")
	initOption      = flag.String("i", "", "有选择地初始化配置文件并退出，可以组合使用 (例 01)\n"+
		"0 -> "+client.ConfFileName+"\n"+
		"1 -> "+client.DNSPodConfFileName+"\n"+
		"2 -> "+client.AliDNSConfFileName+"\n"+
		"3 -> "+client.CloudflareConfFileName)
	confPath             = flag.String("c", "", "指定配置文件目录 (目录有空格请放在双引号中间)")
	printNetworkCardInfo = flag.Bool("n", false, "输出网卡信息并退出")
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
	if client.Conf.CheckCycleMinutes <= 0 {
		check()
	} else {
		cycle := time.NewTicker(time.Duration(client.Conf.CheckCycleMinutes) * time.Minute)
		for {
			check()
			<-cycle.C
		}
	}
}

func runFlag() (exit bool, err error) {
	flag.Parse()
	// 打印网卡信息
	if *printNetworkCardInfo {
		ncr, err2 := client.NetworkCardRespond()
		if err2 != nil {
			err = err2
			return
		}
		var arr []string
		for key := range ncr {
			arr = append(arr, key)
		}
		sort.Strings(arr)
		for _, key := range arr {
			fmt.Printf("%v\n\t%v\n", key, ncr[key])
		}
		exit = true
		return
	}

	// 加载自定义配置文件目录
	if *confPath != "" {
		client.ConfDirectoryName = common.FormatDirectoryPath(*confPath)
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
		err = client.Dpc.LoadConf()
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

func check() {
	// 获取 IP
	ipv4, ipv6, err := client.GetOwnIP(client.Conf.Enable, client.Conf.APIUrl, client.Conf.NetworkCard)
	if err != nil {
		log.Println(err)
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
		wg := sync.WaitGroup{}
		if client.Conf.Services.DNSPod {
			wg.Add(1)
			go asyncServiceInterface(ipv4, ipv6, client.Dpc.Run, &wg)
		}
		if client.Conf.Services.AliDNS {
			wg.Add(1)
			go asyncServiceInterface(ipv4, ipv6, client.Adc.Run, &wg)
		}
		if client.Conf.Services.Cloudflare {
			wg.Add(1)
			go asyncServiceInterface(ipv4, ipv6, client.Cfc.Run, &wg)
		}
		wg.Wait()
	}
}

func asyncServiceInterface(ipv4, ipv6 string, callback client.AsyncServiceCallback, wg *sync.WaitGroup) {
	defer wg.Done()
	msg, err := callback(client.Conf.Enable, ipv4, ipv6)
	for _, row := range err {
		log.Println(row)
	}
	for _, row := range msg {
		log.Println(row)
	}
}
