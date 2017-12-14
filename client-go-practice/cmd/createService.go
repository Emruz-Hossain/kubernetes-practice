// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	//appsv1beta2 "k8s.io/api/apps/v1beta2"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	//typedappsv1beta2 "k8s.io/client-go/kubernetes/typed/apps/v1beta2"
	//typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	//"k8s.io/apimachinery/pkg/util/intstr"
	//"fmt"
	"log"
	"k8s.io/apimachinery/pkg/util/intstr"
	"strconv"
)

// createServiceCmd represents the createService command
var createServiceCmd = &cobra.Command{
	Use:   "createService",
	Short: "crete service with type NodePort",
	Long: `this command create service to enable pods to comunicate each others or to out of node`,


	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("createService called")

		config,err := clientcmd.BuildConfigFromFlags("",getKubeConfigPath())
		if err!=nil{
			log.Fatal("Can't create config",err.Error())
		}

		clientset, err:= kubernetes.NewForConfig(config)

		if err!=nil{
			log.Fatal("Can't create clientset",err.Error())
		}

		serviceClient := clientset.CoreV1().Services(apiv1.NamespaceDefault)


		//-----------Create service template--------------------

		serviceTemplate := &apiv1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name: "wc-service",
			},
			Spec: apiv1.ServiceSpec{
				Selector: map[string]string{
					"app": "wc",
				},
				Type: apiv1.ServiceTypeNodePort,
				Ports: []apiv1.ServicePort{
					{
						Name:"service-port",
						Port:80,
						TargetPort: intstr.IntOrString{
							Type:intstr.String,
							StrVal:"container-port",
						},
					},
				},
			},
		}

		//--------------Create service----------------------
		fmt.Println("Service is creating.......................")

		service,err:= serviceClient.Create(serviceTemplate)

		if err!=nil{
			log.Fatal("Can't create service.",err.Error())
		}
		fmt.Println("Successfully service is created with name:",service.Name)


		//-------------get node ip--------------------
		nodeClient:=clientset.CoreV1().Nodes()
		node,err:=nodeClient.Get("minikube",metav1.GetOptions{})

		if err!=nil{
			log.Fatal("Can't get specified node.",err)
		}
		appURL:="http://"
		appURL+=node.Status.Addresses[0].Address
		appURL+=":"
		appURL+=strconv.Itoa(int(service.Spec.Ports[0].NodePort))
		fmt.Println("You can access apps at :",appURL)

	},
}

func init() {
	RootCmd.AddCommand(createServiceCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// createServiceCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// createServiceCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
