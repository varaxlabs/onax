package watcher

import (
	corev1 "k8s.io/api/core/v1"
)

// GetExitCode extracts the exit code from a pod's first terminated container.
func GetExitCode(pod *corev1.Pod) *int32 {
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Terminated != nil {
			code := cs.State.Terminated.ExitCode
			return &code
		}
	}
	return nil
}

// GetErrorMessage extracts an error message from a pod's container status.
func GetErrorMessage(pod *corev1.Pod) string {
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Terminated != nil && cs.State.Terminated.Reason != "" {
			return cs.State.Terminated.Reason + ": " + cs.State.Terminated.Message
		}
		if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
			return cs.State.Waiting.Reason + ": " + cs.State.Waiting.Message
		}
	}
	return ""
}
