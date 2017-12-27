package controller

import (

	clientversioned "github.com/kubernetes-practice/crd-controller/pkg/client/clientset/versioned"
	crdv1alpha1 "github.com/kubernetes-practice/crd-controller/pkg/apis/crd.emruz.com/v1alpha1"
	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/client-go/util/homedir"
	rt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/clientcmd"

	"log"
	"fmt"
	"time"
	"os"
	"math/rand"
	"strconv"
)
var PreviousPodPhase map[string]string
var PodOwnerKey map[string]string
var synchronizationnChannel chan bool

type Controller struct{
	// for custom deployment
	clientset				 clientversioned.Clientset
	deploymentIndexer		 cache.Indexer
	deploymentInformer 		 cache.Controller
	deploymentWorkQueue		 workqueue.RateLimitingInterface
	deletedDeploymentIndexer cache.Indexer		// if deployment is deleted we may need deplyment object for further processing.

	// for pods under this custom deployment
	kubeclient 			kubernetes.Clientset
	podIndexer			cache.Indexer
	podInformer 		cache.Controller
	podWorkQueue		workqueue.RateLimitingInterface
	deletedPodIndexer 	cache.Indexer
	}

func NewController(clientset clientversioned.Clientset, kubeclientset kubernetes.Clientset)  *Controller{

	//---- ----------For Deployment----------

	// create ListWatcher for custom deployment
	deploymentListWatcher:=&cache.ListWatch{
		ListFunc: func(options meta_v1.ListOptions) (rt.Object, error) {
			return clientset.CrdV1alpha1().CustomDeployments(api_v1.NamespaceDefault).List(meta_v1.ListOptions{})
		},
		WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
			return clientset.CrdV1alpha1().CustomDeployments(api_v1.NamespaceDefault).Watch(options)
		},
	}

	//create workqueue for custom deployment
	deploymentWorkQueue:=workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	//create deleted indexer for custom deployment
	deletedDeploymentIndexer:=cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc,cache.Indexers{})

	//create indexer and informer for custom deployment
	deploymentIndexer,deploymentInformer:= cache.NewIndexerInformer(deploymentListWatcher, &crdv1alpha1.CustomDeployment{},0,cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key,err:=cache.MetaNamespaceKeyFunc(obj)
			if err==nil{
				deploymentWorkQueue.Add(key)
				deletedDeploymentIndexer.Delete(obj) //object is in workqueue hence it should not be here
			}

		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldDeployment:=oldObj.(*crdv1alpha1.CustomDeployment)
			newDeployment:=newObj.(*crdv1alpha1.CustomDeployment)

			if oldDeployment!=newDeployment{		//deployment has been updated
					key,err:=cache.MetaNamespaceKeyFunc(newObj)
					if err==nil{
						deploymentWorkQueue.Add(key)
					}
			}

		},
		DeleteFunc: func(obj interface{}) {
			key,err:=cache.MetaNamespaceKeyFunc(obj)
			if err==nil{
				deletedDeploymentIndexer.Add(obj)	// deployment has been deleted hence we are storing its object in case we need
				deploymentWorkQueue.Add(key)
			}
		},
	},cache.Indexers{})


	//----------------------------For Pods-------------------------------
	podListWatcher:=&cache.ListWatch{
		ListFunc: func(options meta_v1.ListOptions) (rt.Object, error) {
			return kubeclientset.CoreV1().Pods(api_v1.NamespaceDefault).List(meta_v1.ListOptions{})
		},
		WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
			return kubeclientset.CoreV1().Pods(api_v1.NamespaceDefault).Watch(options)
		},
	}

	//create workqueue for custom deployment
	podWorkQueue:=workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	//create deleted indexer for custom deployment
	deletedPodIndexer:=cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc,cache.Indexers{})

	//create indexer and informer for custom deployment
	podIndexer,podInformer:= cache.NewIndexerInformer(podListWatcher, &api_v1.Pod{},0,cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key,err:=cache.MetaNamespaceKeyFunc(obj)
			if err==nil{
				podWorkQueue.Add(key)
				deletedPodIndexer.Delete(obj)
			}

		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldPod:=oldObj.(*api_v1.Pod)
			newPod:=newObj.(*api_v1.Pod)

			if oldPod!=newPod{		//pod has been updated
				key,err:=cache.MetaNamespaceKeyFunc(newObj)
				if err==nil{
					podWorkQueue.Add(key)
				}
			}

		},
		DeleteFunc: func(obj interface{}) {
			key,err:=cache.MetaNamespaceKeyFunc(obj)
			if err==nil{
				deletedPodIndexer.Add(obj)
				podWorkQueue.Add(key)
			}
		},
	},cache.Indexers{})

	return &Controller{
		clientset: 				 clientset,
		deploymentIndexer:		 deploymentIndexer,
		deploymentInformer:		 deploymentInformer,
		deploymentWorkQueue:	 deploymentWorkQueue,
		deletedDeploymentIndexer:deletedDeploymentIndexer,

		kubeclient:	 		kubeclientset,
		podIndexer:	 		podIndexer,
		podInformer: 		podInformer,
		podWorkQueue:		podWorkQueue,
		deletedPodIndexer:	deletedPodIndexer,


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

func StartDeploymentController(thrediness int)  {

	PreviousPodPhase=make(map[string]string)
	PodOwnerKey = make(map[string]string)

	//get path of kubeconfig
	configPath:=getKubeConfigPath();

	//create configuration
	config,err:=clientcmd.BuildConfigFromFlags("",configPath)

	if err!=nil{
		log.Fatal("Can't crete config. Error: %v",err)
	}

	//create clientset
	kubeclientset,err:=kubernetes.NewForConfig(config)

	if err!=nil{
		log.Fatal(err)
	}

	clientset,err:=clientversioned.NewForConfig(config)
	if err!=nil{
		log.Fatal(err)
	}


	//now create controller
	controller := NewController(*clientset,*kubeclientset)

	//lets start the controller
	stopCh:=make(chan struct{})
	defer close(stopCh)

	go controller.RunController(thrediness,stopCh)
	//wait forever
	select{}
}


//runPodWathcer function will start the controller
// stopCh channel is used to send interrupt signal to stop the controller

func (c *Controller)RunController(thrediness int, stopCh chan struct{})  {
	//Don't panic if any error occurs in this function
	defer runtime.HandleCrash()

	//stop workers when we are done
	defer c.deploymentWorkQueue.ShutDown()
	defer c.podWorkQueue.ShutDown()

	log.Println("Starting informers........")
	//starting the informers
	go c.deploymentInformer.Run(stopCh)
	go c.podInformer.Run(stopCh)

	//// Wait for all involved caches to be synced, before processing items from the podWatchQueue is started
	if !cache.WaitForCacheSync(stopCh,c.deploymentInformer.HasSynced,c.podInformer.HasSynced){
		runtime.HandleError(fmt.Errorf("Wating for caches to sync..."))
		return
	}

	// continously run workers at 1 second  interval to process tasks in working queue until stopCh signal
	for i:=0;i<thrediness;i++{
		go wait.Until(c.runWorkerForDeployment,time.Second,stopCh)
		go wait.Until(c.runWorkerForPod,time.Second,stopCh)
	}

	<-stopCh		//stack here until a message appears on stopCh channel.
}

func (c *Controller)runWorkerForDeployment()  {
	for c.processNextItemFromDeploymentWorkQueue(){	//loop until all task in the queue is processed

	}
}

func (c *Controller)runWorkerForPod()  {
	for c.processNextItemFromPodWorkQueue(){	//loop until all task in the queue is processed

	}
}

//----------------------- For Deployment ------------------------------

//Process the first item from the working queue
func (c* Controller) processNextItemFromDeploymentWorkQueue() bool{
	// Get the key of the item in front of queue
	key, isEmpty:=c.deploymentWorkQueue.Get()

	if isEmpty{		//queue is empty hence time to break the loop of the caller function
		return  false
	}

	//Tell the deploymentWorkQueue that we are done with this key.This unblocks the key for other workers
	//This allow safe parallel processing because two item with same key will never be processed in parallel
	defer c.deploymentWorkQueue.Done(key)


	err:=c.performActionOnThisDeploymentKey(key.(string))

	// if any error occours we need to handle it.
	c.handleErrorForDeployment(err,key,5)

	return true
}

func (c *Controller)performActionOnThisDeploymentKey(key string) error {

	//get the object of this key from indexer
	obj,exist,err :=c.deploymentIndexer.GetByKey(key)

	if err!=nil{
		fmt.Printf("Fetching object of key: %s from indexer failed with error: %v\n",key,err)
		return err
	}

	if !exist{	//object is not exist in indexer. maybe it is deleted.
		fmt.Printf("Deployment %s is no more exist.\n",key)

		_,exist,err:=c.deletedDeploymentIndexer.GetByKey(key) //check if it is in the deleted Indexer to be confirmed it is deleted

		if err==nil && exist{
			fmt.Printf("Deployment %s has been deleted.\n",key)
			c.deletedDeploymentIndexer.Delete(key) //done with the object
		}
	}else{
		fmt.Println("Sync/Add/Update happed for deployment ",obj.(*crdv1alpha1.CustomDeployment).GetName())
		customdeploymentName:=obj.(*crdv1alpha1.CustomDeployment).GetName()
		customdeployment,err:=c.clientset.CrdV1alpha1().CustomDeployments(api_v1.NamespaceDefault).Get(customdeploymentName,meta_v1.GetOptions{})
		fmt.Printf("Required: %v | Available: %v | Processing: %v\n",customdeployment.Spec.Replicas,customdeployment.Status.AvailableReplicas,customdeployment.Status.CurrentlyProcessing)

		//If Current State is not same as Expected State preform necessary modification to meet the Goal.
		if customdeployment.Status.AvailableReplicas+customdeployment.Status.CurrentlyProcessing < customdeployment.Spec.Replicas{

			//after creating pod it will take time to get in Running state. in the mean time the pod is said to be processing state.
			err2:=c.UpdateDeploymentStatus(customdeployment,customdeployment.Status.AvailableReplicas,customdeployment.Status.CurrentlyProcessing+1)
			if err2!=nil{
				return err2
			}

			//create pod
			pod,err:= c.CreateNewPod(customdeployment.Spec.Template, customdeployment)

			//Failled to create to pod
			if err!=nil{
				fmt.Printf("Can't create pod. Error: %v\n",err.Error())
				err2=c.UpdateDeploymentStatus(customdeployment,customdeployment.Status.AvailableReplicas,customdeployment.Status.CurrentlyProcessing-1)
 				if err2!=nil{
 					return err2
				}
				return err
			}

			// Pod successfully created. Hence, update deployment status
			podName:=string(pod.GetName())
			PodOwnerKey[podName]=key
			fmt.Printf("PodName: %s OwnerKey: %s\n",podName,key)

			//if err2!=nil{
			//	fmt.Println("Pod created but failed to update DeploymentStatus.")
			//	return err2
			//}

		}else if customdeployment.Status.AvailableReplicas+customdeployment.Status.CurrentlyProcessing > customdeployment.Spec.Replicas{

			err=c.DeleteAPod(customdeployment)

			if err!=nil{
				fmt.Println("Failed to delete pod.")
				return err
			}

		}else{
			// Nothing to do...
		}
	}
	return  nil
}

func (c *Controller)handleErrorForDeployment(err error, key interface{},maxNumberOfRetry int)  {
	if err==nil{
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.deploymentWorkQueue.Forget(key)
		return
	}

	//Requeue the key for retry if it does not exceed the maximum limit of retry
	if c.deploymentWorkQueue.NumRequeues(key)<maxNumberOfRetry{

		fmt.Println("Error in processing event with key: %s\nError: %v",key,err.Error())
		fmt.Printf("Retraying to process the event for %s\n",key)

		// Requeuing the key for retry. This will increase NumRequeues for this key.
		c.deploymentWorkQueue.AddRateLimited(key)
		return
	}

	//Maximum number of requeue limit is over. Forget about the key.
	fmt.Printf("Can't process event with key:%s . The key is being dropped.\n",key)
	c.deploymentWorkQueue.Forget(key)
	runtime.HandleError(err)
}




//----------------------- For Pods ------------------------------

//Process the first item from the working queue
func (c* Controller) processNextItemFromPodWorkQueue() bool{
	// Get the key of the item in front of queue
	key, isEmpty:=c.podWorkQueue.Get()

	if isEmpty{		//queue is empty hence time to break the loop of the caller function
		return  false
	}

	//Tell the podWorkQueue that we are done with this key.This unblocks the key for other workers
	//This allow safe parallel processing because two item with same key will never be processed in parallel
	defer c.podWorkQueue.Done(key)


	err:=c.performActionOnThisPodKey(key.(string))

	// if any error occours we need to handle it.
	c.handleErrorForPod(err,key,5)

	return true
}

func (c *Controller)performActionOnThisPodKey(key string) error {

	//get the object of this key from indexer
	obj,exist,err :=c.podIndexer.GetByKey(key)

	if err!=nil{
		fmt.Printf("Fetching object of key: %s from indexer failed with error: %v\n",key,err)
		return err
	}

	if !exist{	//object is not exist in indexer. maybe it is deleted.

		fmt.Printf("Pod %s is no more exist.\n",key)
		deletedObj,exist,err:=c.deletedPodIndexer.GetByKey(key) //check if it is in the deleted Indexer to be confirmed it is deleted

		if err==nil && exist{
			fmt.Printf("pod %s has been deleted.\n",key)

			c.deletedPodIndexer.Delete(key) //done with the object

			podPhase:="Failed"
			podName:=deletedObj.(*api_v1.Pod).GetName()

			err:=c.checkAndUpdatePodStatus(string(podPhase),key,string(podName))

			delete(PreviousPodPhase,podName)
			delete(PodOwnerKey,key)
			return err

		}


	}else{
		fmt.Println("Sync/Add/Update happed for Pod: ",obj.(*api_v1.Pod).GetName())
		podPhase:=obj.(*api_v1.Pod).Status.Phase
		podName:=obj.(*api_v1.Pod).GetName()

		err:=c.checkAndUpdatePodStatus(string(podPhase),key,string(podName))

		if err==nil{
			PreviousPodPhase[string(podName)]=string(podPhase)
		}
		return err

	}
	return  nil
}

func (c *Controller)handleErrorForPod(err error, key interface{},maxNumberOfRetry int)  {
	if err==nil{
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.podWorkQueue.Forget(key)
		return
	}

	//Requeue the key for retry if it does not exceed the maximum limit of retry
	if c.podWorkQueue.NumRequeues(key)<maxNumberOfRetry{

		fmt.Println("Error in processing event with key: %s\nError: %v",key,err.Error())
		fmt.Printf("Retraying to process the event for %s\n",key)

		// Requeuing the key for retry. This will increase NumRequeues for this key.
		c.podWorkQueue.AddRateLimited(key)
		return
	}

	//Maximum number of requeue limit is exceeded. Forget about the key.
	//Maximum number of requeue limit is exceeded. Forget about the key.
	fmt.Printf("Can't process event with key:%s . The key is being dropped.\n",key)
	c.podWorkQueue.Forget(key)
	runtime.HandleError(err)
}


func (c *Controller)CreateNewPod(podTemplate crdv1alpha1.CustomPodTemplate, customdeployment *crdv1alpha1.CustomDeployment)  (*api_v1.Pod,error){

	podClient:=c.kubeclient.CoreV1().Pods(api_v1.NamespaceDefault)

	pod:=&api_v1.Pod{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:	customdeployment.GetName()+"-"+strconv.Itoa(rand.Int()),
			Labels: podTemplate.GetObjectMeta().GetLabels(),
		},
		Spec: podTemplate.Spec,
	}

	newPod,err:= podClient.Create(pod)

	if err==nil{
		fmt.Printf("New pod with name %v has been created.\n",newPod.GetName())
	}

	return newPod,err
}

func (c *Controller)DeleteAPod(customdeployment *crdv1alpha1.CustomDeployment)  error{

	return nil
}

func (c *Controller)checkAndUpdatePodStatus(podphase string,key string,podName string)  error{
	fmt.Printf("## PreviousPodPhase: %v CurrentPodPhase: %v\n",PreviousPodPhase[key],podphase)
	if PreviousPodPhase[podName]!=podphase{

		podownerkey:=PodOwnerKey[podName]
		ownerObj,exist,err :=c.deploymentIndexer.GetByKey(podownerkey)

		if err!=nil{
			fmt.Println("Error in getting deployment object from podownerkey")
			return err
		}

		if !exist{
			fmt.Println("Owner does not exist.")
			return err
		}
		deploymentName:=ownerObj.(*crdv1alpha1.CustomDeployment).GetName()
		customdeployment,err:=c.clientset.CrdV1alpha1().CustomDeployments(api_v1.NamespaceDefault).Get(deploymentName,meta_v1.GetOptions{})
		if err!=nil{
			return  err
		}
		available:=customdeployment.Status.AvailableReplicas
		processing:=customdeployment.Status.CurrentlyProcessing

		fmt.Printf("PreviousPodPhase: %v CurrentPodPhase: %v\n",PreviousPodPhase[key],podphase)

		if podphase=="Failed"{
			err:=c.UpdateDeploymentStatus(customdeployment,available-1,processing)
			return err

		}else if podphase=="Running"&&PreviousPodPhase[podName]=="Pending"{
				err:=c.UpdateDeploymentStatus(customdeployment,available+1,processing-1)
				return err
		}

	}

	return nil
}

func (c *Controller)UpdateDeploymentStatus(customdeployment *crdv1alpha1.CustomDeployment, AvailableReplicas int32, CurrentlyProcessing int32) error{

	//Don't modify cache. Work on it's copy
	customdeploymentCopy:= customdeployment.DeepCopy()

	customdeploymentCopy.Spec.Replicas = customdeployment.Spec.Replicas
	customdeploymentCopy.Status.AvailableReplicas = AvailableReplicas
	customdeploymentCopy.Status.CurrentlyProcessing = CurrentlyProcessing

	//Now update the cache
	_,err:=c.clientset.CrdV1alpha1().CustomDeployments(api_v1.NamespaceDefault).Update(customdeploymentCopy)


	return err
}
