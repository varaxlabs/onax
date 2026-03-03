package watcher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestGetExitCode_Terminated(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ExitCode: 137,
						},
					},
				},
			},
		},
	}

	code := GetExitCode(pod)
	assert.NotNil(t, code)
	assert.Equal(t, int32(137), *code)
}

func TestGetExitCode_NoTerminated(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{},
					},
				},
			},
		},
	}

	code := GetExitCode(pod)
	assert.Nil(t, code)
}

func TestGetExitCode_EmptyStatuses(t *testing.T) {
	pod := &corev1.Pod{}
	code := GetExitCode(pod)
	assert.Nil(t, code)
}

func TestGetErrorMessage_Terminated(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							Reason:  "OOMKilled",
							Message: "container exceeded memory limit",
						},
					},
				},
			},
		},
	}

	msg := GetErrorMessage(pod)
	assert.Equal(t, "OOMKilled: container exceeded memory limit", msg)
}

func TestGetErrorMessage_Waiting(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason:  "CrashLoopBackOff",
							Message: "back-off 5m0s restarting failed container",
						},
					},
				},
			},
		},
	}

	msg := GetErrorMessage(pod)
	assert.Equal(t, "CrashLoopBackOff: back-off 5m0s restarting failed container", msg)
}

func TestGetErrorMessage_Running(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{},
					},
				},
			},
		},
	}

	msg := GetErrorMessage(pod)
	assert.Empty(t, msg)
}
