package main

import (
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"golang.org/x/net/context"

	"time"
)

func main() {
	var cfg clientv3.Config
	cfg.DialTimeout = 10 * time.Second
	cfg.Endpoints = append(cfg.Endpoints, "192.168.153.40:2379")
	fmt.Println(cfg)
	client, err := clientv3.New(cfg)

	if err != nil {
		panic("")
		return
	}
	//测试使用
	defer client.Close()

	fmt.Println(client)

	//create ttl node and keepalive forever
	resp, err := client.Grant(context.TODO(), 10*time.Second)
	if err != nil {
		fmt.Println(err)
	}
	_, err = client.Put(context.TODO(), "proxy", "1123", clientv3.WithLease(resp.ID))
	if err != nil {
		fmt.Println(err)
	}

	// the key 'foo' will be kept forever
	//ch, kaerr := client.KeepAlive(context.TODO(), resp.ID)
	//if kaerr != nil {
	//	fmt.Println(kaerr)
	//}

	//go func() {
	//	ka := <-ch
	//	fmt.Println("ttl:", ka.TTL)
	//}()

	var gresp *clientv3.GetResponse
	gresp, _ = client.Get(context.Background(), "foo1124")

	for _, ev := range gresp.Kvs {
		fmt.Printf("%s : %s\n", ev.Key, ev.Value)
	}

	rch := client.Watch(context.Background(), "/foo", clientv3.WithPrefix())
	var wresp clientv3.WatchResponse
	fmt.Println("watch ok")
	for wresp = range rch {
		for _, ev := range wresp.Events {
			fmt.Printf("%s %q : %q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
		}
	}
}
