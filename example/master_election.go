package main

import (
	"ugo/namespace"
	"fmt"
	"ugo/common"
	"strings"
	"time"
)

func main() {

	addr := common.AddrInfo{}
	addr.Addr = strings.Split("192.168.153.38:2379,192.168.153.39:2379,192.168.153.40:2379", ",")

	ns := namespace.NewEtcdNameService(&addr,
		"[master-test]")
	ch, _ := ns.ElectionMaster()
	go func() {


		for {
			str := <- ch
			if str == "SlaveToMaster" {
				fmt.Println("slave to master")
			}
			if str == "MasterToSlave" {
				fmt.Println("master to slave")
			}
		}
	}()

	for {
		time.Sleep(100000)
	}
}
