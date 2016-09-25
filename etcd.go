package namespace

import (
	"github.com/coreos/etcd/clientv3"
	"golang.org/x/net/context"
	"fmt"
)

type EtcdManager struct {
	addr   AddrInfo
	Client *clientv3.Client
}

var Etcd_Manager *EtcdManager

func EtcdManagerInit(etcdAddr *AddrInfo) {

	if Etcd_Manager != nil {
		return
	}

	Etcd_Manager = new(EtcdManager)

	var cfg clientv3.Config
	cfg.DialTimeout = ETCD_DIAL_TIMEOUT
	cfg.Endpoints = etcdAddr.Addr
	client, err := clientv3.New(cfg)

	if err != nil {
		panic("etcd new error")
		return
	}

	Etcd_Manager.Client = client
	Etcd_Manager.addr = *etcdAddr

	return
}

func EtcdManagerUnInit() {
	if Etcd_Manager != nil {
		Etcd_Manager.Client.Close()
	}
}

type KeepAlveErrHandler func(param interface{})

func EtcdKeepAlive(key, value string, ttl int64) {

	fmt.Println("[EtcdKeepAlive]")
	client := Etcd_Manager.Client

	//create ttl node with 10s and keepalive forever
	if (ttl == 0) {
		ttl = 5
	}
	lease := clientv3.NewLease(client)
	resp, err := lease.Grant(context.TODO(), ttl)
	if err != nil {
		fmt.Println("[EtcdKeepAlive]", err)
		return
	}

	_, err = client.Put(context.TODO(), key, value, clientv3.WithLease(resp.ID))
	if err != nil {
		fmt.Println("[EtcdKeepAlive]", err)
		return
	}

	// the key 'foo' will be kept forever
	ch, kaerr := lease.KeepAlive(context.TODO(), resp.ID)
	if kaerr != nil {
		fmt.Println("[EtcdKeepAlive]", kaerr)
		return
	}
	go func() {
		for {
			resp := <-ch
			if resp == nil {
				lease.Close()
				break
			}else {
			}
		}
		EtcdKeepAlive(key, value, ttl)
	}()
}

func EtcdKeyCreate(key, value string) error {
	_, err := Etcd_Manager.Client.Put(context.Background(), key, value)
	if err != nil {
		fmt.Println("[EtcdKeyCreate]", err)
		return err
	}
	return nil
}

func EtcdKeyDelete(key, value string) error {
	_, err := Etcd_Manager.Client.Delete(context.Background(), key)
	if err != nil {
		fmt.Println("[EtcdKeyDelete]", err)
		return err
	}
	return nil
}

func EtcdKeyGet(key string) (string, error) {

	resp, err := Etcd_Manager.Client.Get(context.Background(), key)

	if err != nil {
		return "", err
	}
	for _, ev := range resp.Kvs {
		return string(ev.Value), nil
	}

	return "", nil
}

func EtcdKeyModRevisonGet(key string) (int64, error) {

	resp, err := Etcd_Manager.Client.Get(context.Background(), key)

	if err != nil {
		return 0, err
	}
	for _, ev := range resp.Kvs {
		return ev.ModRevision, nil
	}

	return 0, nil
}

func EtcdBatchKeyGet(key string) (*clientv3.GetResponse, error) {

	resp, err := Etcd_Manager.Client.Get(context.Background(), key, clientv3.WithPrefix())

	if err != nil {
		return nil, err
	}

	return resp, nil
}


type WatchHandler func(resp *clientv3.WatchResponse, param interface{})
type WatchErrHandler func(key string, param interface{})

func EtcdKeyBatchWatch(key string, handler WatchHandler, errHandler WatchErrHandler, param interface{}) {

	client := Etcd_Manager.Client

	rch := client.Watch(context.Background(), key, clientv3.WithPrefix())
	go func() {
		for wresp := range rch {
			fmt.Println("watch: ", wresp)
			handler(&wresp, param)
		}
		errHandler(key, param)
	}()

}

