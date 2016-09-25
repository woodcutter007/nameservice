package main

import (
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"golang.org/x/net/context"

	"time"
	"ugo/uconf"
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

	client.Put(context.Background(), "[ns]--ns1--ip", "test1")
	client.Put(context.Background(), "[ns]--ns1--addr", "test1")


	var gresp *clientv3.GetResponse
	gresp, _ = client.Get(context.Background(), "[ns]--", clientv3.WithPrefix())

	fmt.Println(gresp)
	for _, ev := range gresp.Kvs {
		fmt.Printf("%s : %s\n", ev.Key, ev.Value)
	}

	gresp, _ = client.Get(context.Background(), "[proxy]--", clientv3.WithPrefix())
	fmt.Println(gresp)
	for _, ev := range gresp.Kvs {
		fmt.Printf("%s : %s\n", ev.Key, ev.Value)
	}

	client.Delete(context.Background(), uconf.ETCD_CENTER_PREFIX, clientv3.WithPrefix())
	client.Delete(context.Background(), uconf.ETCD_CLIENT_PREFIX, clientv3.WithPrefix())
	client.Delete(context.Background(), uconf.ETCD_CONF_PREFIX, clientv3.WithPrefix())

	client.Delete(context.Background(), "[proxy]", clientv3.WithPrefix())
	client.Delete(context.Background(), "[ns]", clientv3.WithPrefix())
	client.Delete(context.Background(), "proxy", clientv3.WithPrefix())
}
