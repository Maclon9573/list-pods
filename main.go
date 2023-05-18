package main

import (
	"context"
	"flag"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

var (
	kubeConfig   string
	workloadName string
	workloadKind string
	workloadGV   string
	listInterval int64
	maxPodsCount int64
)

func init() {
	flag.StringVar(&kubeConfig, "kubeConfig", "", "path to kube config")
	flag.Int64Var(&listInterval, "listInterval", 5, "interval for list pods action by seconds")
	flag.Int64Var(&maxPodsCount, "maxPodsCount", 0, "max pods count when list pods")
	flag.StringVar(&workloadName, "workloadName", "", "the workload name")
	flag.StringVar(&workloadKind, "workloadKind", "Deployment", "the workload kind")
	flag.StringVar(&workloadGV, "workloadGV", "apps/v1", "the workload group version")
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
	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		panic(err.Error())
	}
	obj := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": workloadGV,
			"kind":       workloadKind,
		},
	}
	gvk := obj.GroupVersionKind()
	gvr, _ := meta.UnsafeGuessKindToResource(gvk)
	resources, err := dynamicClient.Resource(gvr).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	var namespace string
	for _, d := range resources.Items {
		metadata, ok := d.Object["metadata"].(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := metadata["name"]
		if name == workloadName {
			namespace, _ = metadata["namespace"].(string)
			break
		}
	}
	if namespace == "" {
		klog.Errorf("Can not find workload %s", workloadName)
		return
	}

	for {
		klog.Infof("Listing pods in namespace %s for %s/%s...", namespace, workloadKind, workloadName)
		pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}
		klog.Infof("There are %d pods in namespace %s", len(pods.Items), namespace)

		klog.Infof("Getting pods...")
		var index int64
		for _, v := range pods.Items {
			_, err = clientset.CoreV1().Pods(v.Namespace).Get(context.TODO(), v.Name, metav1.GetOptions{})
			if errors.IsNotFound(err) {
				klog.Infof("Pod %s not found in namespace %s", v.Name, v.Namespace)
			} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
				klog.Errorf("Error getting pod %s in namespace %s, %v", v.Name, v.Namespace, statusError.ErrStatus.Message)
			} else if err != nil {
				panic(err.Error())
			} else {
				klog.Infof("Found pod %s in namespace %s", v.Name, v.Namespace)
			}
			index++
			if index >= maxPodsCount && maxPodsCount != 0 {
				break
			}
		}

		time.Sleep(time.Duration(listInterval) * time.Second)
	}
}
