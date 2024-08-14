package Ch_07

import (
	"k8s.io/client-go/kubernetes"
	"strconv"
)

func checkMinimalServerVersion(
	clientset kubernetes.Interface,
	minMinor int,
) (bool, error) {
	discoveryClient := clientset.Discovery()
	info, err := discoveryClient.ServerVersion()
	if err != nil {
		return false, err
	}
	major, err := strconv.Atoi(info.Major)
	if err != nil {
		return false, err
	}
	minor, err := strconv.Atoi(info.Minor)
	if err != nil {
		return false, err
	}

	return major == 1 && minor >= minMinor, nil
}
