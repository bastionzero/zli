package alivecheck

import (
	"context"

	"bastionzero.com/bctl/v1/Server/Websockets/controlWebsocket/controlWebsocketTypes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func AliveCheck(message controlWebsocketTypes.AliveCheckToClusterFromBastionSignalRMessage, wsClient *controlWebsocketTypes.ControlWebsocket) {
	// Let the Bastion know we are alive!
	aliveCheckToBastionFromClusterMessage := new(controlWebsocketTypes.AliveCheckToBastionFromClusterMessage)
	aliveCheckToBastionFromClusterMessage.Alive = true

	// Also let bastion know a list of valid cluster roles
	// Create our api object
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// Then get all cluster roles
	clusterRoles, err := clientset.RbacV1().ClusterRoles().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	aliveCheckToBastionFromClusterMessage.ClusterRoles = []string{}
	for _, clusterRole := range clusterRoles.Items {
		aliveCheckToBastionFromClusterMessage.ClusterRoles = append(aliveCheckToBastionFromClusterMessage.ClusterRoles, clusterRole.Name)
	}

	// Let Bastion know everything
	wsClient.SendAliveCheckToBastionFromClusterMessage(*aliveCheckToBastionFromClusterMessage)
}
