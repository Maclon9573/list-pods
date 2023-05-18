package main

import (
	"context"
	"flag"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	"time"
)

var (
	kubeConfig   string
	workloadName string
	workloadKind string
	listInterval int64
	maxPodsCount int64
)

func init() {
	flag.StringVar(&kubeConfig, "kubeConfig", "", "kube config path")
	flag.Int64Var(&listInterval, "listInterval", 5, "list interval")
	flag.Int64Var(&maxPodsCount, "maxPodsCount", 0, "max pods count")
	flag.StringVar(&workloadName, "workloadName", "", "workload name")
	flag.StringVar(&workloadKind, "workloadKind", "", "workload kind")
}

func main() {
	flag.Parse()

	klog.Info("Starting pods lister...")
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		klog.Errorf("error getting k8s cluster config: %s", err.Error())
		return
	}

	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(err.Error())
	}

	for {
		klog.Infof("Listing pods...")
		pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}
		klog.Infof("There are %d pods in the cluster", len(pods.Items))

		podsMap := make(map[string]string)
		for _, p := range pods.Items {
			podsMap[p.Name] = p.Namespace
		}

		klog.Infof("Getting pods...")
		var index int64
		for k, v := range podsMap {
			_, err = clientset.CoreV1().Pods(v).Get(context.TODO(), k, metav1.GetOptions{})
			if errors.IsNotFound(err) {
				klog.Infof("Pod %s not found in namespace %s", k, v)
			} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
				klog.Errorf("Error getting pod %s in namespace %s, %v", k, v, statusError.ErrStatus.Message)
			} else if err != nil {
				panic(err.Error())
			} else {
				klog.Infof("Found pod %s in namespace %s", k, v)
			}
			index++
			if index >= maxPodsCount && maxPodsCount != 0 {
				break
			}
		}

		time.Sleep(time.Duration(listInterval) * time.Second)
	}
}
