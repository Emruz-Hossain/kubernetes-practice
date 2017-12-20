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


	appsv1beta2 "k8s.io/api/apps/v1beta2"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	//typedappsv1beta2 "k8s.io/client-go/kubernetes/typed/apps/v1beta2"
	//typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	//"k8s.io/apimachinery/pkg/util/intstr"
	//"fmt"
	"log"
)

// createDeploymentCmd represents the createDeployment command
var createDeploymentCmd = &cobra.Command{
	Use:   "createDeployment",
	Short: "create Deployment",
	Long: `Create a kubernetes Deployment object`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("createDeployment called")

		config,err := clientcmd.BuildConfigFromFlags("",getKubeConfigPath())

		if err!=nil{
			log.Fatal("Can't create config",err.Error())
		}

		clientset, err:= kubernetes.NewForConfig(config)

		if err!=nil{
			log.Fatal("Can't create clientset",err.Error())
		}

		deploymentClient := clientset.AppsV1beta2().Deployments(apiv1.NamespaceDefault)


		//-----------Create deployment template--------------------
		var numberOfReplica int32;
		numberOfReplica=2
		deploymentTemplate := &appsv1beta2.Deployment{
			ObjectMeta:metav1.ObjectMeta{
				Name: "wc-deployment",
			},

			Spec:appsv1beta2.DeploymentSpec{
				Replicas: &numberOfReplica,
				Selector: &metav1.LabelSelector{
					MatchLabels:map[string]string{
						"app":"wc",
					},
				},
				Template:apiv1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name: "wc-pod",
						Labels: map[string]string{
							"app":"wc",
						},
					},
					Spec:apiv1.PodSpec{
						Containers:[]apiv1.Container{
							{
								Name:"wc-container",
								Image:"emruzhossain/webcalculator:v5",
								Ports: []apiv1.ContainerPort{
									{
										Name: "container-port",
										ContainerPort: 9000,
										Protocol:apiv1.ProtocolTCP,
									},
								},
							},
						},
					},
				},
			},
		}

		//--------------Create deployment----------------------
		fmt.Println("Deployment is creating.......................")

		deployment,err:= deploymentClient.Create(deploymentTemplate)

		if err!=nil{
			log.Fatal("Can't create deployment",err.Error())
		}
		fmt.Println("Successfully deployment is created with name:",deployment.Name)
	},
}

func init() {
	RootCmd.AddCommand(createDeploymentCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// createDeploymentCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// createDeploymentCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
