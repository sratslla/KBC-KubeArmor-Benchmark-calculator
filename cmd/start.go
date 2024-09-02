package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type CaseEnum string

const (
	WithoutKubeArmor        CaseEnum = "WithoutKubeArmor"
	WithKubeArmorPolicy     CaseEnum = "WithKubeArmorPolicy"
	WithKubeArmorVisibility CaseEnum = "WithKubeArmorVisibility"
)

type ResourceUsage struct {
	Name   string
	CPU    float32
	Memory float32
}

// WK WOKP WOKV
type SingleCaseReport struct {
	Case                    CaseEnum // Case type: WithoutKubeArmor, WithKubeArmor, WithKubeArmorPolicy, WithKubeArmorVisibility
	MetricName              string   // Metric type: policy type, visibility type, none
	Users                   int32
	KubearmorResourceUsages []ResourceUsage
	Throughput              float32
	PercentageDrop          float32
	ResourceUsages          []ResourceUsage // List of resource usages
}

type FinalReport struct {
	Reports []SingleCaseReport
}

var finalReport FinalReport

var defaultThroughput float32

var users int32 = 600
var hpaCPUPercentage string = "50"

// var (
// 	config    *rest.Config
// 	clientset *kubernetes.Clientset
// )

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the benchmark process and apply all the relevant  resources.",
	Long:  `Start the benchmark process and apply all the relevant resources.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("start called")
		// Check if cluster is running then apply manifest files and start autoscalling

		// clientset, err := createClientset()
		// if err != nil {
		// 	fmt.Println("Error creating clientset: %v\n", err)
		// 	return
		// }
		// fmt.Println("1")
		config, err := rest.InClusterConfig()
		if err != nil {
			// Fallback to kubeconfig for local development
			kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err != nil {
				panic(err.Error())
			}
		}

		// Create a clientset
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}

		// Example: List all Pods in the "default" namespace
		pods, err := clientset.CoreV1().Pods("default").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			fmt.Printf("Error listing pods: %v\n", err)
			return
		}
		fmt.Println("2")

		fmt.Println("Pods in the default namespace:")
		for _, pod := range pods.Items {
			fmt.Printf("- %s\n", pod.Name)
		}
		fmt.Println("3")

		// fmt.Println(users, hpaCPUPercentage)
		// fmt.Printf("before config and clientset")

		// config, err := rest.InClusterConfig()
		// if err != nil {
		// 	panic(err.Error())
		// }
		// clientset, err = kubernetes.NewForConfig(config)
		// if err != nil {
		// 	panic(err.Error())
		// }

		fmt.Println("after config and clientset")

		// if isKubernetesClusterRunning() {
		// 	fmt.Println("Kubernetes cluster is running ")
		// } else {
		// 	fmt.Println("Kubernetes cluster is not running or accessible")
		// }
		// REPO_URL := "https://raw.githubusercontent.com/sratslla/KBC-KubeArmor-Benchmark-calculator/main/manifests"
		// manifestPaths := []string{
		// 	"kubernetes-manifests.yaml",
		// 	"loadgenerator_ui.yaml",
		// 	"kube-static-metrics.yaml",
		// 	"prometheusComponent.yaml",
		// }
		yamlContent := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: currencyservice
  labels:
    app: currencyservice
    env: benchmark
spec:
  selector:
    matchLabels:
      app: currencyservice
  template:
    metadata:
      labels:
        app: currencyservice
        env: benchmark
    spec:
      serviceAccountName: currencyservice
      terminationGracePeriodSeconds: 5
      securityContext:
        fsGroup: 1000
        runAsGroup: 1000
        runAsNonRoot: true
        runAsUser: 1000
      containers:
      - name: server
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
              - ALL
          privileged: false
          readOnlyRootFilesystem: true
        image: gcr.io/google-samples/microservices-demo/currencyservice:v0.10.0
        ports:
        - name: grpc
          containerPort: 7000
        env:
        - name: PORT
          value: "7000"
        - name: DISABLE_PROFILER
          value: "1"
        readinessProbe:
          grpc:
            port: 7000
        livenessProbe:
          grpc:
            port: 7000
        resources:
          requests:
            cpu: 100m
            memory: 64Mi
          limits:
            cpu: 200m
            memory: 128Mi
---
apiVersion: v1
kind: Service
metadata:
  name: currencyservice
  labels:
    app: currencyservice
spec:
  type: ClusterIP
  selector:
    app: currencyservice
  ports:
  - name: grpc
    port: 7000
    targetPort: 7000
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: currencyservice
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: loadgenerator
  labels:
    app: loadgenerator
    env: benchmark
spec:
  selector:
    matchLabels:
      app: loadgenerator
  replicas: 1
  template:
    metadata:
      labels:
        app: loadgenerator
        env: benchmark
      annotations:
        sidecar.istio.io/rewriteAppHTTPProbers: "true"
    spec:
      serviceAccountName: loadgenerator
      terminationGracePeriodSeconds: 5
      restartPolicy: Always
      securityContext:
        fsGroup: 1000
        runAsGroup: 1000
        runAsNonRoot: true
        runAsUser: 1000
      initContainers:
      - command:
        - /bin/sh
        - -exc
        - |
          MAX_RETRIES=12
          RETRY_INTERVAL=10
          for i in $(seq 1 $MAX_RETRIES); do
            echo "Attempt $i: Pinging frontend: ${FRONTEND_ADDR}..."
            STATUSCODE=$(wget --server-response http://${FRONTEND_ADDR} 2>&1 | awk '/^  HTTP/{print $2}')
            if [ $STATUSCODE -eq 200 ]; then
                echo "Frontend is reachable."
                exit 0
            fi
            echo "Error: Could not reach frontend - Status code: ${STATUSCODE}"
            sleep $RETRY_INTERVAL
          done
          echo "Failed to reach frontend after $MAX_RETRIES attempts."
          exit 1
        name: frontend-check
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
              - ALL
          privileged: false
          readOnlyRootFilesystem: true
        image: busybox:latest
        env:
        - name: FRONTEND_ADDR
          value: "frontend:80"
      containers:
      - name: main
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
              - ALL
          privileged: false
          readOnlyRootFilesystem: true
        image: sratslla/locust
        env:
        - name: FRONTEND_ADDR
          value: "frontend:80"
        - name: USERS
          value: "600"
        ports:
        - containerPort: 8089 # Locust UI port
      - name: locust-exporter
        image: containersol/locust_exporter
        ports:
        - containerPort: 9646 # Locust Exporter metrics port
      tolerations:
      - key: color
        operator: Equal
        value: blue
        effect: NoSchedule
      nodeSelector:
        nodetype: node1
---
apiVersion: v1
kind: Service
metadata:
  name: loadgenerator
  labels:
    app: loadgenerator
spec:
  selector:
    app: loadgenerator
  ports:
  - name: locust-ui
    port: 8089
    targetPort: 8089
  - name: locust-exporter
    port: 9646
    targetPort: 9646
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: loadgenerator
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: productcatalogservice
  labels:
    app: productcatalogservice
    env: benchmark
spec:
  selector:
    matchLabels:
      app: productcatalogservice
  template:
    metadata:
      labels:
        app: productcatalogservice
        env: benchmark
    spec:
      serviceAccountName: productcatalogservice
      terminationGracePeriodSeconds: 5
      securityContext:
        fsGroup: 1000
        runAsGroup: 1000
        runAsNonRoot: true
        runAsUser: 1000
      containers:
      - name: server
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
              - ALL
          privileged: false
          readOnlyRootFilesystem: true
        image: gcr.io/google-samples/microservices-demo/productcatalogservice:v0.10.0
        ports:
        - containerPort: 3550
        env:
        - name: PORT
          value: "3550"
        - name: DISABLE_PROFILER
          value: "1"
        readinessProbe:
          grpc:
            port: 3550
        livenessProbe:
          grpc:
            port: 3550
        resources:
          requests:
            cpu: 100m
            memory: 64Mi
          limits:
            cpu: 200m
            memory: 128Mi
---
apiVersion: v1
kind: Service
metadata:
  name: productcatalogservice
  labels:
    app: productcatalogservice
spec:
  type: ClusterIP
  selector:
    app: productcatalogservice
  ports:
  - name: grpc
    port: 3550
    targetPort: 3550
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: productcatalogservice
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: checkoutservice
  labels:
    app: checkoutservice
    env: benchmark
spec:
  selector:
    matchLabels:
      app: checkoutservice
  template:
    metadata:
      labels:
        app: checkoutservice
        env: benchmark
    spec:
      serviceAccountName: checkoutservice
      securityContext:
        fsGroup: 1000
        runAsGroup: 1000
        runAsNonRoot: true
        runAsUser: 1000
      containers:
        - name: server
          securityContext:
            allowPrivilegeEscalation: false
            capabilities:
              drop:
                - ALL
            privileged: false
            readOnlyRootFilesystem: true
          image: gcr.io/google-samples/microservices-demo/checkoutservice:v0.10.0
          ports:
          - containerPort: 5050
          readinessProbe:
            grpc:
              port: 5050
          livenessProbe:
            grpc:
              port: 5050
          env:
          - name: PORT
            value: "5050"
          - name: PRODUCT_CATALOG_SERVICE_ADDR
            value: "productcatalogservice:3550"
          - name: SHIPPING_SERVICE_ADDR
            value: "shippingservice:50051"
          - name: PAYMENT_SERVICE_ADDR
            value: "paymentservice:50051"
          - name: EMAIL_SERVICE_ADDR
            value: "emailservice:5000"
          - name: CURRENCY_SERVICE_ADDR
            value: "currencyservice:7000"
          - name: CART_SERVICE_ADDR
            value: "cartservice:7070"
          resources:
            requests:
              cpu: 100m
              memory: 64Mi
            limits:
              cpu: 200m
              memory: 128Mi
---
apiVersion: v1
kind: Service
metadata:
  name: checkoutservice
  labels:
    app: checkoutservice
spec:
  type: ClusterIP
  selector:
    app: checkoutservice
  ports:
  - name: grpc
    port: 5050
    targetPort: 5050
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: checkoutservice
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: shippingservice
  labels:
    app: shippingservice
    env: benchmark
spec:
  selector:
    matchLabels:
      app: shippingservice
  template:
    metadata:
      labels:
        app: shippingservice
        env: benchmark
    spec:
      serviceAccountName: shippingservice
      securityContext:
        fsGroup: 1000
        runAsGroup: 1000
        runAsNonRoot: true
        runAsUser: 1000
      containers:
      - name: server
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
              - ALL
          privileged: false
          readOnlyRootFilesystem: true
        image: gcr.io/google-samples/microservices-demo/shippingservice:v0.10.0
        ports:
        - containerPort: 50051
        env:
        - name: PORT
          value: "50051"
        - name: DISABLE_PROFILER
          value: "1"
        readinessProbe:
          periodSeconds: 5
          grpc:
            port: 50051
        livenessProbe:
          grpc:
            port: 50051
        resources:
          requests:
            cpu: 100m
            memory: 64Mi
          limits:
            cpu: 200m
            memory: 128Mi
---
apiVersion: v1
kind: Service
metadata:
  name: shippingservice
  labels:
    app: shippingservice
spec:
  type: ClusterIP
  selector:
    app: shippingservice
  ports:
  - name: grpc
    port: 50051
    targetPort: 50051
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: shippingservice
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cartservice
  labels:
    app: cartservice
    env: benchmark
spec:
  selector:
    matchLabels:
      app: cartservice
  template:
    metadata:
      labels:
        app: cartservice
        env: benchmark
    spec:
      serviceAccountName: cartservice
      terminationGracePeriodSeconds: 5
      securityContext:
        fsGroup: 1000
        runAsGroup: 1000
        runAsNonRoot: true
        runAsUser: 1000
      containers:
      - name: server
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
              - ALL
          privileged: false
          readOnlyRootFilesystem: true
        image: gcr.io/google-samples/microservices-demo/cartservice:v0.10.0
        ports:
        - containerPort: 7070
        env:
        - name: REDIS_ADDR
          value: "redis-cart:6379"
        resources:
          requests:
            cpu: 200m
            memory: 64Mi
          limits:
            cpu: 300m
            memory: 128Mi
        readinessProbe:
          initialDelaySeconds: 15
          grpc:
            port: 7070
        livenessProbe:
          initialDelaySeconds: 15
          periodSeconds: 10
          grpc:
            port: 7070
---
apiVersion: v1
kind: Service
metadata:
  name: cartservice
  labels:
    app: cartservice
spec:
  type: ClusterIP
  selector:
    app: cartservice
  ports:
  - name: grpc
    port: 7070
    targetPort: 7070
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: cartservice
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis-cart
  labels:
    app: redis-cart
    env: benchmark
spec:
  selector:
    matchLabels:
      app: redis-cart
  template:
    metadata:
      labels:
        app: redis-cart
        env: benchmark
    spec:
      securityContext:
        fsGroup: 1000
        runAsGroup: 1000
        runAsNonRoot: true
        runAsUser: 1000
      containers:
      - name: redis
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
              - ALL
          privileged: false
          readOnlyRootFilesystem: true
        image: redis:alpine
        ports:
        - containerPort: 6379
        readinessProbe:
          periodSeconds: 5
          tcpSocket:
            port: 6379
        livenessProbe:
          periodSeconds: 5
          tcpSocket:
            port: 6379
        volumeMounts:
        - mountPath: /data
          name: redis-data
        resources:
          limits:
            memory: 256Mi
            cpu: 125m
          requests:
            cpu: 70m
            memory: 200Mi
      volumes:
      - name: redis-data
        emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: redis-cart
  labels:
    app: redis-cart
spec:
  type: ClusterIP
  selector:
    app: redis-cart
  ports:
  - name: tcp-redis
    port: 6379
    targetPort: 6379
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: emailservice
  labels:
    app: emailservice
    env: benchmark
spec:
  selector:
    matchLabels:
      app: emailservice
  template:
    metadata:
      labels:
        app: emailservice
        env: benchmark
    spec:
      serviceAccountName: emailservice
      terminationGracePeriodSeconds: 5
      securityContext:
        fsGroup: 1000
        runAsGroup: 1000
        runAsNonRoot: true
        runAsUser: 1000
      containers:
      - name: server
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
              - ALL
          privileged: false
          readOnlyRootFilesystem: true
        image: gcr.io/google-samples/microservices-demo/emailservice:v0.10.0
        ports:
        - containerPort: 8080
        env:
        - name: PORT
          value: "8080"
        - name: DISABLE_PROFILER
          value: "1"
        readinessProbe:
          periodSeconds: 5
          grpc:
            port: 8080
        livenessProbe:
          periodSeconds: 5
          grpc:
            port: 8080
        resources:
          requests:
            cpu: 100m
            memory: 64Mi
          limits:
            cpu: 200m
            memory: 128Mi
---
apiVersion: v1
kind: Service
metadata:
  name: emailservice
  labels:
    app: emailservice
spec:
  type: ClusterIP
  selector:
    app: emailservice
  ports:
  - name: grpc
    port: 5000
    targetPort: 8080
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: emailservice
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: paymentservice
  labels:
    app: paymentservice
    env: benchmark
spec:
  selector:
    matchLabels:
      app: paymentservice
  template:
    metadata:
      labels:
        app: paymentservice
        env: benchmark
    spec:
      serviceAccountName: paymentservice
      terminationGracePeriodSeconds: 5
      securityContext:
        fsGroup: 1000
        runAsGroup: 1000
        runAsNonRoot: true
        runAsUser: 1000
      containers:
      - name: server
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
              - ALL
          privileged: false
          readOnlyRootFilesystem: true
        image: gcr.io/google-samples/microservices-demo/paymentservice:v0.10.0
        ports:
        - containerPort: 50051
        env:
        - name: PORT
          value: "50051"
        - name: DISABLE_PROFILER
          value: "1"
        readinessProbe:
          grpc:
            port: 50051
        livenessProbe:
          grpc:
            port: 50051
        resources:
          requests:
            cpu: 100m
            memory: 64Mi
          limits:
            cpu: 200m
            memory: 128Mi
---
apiVersion: v1
kind: Service
metadata:
  name: paymentservice
  labels:
    app: paymentservice
spec:
  type: ClusterIP
  selector:
    app: paymentservice
  ports:
  - name: grpc
    port: 50051
    targetPort: 50051
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: paymentservice
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend
  labels:
    app: frontend
    env: benchmark
spec:
  selector:
    matchLabels:
      app: frontend
  template:
    metadata:
      labels:
        app: frontend
        env: benchmark
      annotations:
        sidecar.istio.io/rewriteAppHTTPProbers: "true"
    spec:
      serviceAccountName: frontend
      securityContext:
        fsGroup: 1000
        runAsGroup: 1000
        runAsNonRoot: true
        runAsUser: 1000
      containers:
        - name: server
          securityContext:
            allowPrivilegeEscalation: false
            capabilities:
              drop:
                - ALL
            privileged: false
            readOnlyRootFilesystem: true
          image: gcr.io/google-samples/microservices-demo/frontend:v0.10.0
          ports:
          - containerPort: 8080
          readinessProbe:
            initialDelaySeconds: 10
            httpGet:
              path: "/_healthz"
              port: 8080
              httpHeaders:
              - name: "Cookie"
                value: "shop_session-id=x-readiness-probe"
          livenessProbe:
            initialDelaySeconds: 10
            httpGet:
              path: "/_healthz"
              port: 8080
              httpHeaders:
              - name: "Cookie"
                value: "shop_session-id=x-liveness-probe"
          env:
          - name: PORT
            value: "8080"
          - name: PRODUCT_CATALOG_SERVICE_ADDR
            value: "productcatalogservice:3550"
          - name: CURRENCY_SERVICE_ADDR
            value: "currencyservice:7000"
          - name: CART_SERVICE_ADDR
            value: "cartservice:7070"
          - name: RECOMMENDATION_SERVICE_ADDR
            value: "recommendationservice:8080"
          - name: SHIPPING_SERVICE_ADDR
            value: "shippingservice:50051"
          - name: CHECKOUT_SERVICE_ADDR
            value: "checkoutservice:5050"
          - name: AD_SERVICE_ADDR
            value: "adservice:9555"
          - name: SHOPPING_ASSISTANT_SERVICE_ADDR
            value: "shoppingassistantservice:80"
          # # ENV_PLATFORM: One of: local, gcp, aws, azure, onprem, alibaba
          # # When not set, defaults to "local" unless running in GKE, otherwies auto-sets to gcp
          # - name: ENV_PLATFORM
          #   value: "aws"
          - name: ENABLE_PROFILER
            value: "0"
          # - name: CYMBAL_BRANDING
          #   value: "true"
          # - name: ENABLE_ASSISTANT
          #   value: "true"
          # - name: FRONTEND_MESSAGE
          #   value: "Replace this with a message you want to display on all pages."
          # As part of an optional Google Cloud demo, you can run an optional microservice called the "packaging service".
          # - name: PACKAGING_SERVICE_URL
          #   value: "" # This value would look like "http://123.123.123"
          resources:
            requests:
              cpu: 100m
              memory: 64Mi
            limits:
              cpu: 200m
              memory: 128Mi
---
apiVersion: v1
kind: Service
metadata:
  name: frontend
  labels:
    app: frontend
spec:
  type: ClusterIP
  selector:
    app: frontend
  ports:
  - name: http
    port: 80
    targetPort: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: frontend-external
  labels:
    app: frontend
spec:
  type: LoadBalancer
  selector:
    app: frontend
  ports:
  - name: http
    port: 80
    targetPort: 8080
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: frontend
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: recommendationservice
  labels:
    app: recommendationservice
    env: benchmark
spec:
  selector:
    matchLabels:
      app: recommendationservice
  template:
    metadata:
      labels:
        app: recommendationservice
        env: benchmark
    spec:
      serviceAccountName: recommendationservice
      terminationGracePeriodSeconds: 5
      securityContext:
        fsGroup: 1000
        runAsGroup: 1000
        runAsNonRoot: true
        runAsUser: 1000
      containers:
      - name: server
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
              - ALL
          privileged: false
          readOnlyRootFilesystem: true
        image: gcr.io/google-samples/microservices-demo/recommendationservice:v0.10.0
        ports:
        - containerPort: 8080
        readinessProbe:
          periodSeconds: 5
          grpc:
            port: 8080
        livenessProbe:
          periodSeconds: 5
          grpc:
            port: 8080
        env:
        - name: PORT
          value: "8080"
        - name: PRODUCT_CATALOG_SERVICE_ADDR
          value: "productcatalogservice:3550"
        - name: DISABLE_PROFILER
          value: "1"
        resources:
          requests:
            cpu: 100m
            memory: 220Mi
          limits:
            cpu: 200m
            memory: 450Mi
---
apiVersion: v1
kind: Service
metadata:
  name: recommendationservice
  labels:
    app: recommendationservice
spec:
  type: ClusterIP
  selector:
    app: recommendationservice
  ports:
  - name: grpc
    port: 8080
    targetPort: 8080
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: recommendationservice
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: adservice
  labels:
    app: adservice
    env: benchmark
spec:
  selector:
    matchLabels:
      app: adservice
  template:
    metadata:
      labels:
        app: adservice
        env: benchmark
    spec:
      serviceAccountName: adservice
      terminationGracePeriodSeconds: 5
      securityContext:
        fsGroup: 1000
        runAsGroup: 1000
        runAsNonRoot: true
        runAsUser: 1000
      containers:
      - name: server
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
              - ALL
          privileged: false
          readOnlyRootFilesystem: true
        image: gcr.io/google-samples/microservices-demo/adservice:v0.10.0
        ports:
        - containerPort: 9555
        env:
        - name: PORT
          value: "9555"
        resources:
          requests:
            cpu: 200m
            memory: 180Mi
          limits:
            cpu: 300m
            memory: 300Mi
        readinessProbe:
          initialDelaySeconds: 20
          periodSeconds: 15
          grpc:
            port: 9555
        livenessProbe:
          initialDelaySeconds: 20
          periodSeconds: 15
          grpc:
            port: 9555
---
apiVersion: v1
kind: Service
metadata:
  name: adservice
  labels:
    app: adservice
spec:
  type: ClusterIP
  selector:
    app: adservice
  ports:
  - name: grpc
    port: 9555
    targetPort: 9555
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: adservice
# [END gke_release_kubernetes_manifests_microservices_demo]
    `
		fmt.Println("before apply resouce")
		err = applyResources(yamlContent, config, clientset)
		if err != nil {
			panic(err.Error())
		}
		fmt.Println("Resources applied successfully")
		// for _, manifestmanifestPath := range manifestPaths {
		// 	err := applyManifestFromGitHub(REPO_URL, manifestmanifestPath)
		// 	if err != nil {
		// 		fmt.Println("Error applying manifest:", err)
		// 		os.Exit(1)
		// 	}
		// }

		// TODO - optimize it using a Loop
		autoscaleDeployment("cartservice", hpaCPUPercentage, 2, 400)
		autoscaleDeployment("currencyservice", hpaCPUPercentage, 2, 400)
		autoscaleDeployment("emailservice", hpaCPUPercentage, 2, 400)
		autoscaleDeployment("checkoutservice", hpaCPUPercentage, 2, 400)
		autoscaleDeployment("frontend", hpaCPUPercentage, 5, 400)
		autoscaleDeployment("paymentservice", hpaCPUPercentage, 2, 400)
		autoscaleDeployment("productcatalogservice", hpaCPUPercentage, 2, 400)
		autoscaleDeployment("recommendationservice", hpaCPUPercentage, 2, 400)
		autoscaleDeployment("redis-cart", hpaCPUPercentage, 1, 400)
		autoscaleDeployment("shippingservice", hpaCPUPercentage, 2, 400)
		autoscaleDeployment("adservice", hpaCPUPercentage, 1, 400)

		// TODO - Automatically locust start using flag - DONE

		// Start the benchmark
		time.Sleep(1 * time.Minute)

		// getting externalIP of a node
		externalIP, err := getExternalIP()
		if err != nil {
			fmt.Println("Error getting external IP:", err)
			return
		}

		prometheusURL := fmt.Sprintf("http://%s:30000", externalIP)
		promClient, err := NewPrometheusClient(prometheusURL)
		if err != nil {
			fmt.Println("Error creating Prometheus client:", err)
			return
		}

		locustQuery := `locust_users{job="locust"}`

		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		// Check if locust users have reached required amount every 10 sec.
		for range ticker.C {
			locustResult, err := QueryPrometheus(promClient, locustQuery)
			if err != nil {
				fmt.Println("Error querying Prometheus for Locust metrics:", err)
				return
			}

			locustUsers := 0
			if locustResult.Type() == model.ValVector {
				vector := locustResult.(model.Vector)
				for _, sample := range vector {
					locustUsers = int(sample.Value)
					fmt.Printf("locustUsers %v", locustUsers)
				}
			}

			if locustUsers >= int(users) {
				fmt.Println("locust users reached 300. data will be fetched now to calculate avg benchmark.")
				break
			}

			fmt.Printf("\rWaiting for locust_users to reach 300\n")
		}

		// waiting 1 min for resources to stabalization and 10 mins for calculating avg
		time.Sleep(11 * time.Minute)

		calculateBenchMark(promClient, WithoutKubeArmor, "")

		deployments := []string{
			"cartservice",
			"currencyservice",
			"emailservice",
			"checkoutservice",
			"frontend",
			"paymentservice",
			"productcatalogservice",
			"recommendationservice",
			"redis-cart",
			"shippingservice",
			"adservice",
		}

		replicasMap := make(map[string]int)

		for _, deployment := range deployments {
			replicas, err := getCurrentReplicas(deployment)
			if err != nil {
				fmt.Printf("Error getting current replicas %s : %v\n", deployment, err)
				continue
			}
			replicasMap[deployment] = replicas
		}

		for deployment, replicas := range replicasMap {
			fmt.Printf("%s: %d replicas\n", deployment, replicas)
		}

		for _, deployment := range deployments {
			err := deleteHPA(deployment)
			if err != nil {
				fmt.Printf("Error deleting HPA for %s: %v\n", deployment, err)
				continue
			}
		}

		for deployment, replicas := range replicasMap {
			err := scaleDeployment(deployment, replicas)
			if err != nil {
				fmt.Printf("Error svaling deployment %s to %d replicas: %v\n", deployment, replicas, err)
				continue
			}
		}

		err = installKubearmor()
		if err != nil {
			fmt.Printf("Error installing Kubearmor: %v\n", err)
			return
		}

		err = runKarmorInstall()
		if err != nil {
			fmt.Printf("Error running karmor install: %v\n", err)
			return
		}

		err = configureKubearmorRelay()
		if err != nil {
			fmt.Printf("Error configuring kubearmor relay: %v\n", err)
			return
		}

		fmt.Println("exec3 called")
		time.Sleep(6 * time.Minute)
		calculateBenchMark(promClient, WithKubeArmorVisibility, "none")

		changeVisiblity("process")
		time.Sleep(6 * time.Minute)
		calculateBenchMark(promClient, WithKubeArmorVisibility, "process")

		changeVisiblity("process, file")
		time.Sleep(6 * time.Minute)
		calculateBenchMark(promClient, WithKubeArmorVisibility, "process & file")

		changeVisiblity("process, network")
		time.Sleep(6 * time.Minute)
		calculateBenchMark(promClient, WithKubeArmorVisibility, "process & network")

		changeVisiblity("process, network, file")
		time.Sleep(6 * time.Minute)
		calculateBenchMark(promClient, WithKubeArmorVisibility, "process, network & file")
		changeVisiblity("none")

		// Apply Policies and check

		// Process Policy.
		// err = applyManifestFromGitHub(REPO_URL, "policy-process.yaml")
		// if err != nil {
		// 	fmt.Println("Error applying manifest:", err)
		// }
		// time.Sleep(6 * time.Minute)
		// calculateBenchMark(promClient, WithKubeArmorPolicy, "process")

		// // Process and File Policy.
		// err = applyManifestFromGitHub(REPO_URL, "policy-file.yaml")
		// if err != nil {
		// 	fmt.Println("Error applying manifest:", err)
		// }
		// time.Sleep(6 * time.Minute)
		// calculateBenchMark(promClient, WithKubeArmorPolicy, "process & file")

		// // Process, File and Network.
		// err = applyManifestFromGitHub(REPO_URL, "policy-network.yaml")
		// if err != nil {
		// 	fmt.Println("Error applying manifest:", err)
		// }
		time.Sleep(6 * time.Minute)
		calculateBenchMark(promClient, WithKubeArmorPolicy, "process, file and network")

		// print final report
		printFinalReport(finalReport)

		// Write the data to markdown file.
		templateContent, err := ioutil.ReadFile("report_template.md")
		if err != nil {
			fmt.Println("Error reading template file:", err)
			return
		}

		// Create a new template and parse the Markdown template content
		tmpl, err := template.New("markdown").Parse(string(templateContent))
		if err != nil {
			fmt.Println("Error parsing template:", err)
			return
		}

		// Create a file to write the Markdown content
		file, err := os.Create("final_report.md")
		if err != nil {
			fmt.Println("Error creating file:", err)
			return
		}
		defer file.Close()

		// Execute the template and write the content to the file
		err = tmpl.Execute(file, finalReport)
		if err != nil {
			fmt.Println("Error executing template:", err)
			return
		}

		fmt.Println("Markdown file created successfully!")

	},
}

func init() {
	startCmd.Flags().Int32VarP(&users, "users", "u", 600, "Number of users to simulate")
	startCmd.Flags().StringVarP(&hpaCPUPercentage, "cpuPercent", "c", "50", "CPU Percentage for HPA")
	rootCmd.AddCommand(startCmd)
}

func applyResources(yamlData string, config *rest.Config, clientset *kubernetes.Clientset) error {
	// Create a dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}

	// Create a discovery client to find the GVRs for the resources
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return err
	}
	fmt.Println("a")
	// Create a RESTMapper to map resources to GVRs
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClient))
	context := context.TODO()

	// Split the YAML into individual resource definitions
	resources := strings.Split(yamlData, "---")
	fmt.Println("b")
	for _, resource := range resources {
		resource = strings.TrimSpace(resource)
		fmt.Println("c")
		fmt.Println(resource)
		if len(resource) == 0 {
			continue
		}

		fmt.Println("d")
		// Decode the YAML into a map[string]interface{}
		var rawObj map[string]interface{}
		err := yaml.Unmarshal([]byte(resource), &rawObj)
		if err != nil {
			return fmt.Errorf("error unmarshaling resource into rawObj: %v", err)
		}

		// Convert map[interface{}]interface{} to map[string]interface{}
		obj := &unstructured.Unstructured{Object: convertToMapStringInterface(rawObj)}
		fmt.Println("e")

		gvk := obj.GroupVersionKind()
		fmt.Printf("GVK: %+v\n", gvk)

		if gvk.Kind == "" || gvk.Version == "" {
			return fmt.Errorf("GVK is empty: %+v", gvk)
		}

		m, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return fmt.Errorf("error getting REST mapping for GVK %+v: %v", gvk, err)
		}
		fmt.Println("f")
		// Apply the resource using the dynamic client
		resourceInterface := dynamicClient.Resource(m.Resource).Namespace(obj.GetNamespace())
		_, err = resourceInterface.Create(context, obj, metav1.CreateOptions{})
		if err != nil {
			return err
		}

		fmt.Printf("Applied %s %s\n", gvk.Kind, obj.GetName())
	}
	fmt.Println("g")
	return nil
}

func convertToMapStringInterface(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		switch typedValue := v.(type) {
		case map[interface{}]interface{}:
			result[k] = convertToMapStringInterface(convertMapInterfaceToString(typedValue))
		case map[string]interface{}:
			result[k] = convertToMapStringInterface(typedValue)
		case []interface{}:
			result[k] = convertSliceInterfaceToString(typedValue)
		default:
			result[k] = v
		}
	}
	return result
}

func convertMapInterfaceToString(m map[interface{}]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		key := fmt.Sprintf("%v", k)
		switch typedValue := v.(type) {
		case map[interface{}]interface{}:
			result[key] = convertToMapStringInterface(convertMapInterfaceToString(typedValue))
		case map[string]interface{}:
			result[key] = convertToMapStringInterface(typedValue)
		case []interface{}:
			result[key] = convertSliceInterfaceToString(typedValue)
		default:
			result[key] = v
		}
	}
	return result
}

func convertSliceInterfaceToString(slice []interface{}) []interface{} {
	result := make([]interface{}, len(slice))
	for i, v := range slice {
		switch typedValue := v.(type) {
		case map[interface{}]interface{}:
			result[i] = convertToMapStringInterface(convertMapInterfaceToString(typedValue))
		case map[string]interface{}:
			result[i] = convertToMapStringInterface(typedValue)
		case []interface{}:
			result[i] = convertSliceInterfaceToString(typedValue)
		default:
			result[i] = v
		}
	}
	return result
}

// func getKubeConfig() (*rest.Config, error) {
// 	var config *rest.Config
// 	var err error

// 	// Try to load in-cluster config
// 	config, err = rest.InClusterConfig()
// 	if err != nil {
// 		// Fallback to kubeconfig file
// 		kubeconfig := filepath.Join(homeDir(), ".kube", "config")
// 		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to load kubeconfig: %v", err)
// 		}
// 	}

// 	return config, nil
// }

// func homeDir() string {
// 	if h := os.Getenv("HOME"); h != "" {
// 		return h
// 	}
// 	return os.Getenv("USERPROFILE") // windows
// }

// func createClientset() (*kubernetes.Clientset, error) {
// 	config, err := getKubeConfig()
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Create the Clientset
// 	clientset, err := kubernetes.NewForConfig(config)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create Kubernetes clientset: %v", err)
// 	}

//		return clientset, nil
//	}
// func applyYAML(clientset *kubernetes.Clientset, yamlContent string) error {
// 	// Split the YAML content by '---'
// 	yamls := strings.Split(yamlContent, "---")

// 	for _, yamlData := range yamls {
// 		if strings.TrimSpace(yamlData) == "" {
// 			continue
// 		}

// 		// Decode the YAML into an unstructured object
// 		obj := &runtime.Unknown{}
// 		_, _, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(yamlData), nil, obj)
// 		if err != nil {
// 			return fmt.Errorf("failed to decode YAML: %v", err)
// 		}

// 		// Get the REST mapping for the object
// 		gvk := obj.GetObjectKind().GroupVersionKind()
// 		restMapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
// 		if err != nil {
// 			return fmt.Errorf("failed to get REST mapping: %v", err)
// 		}

// 		// Apply the object using the dynamic client
// 		dynamicClient, err := dynamic.NewForConfig(config)
// 		if err != nil {
// 			return fmt.Errorf("failed to create dynamic client: %v", err)
// 		}

// 		resourceClient := dynamicClient.Resource(restMapping.Resource).Namespace(metaAccessor.Namespace(obj))

// 		_, err = resourceClient.Create(context.TODO(), obj, metav1.CreateOptions{})
// 		if errors.IsAlreadyExists(err) {
// 			fmt.Printf("Resource %s already exists, updating...\n", obj.GetName())
// 			retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
// 				_, updateErr := resourceClient.Update(context.TODO(), obj, metav1.UpdateOptions{})
// 				return updateErr
// 			})
// 			if retryErr != nil {
// 				return fmt.Errorf("failed to update resource: %v", retryErr)
// 			}
// 		} else if err != nil {
// 			return fmt.Errorf("failed to create resource: %v", err)
// 		}

// 		fmt.Printf("Resource %s applied successfully.\n", obj.GetName())
// 	}

// 	return nil
// }

// func isKubernetesClusterRunning() bool {
// 	// cmd := exec.Command("kubectl", "cluster-info")

// 	// var output bytes.Buffer
// 	// cmd.Stdout = &output
// 	// err := cmd.Run()
// 	// if err != nil {
// 	// 	return false
// 	// }
// 	// // fmt.Println(cmd, output.String())
// 	// return true

// 	_, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
// 	return err == nil
// }

// func applyManifestFromGitHub(repoURL, manifestPath string) error {
// 	// Fetch the YAML manifest file from GitHub
// 	fmt.Println("INside applyManifestFromGitHub")
// 	resp, err := http.Get(fmt.Sprintf("%s/%s", repoURL, manifestPath))
// 	if err != nil {
// 		return fmt.Errorf("error fetching manifest file: %v", err)
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		return fmt.Errorf("failed to fetch manifest file, status code: %d", resp.StatusCode)
// 	}

// 	// Read the content of the manifest file
// 	manifestContent, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		return fmt.Errorf("error reading manifest content: %v", err)
// 	}

// 	// Split the YAML content by "---"
// 	documents := bytes.Split(manifestContent, []byte("---"))

// 	// Create a dynamic client
// 	dynamicClient, err := dynamic.NewForConfig(config)
// 	if err != nil {
// 		return fmt.Errorf("error creating dynamic client: %v", err)
// 	}

// 	for _, doc := range documents {
// 		if len(bytes.TrimSpace(doc)) == 0 {
// 			continue
// 		}

// 		// Convert YAML to unstructured.Unstructured
// 		var unstructuredObj unstructured.Unstructured
// 		decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(doc), 4096)
// 		if err := decoder.Decode(&unstructuredObj); err != nil {
// 			return fmt.Errorf("error decoding YAML to unstructured object: %v", err)
// 		}

// 		// Get the GVR (GroupVersionResource) of the resource
// 		gvk := unstructuredObj.GroupVersionKind()
// 		gvr, _ := meta.UnsafeGuessKindToResource(gvk)

// 		// Get the resource interface
// 		resourceClient := dynamicClient.Resource(gvr).Namespace(unstructuredObj.GetNamespace())
// 		if unstructuredObj.GetNamespace() == "" {
// 			resourceClient = dynamicClient.Resource(gvr)
// 		}

// 		// Apply the resource
// 		_, err = resourceClient.Create(context.TODO(), &unstructuredObj, metav1.CreateOptions{})
// 		if err != nil && !kerrors.IsAlreadyExists(err) {
// 			return fmt.Errorf("error applying resource: %v", err)
// 		}
// 	}

// 	return nil
// }

// func applyManifestFromGitHub(repoURL, yamlFilePath string) error {
// 	// cmd := exec.Command("kubectl", "apply", "-f", fmt.Sprintf("%s/%s", repoURL, yamlFilePath))
// 	// var output bytes.Buffer
// 	// cmd.Stdout = &output
// 	// err := cmd.Run()
// 	// if err != nil {
// 	// 	fmt.Println("error applying manifest", output.String())
// 	// 	return fmt.Errorf("error applying manifest: %v\n%s", err, output.String())
// 	// }
// 	// fmt.Println("Manifest applied successfully.", output.String())
// 	// return nil

// 	// NEW
// 	url := fmt.Sprintf("%s/%s", repoURL, yamlFilePath)
// 	resp, err := http.Get(url)
// 	if err != nil {
// 		return fmt.Errorf("error fetching manifest: %v", err)
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		return fmt.Errorf("failed to download YAML file from GitHub: received status code %d", resp.StatusCode)
// 	}

// 	yamlData, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		return fmt.Errorf("failed to read YAML data: %v", err)
// 	}

// 	// Step 2: Decode the YAML into Kubernetes objects
// 	var objs []runtime.Object
// 	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(yamlData), 4096)
// 	for {
// 		var obj map[string]interface{}
// 		if err := decoder.Decode(&obj); err != nil {
// 			if err.Error() == "EOF" {
// 				break
// 			}
// 			return fmt.Errorf("failed to decode YAML: %w", err)
// 		}

// 		jsonData, err := json.Marshal(obj)
// 		if err != nil {
// 			return fmt.Errorf("failed to marshal map to JSON: %w", err)
// 		}

// 		unstructuredObj := &unstructured.Unstructured{}
// 		if err := unstructuredObj.UnmarshalJSON(jsonData); err != nil {
// 			return fmt.Errorf("failed to unmarshal JSON: %w", err)
// 		}
// 		objs = append(objs, unstructuredObj)
// 	}

// 	// Step 3: Apply the manifests to the Kubernetes cluster
// 	for _, obj := range objs {
// 		gvk := obj.GetObjectKind().GroupVersionKind()
// 		if err := applyObject(clientset, gvk, obj); err != nil {
// 			return fmt.Errorf("failed to apply object %s: %w", gvk.String(), err)
// 		}
// 	}

// 	return nil
// }

// func applyObject(clientset *kubernetes.Clientset, gvk schema.GroupVersionKind, obj runtime.Object) error {
// 	u, ok := obj.(*unstructured.Unstructured)
// 	if !ok {
// 		return fmt.Errorf("object is not unstructured")
// 	}

// 	m, err := meta.Accessor(u)
// 	if err != nil {
// 		return fmt.Errorf("failed to access metadata: %w", err)
// 	}

// 	// Get the appropriate Kubernetes client for the resource
// 	switch gvk.Kind {
// 	case "Deployment":
// 		return applyDeployment(clientset, m.GetNamespace(), u)
// 	case "Service":
// 		return applyService(clientset, m.GetNamespace(), u)
// 	case "ServiceAccount":
// 		return applyServiceAccount(clientset, m.GetNamespace(), u)
// 	default:
// 		return fmt.Errorf("unsupported resource kind: %s", gvk.Kind)
// 	}
// }

// func applyDeployment(clientset *kubernetes.Clientset, namespace string, obj *unstructured.Unstructured) error {
// 	deploymentClient := clientset.AppsV1().Deployments(namespace)
// 	deployment := &appsv1.Deployment{}
// 	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, deployment); err != nil {
// 		return fmt.Errorf("failed to convert unstructured object to Deployment: %w", err)
// 	}
// 	_, err := deploymentClient.Update(deployment, metav1.UpdateOptions{})
// 	if err != nil {
// 		_, err = deploymentClient.Create(deployment, metav1.CreateOptions{})
// 	}
// 	return err
// }

// func applyService(clientset *kubernetes.Clientset, namespace string, obj *unstructured.Unstructured) error {
// 	serviceClient := clientset.CoreV1().Services(namespace)
// 	service := &v1.Service{}
// 	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, service); err != nil {
// 		return fmt.Errorf("failed to convert unstructured object to Service: %w", err)
// 	}
// 	_, err := serviceClient.Update(service, metav1.UpdateOptions{})
// 	if err != nil {
// 		_, err = serviceClient.Create(service, metav1.CreateOptions{})
// 	}
// 	return err
// }

// func applyServiceAccount(clientset *kubernetes.Clientset, namespace string, obj *unstructured.Unstructured) error {
// 	serviceAccountClient := clientset.CoreV1().ServiceAccounts(namespace)
// 	serviceAccount := &v1.ServiceAccount{}
// 	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, serviceAccount); err != nil {
// 		return fmt.Errorf("failed to convert unstructured object to ServiceAccount: %w", err)
// 	}
// 	_, err := serviceAccountClient.Update(serviceAccount, metav1.UpdateOptions{})
// 	if err != nil {
// 		_, err = serviceAccountClient.Create(serviceAccount, metav1.CreateOptions{})
// 	}
// 	return err
// }

func autoscaleDeployment(deploymentName string, cpuPercent string, minReplicas, maxReplicas int) {
	cmd := exec.Command("kubectl", "autoscale", "deployment", deploymentName,
		fmt.Sprintf("--cpu-percent=%s", cpuPercent),
		fmt.Sprintf("--min=%d", minReplicas),
		fmt.Sprintf("--max=%d", maxReplicas))

	var output bytes.Buffer
	cmd.Stdout = &output
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error autoscaling deployment %s: %v\n%s", deploymentName, err, output.String())
	} else {
		fmt.Printf("Deployment %s autoscaled successfully.\n", deploymentName)
	}
}

// Make a Prometheus Client
func NewPrometheusClient(prometheusURL string) (v1.API, error) {
	client, err := api.NewClient(api.Config{
		Address: prometheusURL,
	})
	if err != nil {
		return nil, err
	}
	return v1.NewAPI(client), nil
}

// Query using PromQL
func QueryPrometheus(api v1.API, query string) (model.Value, error) {
	result, warnings, err := api.Query(context.Background(), query, time.Now())
	if err != nil {
		return nil, err
	}
	if len(warnings) > 0 {
		fmt.Println("Warnings received during query execution:", warnings)
	}
	return result, nil
}

// Get the externalIP of a node as prometheus is running on "externalIP:30000"
func getExternalIP() (string, error) {
	cmd := exec.Command("kubectl", "get", "nodes", "-o", "json")

	var output bytes.Buffer
	cmd.Stdout = &output
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error getting nodes: %v", err)
	}

	var nodes struct {
		Items []struct {
			Status struct {
				Addresses []struct {
					Type    string `json:"type"`
					Address string `json:"address"`
				} `json:"addresses"`
			} `json:"status"`
		} `json:"items"`
	}

	err = json.Unmarshal(output.Bytes(), &nodes)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling nodes JSON: %v", err)
	}

	for _, node := range nodes.Items {
		for _, address := range node.Status.Addresses {
			if address.Type == "ExternalIP" {
				return address.Address, nil
			}
		}
	}

	return "", fmt.Errorf("no external IP found")
}

func calculateBenchMark(promClient v1.API, scenario CaseEnum, Metric string) {
	// TODO - Use fmt.Sprintf and add the service name from a variable.
	queries := map[string]map[string]string{
		"FRONTEND": {
			"cpu":    `sum(rate(container_cpu_usage_seconds_total{pod=~"frontend-.*", container="", namespace="default"}[5m])) * 1000`,
			"memory": `sum(container_memory_usage_bytes{pod=~"frontend-.*", namespace="default"}) / 1024 / 1024`,
		},
		"AD": {
			"cpu":    `sum(rate(container_cpu_usage_seconds_total{pod=~"adservice-.*", container="", namespace="default"}[5m])) * 1000`,
			"memory": `sum(container_memory_usage_bytes{pod=~"adservice-.*", namespace="default"}) / 1024 / 1024`,
		},
		"CART": {
			"cpu":    `sum(rate(container_cpu_usage_seconds_total{pod=~"cartservice-.*", container="", namespace="default"}[5m])) * 1000`,
			"memory": `sum(container_memory_usage_bytes{pod=~"cartservice-.*", namespace="default"}) / 1024 / 1024`,
		},
		"CHECKOUT": {
			"cpu":    `sum(rate(container_cpu_usage_seconds_total{pod=~"checkoutservice-.*", container="", namespace="default"}[5m])) * 1000`,
			"memory": `sum(container_memory_usage_bytes{pod=~"checkoutservice-.*", namespace="default"}) / 1024 / 1024`,
		},
		"CURRENCY": {
			"cpu":    `sum(rate(container_cpu_usage_seconds_total{pod=~"currencyservice-.*", container="", namespace="default"}[5m])) * 1000`,
			"memory": `sum(container_memory_usage_bytes{pod=~"currencyservice-.*", namespace="default"}) / 1024 / 1024`,
		},
		"EMAIL": {
			"cpu":    `sum(rate(container_cpu_usage_seconds_total{pod=~"emailservice-.*", container="", namespace="default"}[5m])) * 1000`,
			"memory": `sum(container_memory_usage_bytes{pod=~"emailservice-.*", namespace="default"}) / 1024 / 1024`,
		},
		"LOAD": {
			"cpu":    `sum(rate(container_cpu_usage_seconds_total{pod=~"loadgenerator-.*", container="", namespace="default"}[5m])) * 1000`,
			"memory": `sum(container_memory_usage_bytes{pod=~"loadgenerator-.*", namespace="default"}) / 1024 / 1024`,
		},
		"PAYMENT": {
			"cpu":    `sum(rate(container_cpu_usage_seconds_total{pod=~"paymentservice-.*", container="", namespace="default"}[5m])) * 1000`,
			"memory": `sum(container_memory_usage_bytes{pod=~"paymentservice-.*", namespace="default"}) / 1024 / 1024`,
		},
		"PRODUCT": {
			"cpu":    `sum(rate(container_cpu_usage_seconds_total{pod=~"productcatalogservice-.*", container="", namespace="default"}[5m])) * 1000`,
			"memory": `sum(container_memory_usage_bytes{pod=~"productcatalogservice-.*", namespace="default"}) / 1024 / 1024`,
		},
		"RECOMMENDATION": {
			"cpu":    `sum(rate(container_cpu_usage_seconds_total{pod=~"recommendationservice-.*", container="", namespace="default"}[5m])) * 1000`,
			"memory": `sum(container_memory_usage_bytes{pod=~"recommendationservice-.*", namespace="default"}) / 1024 / 1024`,
		},
		"REDIS": {
			"cpu":    `sum(rate(container_cpu_usage_seconds_total{pod=~"redis-.*", container="", namespace="default"}[5m])) * 1000`,
			"memory": `sum(container_memory_usage_bytes{pod=~"redis-.*", namespace="default"}) / 1024 / 1024`,
		},
		"SHIPPING": {
			"cpu":    `sum(rate(container_cpu_usage_seconds_total{pod=~"shippingservice-.*", container="", namespace="default"}[5m])) * 1000`,
			"memory": `sum(container_memory_usage_bytes{pod=~"shippingservice-.*", namespace="default"}) / 1024 / 1024`,
		},
	}

	var resourceUsages []ResourceUsage

	for serviceName, queryMap := range queries {
		cpuQuery := queryMap["cpu"]
		memoryQuery := queryMap["memory"]

		cpuTempResult, err := QueryPrometheus(promClient, cpuQuery)
		if err != nil {
			fmt.Printf("Error querying Prometheus for CPU metrics (%s): %v\n", serviceName, err)
			continue
		}
		memoryTempResult, err := QueryPrometheus(promClient, memoryQuery)
		if err != nil {
			fmt.Printf("Error querying Prometheus for Memory metrics (%s): %v\n", serviceName, err)
			continue
		}
		cpuResult, _ := parseUsage(cpuTempResult)
		memoryResult, _ := parseUsage(memoryTempResult)

		resourceUsage := ResourceUsage{
			Name:   serviceName,
			CPU:    cpuResult,
			Memory: memoryResult,
		}
		resourceUsages = append(resourceUsages, resourceUsage)
	}

	locustThroughputQuery := `avg_over_time(locust_requests_current_rps{job="locust", name="Aggregated"}[5m])`
	locustThroughput, err := QueryPrometheus(promClient, locustThroughputQuery)
	if err != nil {
		fmt.Println("Error querying Prometheus for Locust metrics:", err)
		return
	}
	locustThroughputResult, _ := parseUsage(locustThroughput)

	kubearmorQueries := map[string]map[string]string{
		"KUBEARMOR": {
			"cpu":    `sum(rate(container_cpu_usage_seconds_total{pod=~"kubearmor-bpf-containerd-.*", container="", namespace="kubearmor"}[5m])) * 1000`,
			"memory": `sum(container_memory_usage_bytes{pod=~"kubearmor-bpf-containerd-.*", namespace="kubearmor"}) / 1024 / 1024`,
		},
		"KUBEARMOR-RELAY": {
			"cpu":    `sum(rate(container_cpu_usage_seconds_total{pod=~"kubearmor-relay-.*", container="", namespace="kubearmor"}[5m])) * 1000`,
			"memory": `sum(container_memory_usage_bytes{pod=~"kubearmor-relay-.*", namespace="kubearmor"}) / 1024 / 1024`,
		},
	}

	var KubearmorResourceUsages []ResourceUsage
	for serviceName, queryMap := range kubearmorQueries {
		cpuQuery := queryMap["cpu"]
		memoryQuery := queryMap["memory"]

		cpuTempResult, err := QueryPrometheus(promClient, cpuQuery)
		if err != nil {
			fmt.Printf("Error querying Prometheus for CPU metrics (%s): %v\n", serviceName, err)
			continue
		}
		memoryTempResult, err := QueryPrometheus(promClient, memoryQuery)
		if err != nil {
			fmt.Printf("Error querying Prometheus for Memory metrics (%s): %v\n", serviceName, err)
			continue
		}
		cpuResult, _ := parseUsage(cpuTempResult)
		memoryResult, _ := parseUsage(memoryTempResult)

		resourceUsage := ResourceUsage{
			Name:   serviceName,
			CPU:    cpuResult,
			Memory: memoryResult,
		}
		KubearmorResourceUsages = append(KubearmorResourceUsages, resourceUsage)
	}

	if scenario == "WithoutKubeArmor" {
		defaultThroughput = locustThroughputResult
	}
	partialReport := SingleCaseReport{
		Case:                    scenario,
		MetricName:              Metric,
		Users:                   users,
		KubearmorResourceUsages: KubearmorResourceUsages,
		Throughput:              locustThroughputResult,
		PercentageDrop:          ((defaultThroughput - locustThroughputResult) / defaultThroughput) * 100,
		ResourceUsages:          resourceUsages,
	}
	finalReport.Reports = append(finalReport.Reports, partialReport)
}

func parseUsage(result model.Value) (float32, error) {
	re := regexp.MustCompile(`=>\s+([0-9.]+)\s+@`)
	matches := re.FindStringSubmatch(result.String())
	if len(matches) > 1 {
		// Convert the extracted string to float32
		value, err := strconv.ParseFloat(matches[1], 32)
		if err != nil {
			return 0, fmt.Errorf("unable to convert string to float32: %v", err)
		}
		return float32(value), nil
	} else {
		return 0, fmt.Errorf("unable to parse result: %s", result)
	}
}

func getCurrentReplicas(deploymentName string) (int, error) {
	cmd := exec.Command("kubectl", "get", "hpa", deploymentName, "-o", "jsonpath={.status.currentReplicas}")
	var output bytes.Buffer
	cmd.Stdout = &output
	err := cmd.Run()
	if err != nil {
		return 0, fmt.Errorf("error getting the current replicas %V \n%s", err, output.String())
	}
	var replicas int
	err = json.Unmarshal(output.Bytes(), &replicas)
	if err != nil {
		return 0, fmt.Errorf("error unmarshalling current replicas: %v", err)
	}
	return replicas, nil
}

func scaleDeployment(deploymentName string, replicas int) error {
	cmd := exec.Command("kubectl", "scale", "deployment", deploymentName, fmt.Sprintf("--replicas=%d", replicas))
	var output bytes.Buffer
	cmd.Stdout = &output
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error scaling deployment: %v\n%s", err, output.String())
	}
	fmt.Printf("Deployment %s scaled to %d replicas successfully.\n", deploymentName, replicas)
	return nil
}

func deleteHPA(deploymentName string) error {
	cmd := exec.Command("kubectl", "delete", "hpa", deploymentName)
	var output bytes.Buffer
	cmd.Stdout = &output
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error deleting HPA: %v\n%s", err, output.String())
	}
	fmt.Printf("HPA for %s deleted successfully.\n", deploymentName)
	return nil
}

func installKubearmor() error {
	cmd := exec.Command("sh", "-c", "curl -sfL http://get.kubearmor.io/ | sudo sh -s -- -b /usr/local/bin")
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error installing Kubearmor: %v\n%s", err, output.String())
	}
	fmt.Println("Kubearmor installed successfully.")
	return nil
}

func runKarmorInstall() error {
	cmd := exec.Command("karmor", "install")
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error running karmor install: %v\n%s", err, output.String())
	}
	fmt.Println("karmor installed successfully.")
	return nil
}

func configureKubearmorRelay() error {
	// Command to patch the kubearmor-relay deployment
	patch := `{"spec": {"template": {"spec": {"tolerations": [{"key": "color", "operator": "Equal", "value": "blue", "effect": "NoSchedule"}], "nodeSelector": {"nodetype": "node1"}}}}}`
	cmd := exec.Command("kubectl", "patch", "deploy", "kubearmor-relay", "-n", "kubearmor", "--patch", patch)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error configuring kubearmor relay: %v\n%s", err, output.String())
	}
	fmt.Println("kubearmor-relay configured successfully.")
	return nil
}

func changeVisiblity(visiblityMode string) error {
	cmd := exec.Command("kubectl", "annotate", "ns", "default", fmt.Sprintf("kubearmor-visibility=%s", visiblityMode), "--overwrite")
	var output bytes.Buffer
	cmd.Stderr = &output
	cmd.Stdout = &output
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error annotating namespace with %s: %v\n%s", visiblityMode, err, output.String())
	}
	fmt.Printf("Namespace annotated with visibility type %s successfully.\n", visiblityMode)
	return nil
}
func printFinalReport(report FinalReport) {
	fmt.Println("Final Report:")
	for _, caseReport := range report.Reports {
		fmt.Printf("\nCase: %s\n", caseReport.Case)
		fmt.Printf("Metric Name: %s\n", caseReport.MetricName)
		fmt.Printf("Users: %d\n", caseReport.Users)
		for _, usage := range caseReport.KubearmorResourceUsages {
			fmt.Printf("  Service: %s\n", usage.Name)
			fmt.Printf("    CPU: %.2f\n", usage.CPU)
			fmt.Printf("    Memory: %.2f MB\n", usage.Memory)
		}
		fmt.Printf("Throughput: %.2f\n", caseReport.Throughput)
		fmt.Printf("Percentage Drop: %.2f%%\n", caseReport.PercentageDrop)
		fmt.Println("Resource Usages:")
		for _, usage := range caseReport.ResourceUsages {
			fmt.Printf("  Service: %s\n", usage.Name)
			fmt.Printf("    CPU: %.2f\n", usage.CPU)
			fmt.Printf("    Memory: %.2f MB\n", usage.Memory)
		}
	}
}
