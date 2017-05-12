package main

import (
	"k8s.io/client-go/1.5/tools/clientcmd"
	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api/v1"
)

func main() {
	namespace,podName:="kube-system","kubernetes-dashboard-1723065636-jvw02"
	config, err := clientcmd.BuildConfigFromFlags("http://192.168.254.60:8080", "")
	if err!=nil{
		print(err)
	}
	client, err := kubernetes.NewForConfig(config)
	if err!=nil{
		print(err)
	}
	var pod *v1.Pod
	pod,err=client.Core().Pods(namespace).Get(podName)
	print(pod,err)
	print(pod.Name)
	print(pod.ObjectMeta.Annotations)
}
