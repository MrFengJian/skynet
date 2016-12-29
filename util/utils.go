package util

import (
        "os"
        "fmt"
        "strings"
        log "github.com/Sirupsen/logrus"
        "github.com/containernetworking/cni/pkg/skel"
        "github.com/containernetworking/cni/pkg/types"
)

// AddIgnoreUnknownArgs appends the 'IgnoreUnknown=1' option to CNI_ARGS before calling the IPAM plugin. Otherwise, it will
// complain about the Kubernetes arguments. See https://github.com/kubernetes/kubernetes/pull/24983
func AddIgnoreUnknownArgs() error {
        cniArgs := "IgnoreUnknown=1"
        if os.Getenv("CNI_ARGS") != "" {
                cniArgs = fmt.Sprintf("%s;%s", cniArgs, os.Getenv("CNI_ARGS"))
        }
        return os.Setenv("CNI_ARGS", cniArgs)
}


// Set up logging for both skynet  using the provided log level,
func ConfigureLogging(logLevel string) {
        if strings.EqualFold(logLevel, "debug") {
                log.SetLevel(log.DebugLevel)
        } else if strings.EqualFold(logLevel, "info") {
                log.SetLevel(log.InfoLevel)
        } else {
                // Default level
                log.SetLevel(log.WarnLevel)
        }

        log.SetOutput(os.Stderr)
}

// Create a logger which always includes common fields
func CreateContextLogger(workload string) *log.Entry {
        // A common pattern is to re-use fields between logging statements by re-using
        // the logrus.Entry returned from WithFields()
        contextLogger := log.WithFields(log.Fields{
                "Workload": workload,
        })

        return contextLogger
}

func GetPortIdentifier(args *skel.CmdArgs) (portName, podName, namespace string, err error) {
        // Determine if running under k8s by checking the CNI args
        k8sArgs := K8sArgs{}
        if err = types.LoadArgs(args.Args, &k8sArgs); err != nil {
                return portName, "", "", err
        }

        if string(k8sArgs.K8S_POD_NAMESPACE) != "" && string(k8sArgs.K8S_POD_NAME) != "" {
                portName = fmt.Sprintf("%s_%s", k8sArgs.K8S_POD_NAMESPACE, k8sArgs.K8S_POD_NAME)
        } else {
                portName = args.ContainerID
        }
        return portName, string(k8sArgs.K8S_POD_NAME), string(k8sArgs.K8S_POD_NAMESPACE), nil
}