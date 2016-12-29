package core

import (
	"fmt"
	"github.com/hunterhug/go_tool/util"
	"strings"
)

func getasins() []map[string]interface{} {

	num, e := RedisClient.Llen(MyConfig.Asinpool)
	if e != nil {
		panic(e)
	}
	if num > 0 {
		panic("Asinpool exist Today " + MyConfig.Asinpool)
	}
	result, err := DataDb.Select(strings.Replace(MyConfig.Asinsql, "{?}", Today, -1))
	if err != nil {
		panic(err)
	}

	return result
}

func sentasintoredis(results []map[string]interface{}) {
	sitetype := "https://www.amazon.com/dp/"
	switch SpiderType {
	case USA:
		sitetype = "https://www.amazon.com/dp/"
	case UK:
		sitetype = "https://www.amazon.co.uk/dp/"
	case JP:
		sitetype = "https://www.amazon.co.jp/dp/"
	case DE:
		sitetype = "https://www.amazon.de/dp/"
	default:
		panic("spider type error")

	}
	for _, result := range results {
		dudu := result["id"].(string)
		_, err := RedisClient.Lpush(MyConfig.Asinpool, sitetype+dudu)
		if err != nil {
			fmt.Printf("error:%v,%v\n", dudu, err)
		} else {
			fmt.Println("-")
		}
	}
}

func AsinPool() {
	OpenMysql()
	urls := getasins()
	sentasintoredis(urls)
}

func getips() map[string][]string {
	finalips := make(map[string][]string)
	ipuse := MyConfig.Ipuse
	ips := MyConfig.Ips
	for ipzonename, _ := range ipuse {
		finalips[ipzonename] = []string{}
		if ipmanyzone, ok := ips[ipzonename]; ok {
			for _, iponezone := range ipmanyzone {
				temp := strings.Split(iponezone, "-")
				templen := len(temp)
				if templen == 1 {
					finalips[ipzonename] = append(finalips[ipzonename], iponezone)
				} else if templen == 2 {
					//127.0.0.1-15
					ipend, err := util.SI(temp[1]) //15
					if err != nil {
						continue
					}
					insidetemp := strings.Split(temp[0], ".") //127 0 0 1
					if len(insidetemp) != 4 {
						continue
					} else {
						ip := strings.Join(insidetemp[0:3], ".")
						ipstart, err := util.SI(insidetemp[3])
						if err != nil {
							continue
						}
						for i := ipstart; i <= ipend; i++ {
							finalips[ipzonename] = append(finalips[ipzonename], ip+"."+util.IS(i))
						}
					}
				} else {
					continue
				}
			}
		}
	}

	filter := map[string][]string{}
	for k, ips := range finalips {
		if len(ips) == 0 {
			continue
		}
		ipport := strings.TrimSpace(ipuse[k].Port)
		ipsecret := strings.TrimSpace(ipuse[k].Secret)
		if ipsecret != "" {
			ipsecret = ipsecret + "@"
		}
		tempips := []string{}
		for _, ii := range ips {
			dudu := strings.Split(ii, ".")
			if len(dudu) != 4 {
				continue
			}
			IPdudu := true
			for _, n := range dudu {
				num, e := util.SI(n)
				if e != nil {
					IPdudu = false
					break
				}
				if num > 255 || num <= 0 {
					IPdudu = false
					break
				}
			}
			if IPdudu {

				tempips = append(tempips, ipsecret+ii+":"+ipport)
			}
		}
		if len(tempips) != 0 {
			filter[k] = tempips
		}
	}

	return filter
}

func shuffleip(ips map[string][]string) []string {
	returnip := []string{}
	if len(ips) == 0 {
		return returnip
	}
	smallsize := 100000000000
	dudu := make(map[string]int)
	for index, j := range ips {
		dudu[index] = len(j)
		if len(j) < smallsize {
			smallsize = len(j)
		}
	}
	fmt.Printf("%#v\n", dudu)
	for s := 0; s < smallsize; s++ {
		for _, j := range ips {
			returnip = append(returnip, j[s])
		}
	}
	for _, j := range ips {
		if len(j) > smallsize {
			for s := smallsize; s < len(j); s++ {
				returnip = append(returnip, j[s])
			}
		}
	}
	return returnip
}

func Sentiptoredis(ips []string) string {
	if len(ips) == 0 {
		return "IP Empty"
	}
	poolname := MyConfig.Proxypool
	if MyConfig.Proxyinit {
		err := RedisClient.Client.Del(poolname).Err()
		if err != nil {
			fmt.Println(err.Error())
			panic("redis del panic")
		}
	}
	returns := ""
	for _, ip := range ips {
		_, err := RedisClient.Lpush(poolname, ip)
		if err != nil {
			fmt.Printf("%s error:%v\n", ip, err)
			returns = returns + fmt.Sprintf("%s error:%v\n", ip, err)
		} else {
			fmt.Printf("%s success\n", ip)
			returns = returns + fmt.Sprintf("%s success\n", ip)
		}
	}
	return returns
}

var IPPOOL []string = []string{}

func IPPool() {
	OpenMysql()
	ips := getips()
	if len(ips) == 0 {
		//panic("ip zero")
	}
	shuips := shuffleip(ips)
	IPPOOL = shuips
	fmt.Printf("%#v\n", shuips)
	Sentiptoredis(shuips)
	go Clean()
	// montior
	go Montior()
	for {
		fmt.Println("ippool wait 1800 secord...")
		util.Sleep(1800)
		num, e := RedisClient.Llen(MyConfig.Proxypool)
		if e == nil && num == 0 {
			fmt.Printf("stop %d hours to full ippool\n", MyConfig.Proxyloophours)
			util.Sleep(MyConfig.Proxyloophours * 3600)
			Sentiptoredis(shuips)
		}
	}
}

func geturls() []string {
	num, e := RedisClient.Llen(MyConfig.Urlpool)
	if e != nil {
		panic(e)
	}
	if num > 0 {
		panic("Urlpool exist Today:" + MyConfig.Urlpool)
	}

	urls := []string{}
	fmt.Println(MyConfig.Urlsql)
	result, err := BasicDb.Select(MyConfig.Urlsql)
	if err != nil {
		panic(err)
	}
	for _, index := range result {
		urls = append(urls, index["id"].(string)+"|"+index["url"].(string)+"|"+index["name"].(string)+"|"+index["bigpname"].(string)+"|"+index["page"].(string))
	}
	return urls
}

func senturltoredis(urls []string) {
	for _, url := range urls {
		_, err := RedisClient.Lpush(MyConfig.Urlpool, url)
		if err != nil {
			fmt.Printf("error:%v,%v\n", url, err)
		} else {
			fmt.Println("-")
			//fmt.Println(url + " success!")
		}
	}
}

func UrlPool() {
	OpenMysql()
	urls := geturls()
	senturltoredis(urls)
}
