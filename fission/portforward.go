package main

import (
	"fmt"
	"os"
	"net"
	"strconv"
	"github.com/fission/fission/crd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/tools/remotecommand"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func findFreePort() (string, error) {

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return "", err
	}
	defer listener.Close()
	return strconv.Itoa(listener.Addr().(*net.TCPAddr).Port), nil
}

func runportForward(localPort string) error {

	//KUBECONFIG needs to be set to the correct path i.e ~/.kube/config
	config, PodClient, _, err := crd.GetKubernetesClient()
	if err != nil {
		msg := fmt.Sprint("%v\n", err)
		fatal(msg)
	}

	//get the podname for the controller
	PodList, err := PodClient.CoreV1().Pods("").List(meta_v1.ListOptions{LabelSelector:"svc=controller"})
	if err != nil {
		fatal("Error getting PodList with selector")
	}
	var podName string
	var podNameSpace string

	//there should only be one Pod in this list, the controller pod
	for _, item := range PodList.Items {

		podName = item.Name
		podNameSpace = item.Namespace
		break
	}

	//get the ControllerPort
	controllerService, err := PodClient.CoreV1().Services(podNameSpace).Get("controller", meta_v1.GetOptions{})
	if err != nil {
		fatal(fmt.Sprintf("Error getting controller service:%v", err))
	}
	var targetPort string
	for  _, servicePort := range controllerService.Spec.Ports {

		targetPort = servicePort.TargetPort.String()
	}

	StopChannel := make(chan struct{}, 1)
	ReadyChannel := make(chan struct{})

	//create request URL
	req := PodClient.CoreV1Client.RESTClient().Post().Resource("pods").Namespace(podNameSpace).Name(podName).SubResource("portforward")
	url := req.URL()

	//create ports slice
	portCombo := localPort + ":" + targetPort
	ports := []string{portCombo}

	//actually start the port-forwarding process here
	dialer, err := remotecommand.NewExecutor(config, "POST", url)
	if err != nil {
		msg := fmt.Sprintf("newexecutor errored out :%v", err)
		fatal(msg)
	}
	fw, err := portforward.New(dialer, ports , StopChannel, ReadyChannel, os.Stdout, os.Stderr)

	if err != nil {
		msg := fmt.Sprintf("portforward.new errored out :%v", err)
		fatal(msg)
	}
	return fw.ForwardPorts()
}