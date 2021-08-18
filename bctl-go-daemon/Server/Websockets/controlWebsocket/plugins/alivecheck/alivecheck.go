package alivecheck

import (
	"context"
	"regexp"

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
	clusterRoleBindings, err := clientset.RbacV1().ClusterRoleBindings().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	clusterUsers := make(map[string]bool)

	for _, clusterRoleBinding := range clusterRoleBindings.Items {
		// Now loop over the subjects if we can find any user subjects
		for _, subject := range clusterRoleBinding.Subjects {
			if subject.Kind == "User" {
				// We do not consider any system:... or eks:..., basically any system: looking roles as valid. This can be overridden from Bastion
				var systemRegexPatten = regexp.MustCompile(`[a-zA-Z0-9]*:[a-za-zA-Z0-9-]*`)
				if !systemRegexPatten.MatchString(subject.Name) {
					clusterUsers[subject.Name] = true
				}
			}
		}
	}

	// Then get all roles
	roleBindings, err := clientset.RbacV1().RoleBindings("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	for _, roleBindings := range roleBindings.Items {
		// Now loop over the subjects if we can find any user subjects
		for _, subject := range roleBindings.Subjects {
			if subject.Kind == "User" {
				// We do not consider any system:... or eks:..., basically any system: looking roles as valid. This can be overridden from Bastion
				var systemRegexPatten = regexp.MustCompile(`[a-zA-Z0-9]*:[a-za-zA-Z0-9-]*`)
				if !systemRegexPatten.MatchString(subject.Name) {
					clusterUsers[subject.Name] = true
				}
			}
		}
	}

	// Now build our response
	aliveCheckToBastionFromClusterMessage.ClusterUsers = []string{}
	for key, _ := range clusterUsers {
		aliveCheckToBastionFromClusterMessage.ClusterUsers = append(aliveCheckToBastionFromClusterMessage.ClusterUsers, key)
	}

	// Let Bastion know everything
	wsClient.SendAliveCheckToBastionFromClusterMessage(*aliveCheckToBastionFromClusterMessage)
}
