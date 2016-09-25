
package namespace

import (
	"sync"
	"math/rand"
	"github.com/coreos/etcd/clientv3"
	"time"
	"golang.org/x/net/context"
	"fmt"
	"uframework/log"
)

type MasterHandler func(name string, master bool, param interface{})

type NameService interface {
	Publish(addr string)error
	ElectionMaster(callback MasterHandler)(chan string, error)

	Subscribe()error
	GetInstance()string
}
/*
type ZkNameService struct {
	sync.RWMutex
	zkCon         *zookeeper.ZkConn
	name          string
	serviceInst   []string
}

func NewZkNameService(zkAddr *common.AddrInfo, name string) *ZkNameService {

	ref := new(ZkNameService)
	zkCon, err := zookeeper.GetZkInstance(zkAddr.Addr[0])
	if (err != nil) {
		uflog.ERROR("[NewNameService] ", err)
		panic(err)
		return nil
	}
	ref.zkCon = zkCon

	ref.name = name

	return ref
}

func (ns *ZkNameService)Publish(addr string) error {

	path := "/" + ns.name + "/" + addr
	_, err := ns.zkCon.CreateNode(path, []byte(addr))
	if (err != nil) {
		fmt.Println("Publish error", err)
		uflog.ERROR("[Publish]", err)
		return err
	}
	fmt.Println("create success ", path)

	return nil
}

func (ns *ZkNameService)ElectionMaster(callback MasterHandler) (chan string, error) {

	return nil, nil
}

func (ns *ZkNameService)Subscribe() error {

	path := "/" + ns.name

	ns.Lock()
	defer ns.Unlock()

	list, err := ns.zkCon.ListChildren(path)
	if (err != nil) {
		fmt.Println(err)
		uflog.ERROR("[Subscribe]", err)
		return err
	}

	ns.serviceInst = list
	fmt.Println(list)

	_, _, ch, err1 := ns.zkCon.GetChildrenWatcher(path)
	if err1 != nil {

		fmt.Println(err1)
		uflog.ERROR("[Subscribe]", err1)
		return err
	}
	go func() {
		select {
		case _ = <-ch:
			ns.Subscribe()
		}
	}()

	return nil
}

func (ns *ZkNameService)GetInstance() string {

	ns.Lock()
	defer ns.Unlock()
	num := rand.Intn(len(ns.serviceInst))

	return ns.serviceInst[num]
}
*/
type EtcdNameService struct {
	sync.RWMutex
	name            string
	addr            string

	serviceInst     map[string]string

	master_key      string
	master          bool
	lease           clientv3.Lease
	ch              chan string
}

func NewEtcdNameService(addr *AddrInfo, name string) *EtcdNameService {

	ns := new(EtcdNameService)
	EtcdManagerInit(addr)

	ns.name = name
	ns.addr = ""
	ns.serviceInst = make(map[string]string)

	return ns
}

func (ns *EtcdNameService)Publish(addr string)error {

	ns.Lock()
	defer ns.Unlock()

	if (ns.addr != "") {
		return nil
	}

	key := "/" + ns.name + "/instance/" + addr
	EtcdKeepAlive(key, addr, 5)

	ns.addr = addr
	return nil
}

func keepAliveFail(ns *EtcdNameService) {

	fmt.Println("[keepAliveFail]")

	if (ns.master) {
		ns.ch <- "MasterToSlave"
		ns.master = false
		ns.lease.Close()
	}
}


func (ns *EtcdNameService)masterCompetition() {

	fmt.Println("[masterCompetition] ", ns.name)
	ns.lease = clientv3.NewLease(Etcd_Manager.Client)
	gresp, err := ns.lease.Grant(context.TODO(), 5)
	if err != nil {
		fmt.Println("[masterCompetition]", err)
		ns.lease.Close()
		return
	}

	resp, err := Etcd_Manager.Client.Txn(context.TODO()).
		If(clientv3.Compare(clientv3.CreateRevision(ns.master_key), ">", 0)).
		Then().
		Else(clientv3.OpPut(ns.master_key, ns.addr, clientv3.WithLease(gresp.ID))).
		Commit()

	if (err != nil) {
		fmt.Println(err)
	}
	if (!resp.Succeeded) {
		ch, kaerr := ns.lease.KeepAlive(context.TODO(), gresp.ID)
		if (kaerr != nil) {
			fmt.Println(kaerr)
		}

		fmt.Println("[masterCompetition] ", ns.name, " master")
		ns.ch <- "SlaveToMaster"

		ns.master = true
		go func() {
			for {
				resp := <-ch
				if resp == nil {
					break
				}
			}
			keepAliveFail(ns)
		}()
	}else {
		ns.lease.Close()
	}
}

type ElectionErr struct {

}
func (*ElectionErr) String()string {
	return "EnableElectionMaster must call before Publish"
}

func masterWatchHandle(resp *clientv3.WatchResponse, param interface{}) {

	fmt.Println("[masterWatchHandle]: ")
	ns := param.(*EtcdNameService)

	ns.Lock()
	defer ns.Unlock()
	for _, ev := range resp.Events {
		if ev.Type == 1 && !ns.master {
			ns.masterCompetition()
		}
	}

}

func masterWatchErrHandle(key string, param interface{}) {


}

func (ns *EtcdNameService) ElectionMaster() (<-chan string, error) {

	ns.Lock()
	defer ns.Unlock()

	ns.master_key = "/" + ns.name + "master"
	ns.ch = make(chan string, 1)

	EtcdKeyBatchWatch(ns.master_key, masterWatchHandle, masterWatchErrHandle, ns)

	ns.masterCompetition()

	return ns.ch, nil
}


func watchHandle(resp *clientv3.WatchResponse, param interface{}) {

	ns := param.(*EtcdNameService)
	ns.Lock()
	defer ns.Unlock()

	for _, ev := range resp.Events {

		key := string(ev.Kv.Key)
		if (ev.Type == 1) {
			delete(ns.serviceInst, key)
		}else {
			ns.serviceInst[key] = string(ev.Kv.Value)
		}
	}
	uflog.INFO(ns.serviceInst)
}

func watchErrHandle(key string, param interface{}) {

	ns := param.(*EtcdNameService)
	time.Sleep(2000)
	ns.Subscribe()
}

func (ns *EtcdNameService)Subscribe()error {

	ns.Lock()
	defer ns.Unlock()

	prefix := "/" + ns.name + "/instance"
	EtcdKeyBatchWatch(prefix, watchHandle, watchErrHandle, ns)

	resp, err := EtcdBatchKeyGet(prefix)
	if (err != nil) {
		uflog.ERROR(err)
		return err
	}
	for _, kv := range resp.Kvs {
		ns.serviceInst[string(kv.Key)] = string(kv.Value)
		fmt.Println(string(kv.Key))
	}
	fmt.Println(ns.serviceInst)

	return nil
}

func (ns *EtcdNameService)GetInstance() string {

	ns.Lock()
	defer ns.Unlock()

	if (len(ns.serviceInst) == 0) {
		return ""
	}
	num := rand.Intn(len(ns.serviceInst))

	var i int = 0
	for _, value := range ns.serviceInst {
		if (num == i) {
			return value
		}
		i++
	}

	return ""
}


