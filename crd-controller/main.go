package main

import (
	// "k8s.io/code-generator"
	 "github.com/kubernetes-practice/crd-controller/pkg/controller"

)
func main()  {
	controller.StartDeploymentController(1)

}