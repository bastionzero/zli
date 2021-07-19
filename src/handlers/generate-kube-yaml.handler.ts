import { Logger } from '../logger.service/logger';
import { ConfigService, KubeConfig } from '../config.service/config.service';

const pem = require('pem')
const path = require('path');
const fs = require('fs');


export async function generateKubeYamlHandler(
    logger: Logger
) {
    // TODO: We are missing a Bastion API call here

    // Now generate a kubeConfig
    let kubeYaml = `
---
# Service account to use to allow us to talk with our cluster
apiVersion: v1
kind: ServiceAccount
metadata:
    name: bctl-server-sa
    namespace: default
---
# Create a cluster role for our SA
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
    namespace: default
    name: bctl-server-sa-role
rules:
    - apiGroups: [""]
      resources: ["users", "groups", "serviceaccounts"]
      verbs: ["impersonate"]
---
# Now bind our new cluster role to our SA
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
    name: bctl-serverp-sa-rolebinding
subjects:
    - kind: ServiceAccount
      namespace: default
      name: bctl-server-sa
roleRef:
    kind: ClusterRole
    name: bctl-server-sa-role
    apiGroup: rbac.authorization.k8s.io
---
apiVersion: apps/v1
kind: Deployment
metadata:
    name: bctl-server
    labels:
        app: bctl-server
spec:
    replicas: 1
    selector:
        matchLabels:
            app: bctl-server
    template:
        metadata:
            labels:
                app: bctl-server
        spec:
            serviceAccountName: bctl-server-sa
            containers:
            - name: bctl-server
              image: 238681891460.dkr.ecr.us-east-1.amazonaws.com/bctlserver:2.0
              imagePullPolicy: Always
              ports:
              - containerPort: 6001
                name: bctl-port
              env:
              - name: ACTIVATION_CODE
                value: "1234"
              - name: ORG_ID
                value: "1234"
              - name: IDP_NAME
                value: "Google"
              - name: ENV
                value: "Default"
              resources:
                requests:
                  memory: 1G
                  cpu: "1"
                limits:
                  memory: 1G
                  cpu: "1"    
    `

    // Show it to the user
    logger.info(kubeYaml)
}