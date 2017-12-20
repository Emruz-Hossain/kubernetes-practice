package main

import (
	"k8s.io/api/core/v1"
	//meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/client-go/util/homedir"



	"log"
	"fmt"
	"time"
	"os"
	//"github.com/mailru/easyjson/tests"
)

type Controller struct{
	indexer	 	cache.Indexer					//cache the objects we are monitoring
	queue		workqueue.RateLimitingInterface	//store key of objects if any event occurs to these objects. if multiple event occurs to a object then these events collapse and the latest update is kept.
	informer 	cache.Controller				//informer to inform if any change happens to the objects we are monitoring
	deletedIndexer cache.Indexer				//if a object is deleted then indexer does not hold the object rather it hold a tombstone  "DeletedFinalStateUnknown
												//so we are storing the deleted objects in deletedIndexer for further processing.
}

func newController(indexer cache.Indexer,queue workqueue.RateLimitingInterface,informer cache.Controller,deletedIndexer cache.Indexer)  *Controller{
	return &Controller{
		indexer: 	indexer,
		queue:			queue,
		informer:		informer,
		deletedIndexer:	deletedIndexer,
	}
}
func getKubeConfigPath () string {

	var kubeConfigPath string

	homeDir:=homedir.HomeDir()

	if _,err:=os.Stat(homeDir+"/.kube/config");err==nil{
		kubeConfigPath=homeDir+"/.kube/config"
	}else{
		fmt.Println("Enter kubernetes config directory: ")
		fmt.Scanf("%s",kubeConfigPath)
	}

	return kubeConfigPath
}

func main()  {
	//get path of kubeconfig
	configPath:=getKubeConfigPath();

	//create configuration
	config,err:=clientcmd.BuildConfigFromFlags("",configPath)
	if err!=nil{
		log.Fatal("Can't crete config. Error: %v",err)
	}

	//create clientset
	clientset,err:=kubernetes.NewForConfig(config)

	//create a pod List Watcher
	podListWatcher:=cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(),"pods",v1.NamespaceDefault,fields.Everything())

	//create working queue
	queue:=workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	//create a indexer for holding deleted objects
	deletedIndexer:=cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc,cache.Indexers{})
	//create informer

	indexer,informer := cache.NewIndexerInformer(podListWatcher, &v1.Pod{},0,cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key,err:=cache.MetaNamespaceKeyFunc(obj)
			if err==nil{
				queue.Add(key)
				deletedIndexer.Delete(obj) // object is in queue. hence it should not be in deletedIndexer
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			key,err:=cache.MetaNamespaceKeyFunc(newObj)
			if err==nil{
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key,err:=cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err==nil{
				deletedIndexer.Add(obj) //object is deleted. put it in deletedIndexer
				queue.Add(key)
			}
		},
	},cache.Indexers{})

	//now create controller
	controller := newController(indexer,queue,informer,deletedIndexer)

	//lets start the container
	stopCh:=make(chan struct{})
	defer close(stopCh)

	go controller.runPodWatcher(stopCh)

	//wait forever
	select{}
}


//runPodWathcer function will start the controller
// stopCh channel is used to send interrupt signal to stop the controller

func (c *Controller)runPodWatcher(stopCh chan struct{})  {
	//Don't panic if any error occurs in this function
	defer runtime.HandleCrash()

	//stop worker when we are done
	defer c.queue.ShutDown()

	log.Println("Starting pod controller........")

	//start the informer
	go c.informer.Run(stopCh)

	//wait to caches to be synced.
	if !cache.WaitForCacheSync(stopCh,c.informer.HasSynced){
		runtime.HandleError(fmt.Errorf("Wating for caches to sync..."))
		return
	}

	// continously run runWorker at 1 second  interval to process tasks in working queue until stopCh signal
	go wait.Until(c.runWorker,time.Second,stopCh)

	<-stopCh
}

func (c *Controller)runWorker()  {
	for c.processNextItem(){	//loop until all task in the queue is processed

	}
}

//Process the first item from the working queue
func (c* Controller) processNextItem() bool{
	// Get the key of the item in front of queue
	key, isEmpty:=c.queue.Get()

	if isEmpty{		//queue is empty hence time to break the loop of the caller function
		return  false
	}

	//we are processing the key hence the queue is done with the key
	defer c.queue.Done(key)


	err:=c.workWithTheEvent(key.(string))

	// if any error happen we need to handle it.
	c.handleError(err,key,5)

	return true
}

func (c *Controller)workWithTheEvent(key string) error {
	//get the object of this key from indexer
	obj,exist,err :=c.indexer.GetByKey(key)

	if err!=nil{
		fmt.Printf("Fetching object of key: %s from indexer failed with error: %v\n",key,err)
		return err
	}
	if !exist{	//object is not exist in indexer. maybe it is deleted.
		fmt.Printf("Pod with key: %s is no more exist.\n",key)
		obj,exist,err:=c.deletedIndexer.GetByKey(key)
		if err==nil && exist{
			fmt.Printf("..Pod %s is deleted.\n",obj.(*v1.Pod).GetName())

			c.deletedIndexer.Delete(key) //done with the object
		}
	}else{
		fmt.Println("Sync/Add/Update happed for pod ",obj.(*v1.Pod).GetName())
	}

	return  nil
}

func (c *Controller)handleError(err error, key interface{},maxNumberOfRetry int)  {
	if err==nil{	//no error. key is not required anymore. queue should forget the key's history
		c.queue.Forget(key)
		return
	}

	if c.queue.NumRequeues(key)<maxNumberOfRetry{
		fmt.Printf("Retraying to process the event with key %s\n",key)

		c.queue.AddRateLimited(key)
		return
	}

	//Retry limit is over. forget about the key.
	fmt.Printf("Can't process event with pod key:%s dropping it.\n",key)
	c.queue.Forget(key)
	runtime.HandleError(err)

}