package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-ping/ping"
	"os"
	"os/exec"
	"time"
)

var NormalCount uint64 = 0
var RestartCount uint64 = 0

func main() {

	go func() {
		for {
			pinger, err := ping.NewPinger(os.Args[1])
			if err != nil {
				panic(err)
			}

			pinger.Size = 24
			//ping间隔
			pinger.Interval = time.Second
			pinger.TTL = 64

			//每轮ping的次数
			pinger.Count = 10

			//一轮ping的时间
			pinger.Timeout = time.Second * 10

			//需要root权限（仅限Openwrt）
			pinger.SetPrivileged(true)

			pinger.OnRecv = func(pkt *ping.Packet) {
				fmt.Printf("%d bytes from %s: icmp_seq=%d time=%v ttl=%v\n",
					pkt.Nbytes, pkt.IPAddr, pkt.Seq, pkt.Rtt, pkt.Ttl)
			}
			pinger.OnDuplicateRecv = func(pkt *ping.Packet) {
				fmt.Printf("%d bytes from %s: icmp_seq=%d time=%v ttl=%v (DUP!)\n",
					pkt.Nbytes, pkt.IPAddr, pkt.Seq, pkt.Rtt, pkt.Ttl)
			}

			//一轮ping结束时回调
			pinger.OnFinish = func(statistics *ping.Statistics) {
				//丢包大于50%重启路由器网络
				fmt.Printf("PacketLoss: %v%% \n", statistics.PacketLoss)
				if statistics.PacketLoss > 50 {
					fmt.Println("restart network")

					//重启网络
					cmd := exec.Command("/etc/init.d/network", "restart")
					err := cmd.Run()
					if err != nil {
						fmt.Println(err.Error())
					}
					RestartCount++
				} else {
					NormalCount++
				}
			}

			fmt.Printf("PING %s (%s):\n", pinger.Addr(), pinger.IPAddr())
			err = pinger.Run()
			if err != nil {
				//网络挂了可能会报错，发生错误也重启网络
				fmt.Println(err)
				fmt.Println("ping error, restart network")
				//重启网络
				cmd := exec.Command("/etc/init.d/network", "restart")
				err := cmd.Run()
				if err != nil {
					fmt.Println(err.Error())
				}
				RestartCount++
			}

			time.Sleep(time.Second * 5)
		}
	}()

	//启动gin服务查看计数器
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.GET("/", getCount)
	err := r.Run(":11111")
	if err != nil {
		panic(err)
	}

}

func getCount(c *gin.Context) {
	c.String(200, "NormalCount : %d \nRestartCount : %d", NormalCount, RestartCount)
}
