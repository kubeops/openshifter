package webhook

import (
	"context"
	"fmt"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"kubeops.dev/openshifter/internal/tracker"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// +kubebuilder:webhook:path=/mutate--v1-namespace,mutating=true,failurePolicy=fail,sideEffects=None,groups="",resources=namespaces,verbs=create;update,versions=v1,name=mnamespace.kb.io,admissionReviewVersions=v1

// NamespaceAnnotator annotates Namespaces
type NamespaceAnnotator struct{}

func (a *NamespaceAnnotator) Default(ctx context.Context, obj runtime.Object) error {
	log := logf.FromContext(ctx)
	ns, ok := obj.(*core.Namespace)
	if !ok {
		return fmt.Errorf("expected a Namespace but got a %T", obj)
	}

	if tracker.NSSkipList.Has(ns.Name) {
		return nil
	}

	curUid, foundUid := ns.Annotations[tracker.KeyUid]
	_, foundFsGroup := ns.Annotations[tracker.KeyFsGroup]
	if foundUid && foundFsGroup {
		return nil
	}

	if !foundUid {
		nuUid := tracker.Uid.Add(tracker.UidRange)
		curUid = fmt.Sprintf("%d/%d", nuUid, tracker.UidRange)
	}

	if ns.Annotations == nil {
		ns.Annotations = map[string]string{}
	}
	if !foundUid {
		ns.Annotations[tracker.KeyUid] = curUid
	}
	if !foundFsGroup {
		ns.Annotations[tracker.KeyFsGroup] = curUid
	}
	log.Info("Annotated Namespace")

	return nil
}
