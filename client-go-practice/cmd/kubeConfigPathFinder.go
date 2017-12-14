package cmd

import (
	"k8s.io/client-go/util/homedir"
	"os"
	"fmt"
)

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