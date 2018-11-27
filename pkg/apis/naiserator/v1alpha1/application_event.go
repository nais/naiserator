package v1alpha1

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (in *Application) CreateEvent(reason, message, typeStr string) *corev1.Event {
	return &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "naiserator-event",
			Namespace:    in.Namespace,
		},
		ReportingController: "naiserator",
		ReportingInstance:   "naiserator",
		Action:              reason,
		Reason:              reason,
		InvolvedObject:      in.GetObjectReference(),
		Source:              corev1.EventSource{Component: "naiserator"},
		Message:             message,
		EventTime: metav1.MicroTime{Time: time.Now()},
		Type: typeStr,
	}
}
