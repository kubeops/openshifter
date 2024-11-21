package webhook

import (
	"context"
	"fmt"
	"strings"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	clustermeta "kmodules.xyz/client-go/cluster"
	"kubeops.dev/openshifter/internal/tracker"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// +kubebuilder:webhook:path=/mutate--v1-namespace,mutating=true,failurePolicy=fail,sideEffects=None,groups="",resources=namespaces,verbs=create;update,versions=v1,name=mnamespace.kb.io,admissionReviewVersions=v1

// NamespaceAnnotator annotates Namespaces
type NamespaceAnnotator struct {
	Mapper meta.RESTMapper
}

func (a *NamespaceAnnotator) Default(ctx context.Context, obj runtime.Object) error {
	log := logf.FromContext(ctx)
	ns, ok := obj.(*core.Namespace)
	if !ok {
		return fmt.Errorf("expected a Namespace but got a %T", obj)
	}

	if tracker.NSSkipList.Has(ns.Name) {
		return nil
	}
	if HasPSA(ns) {
		return nil
	}

	if ns.Annotations == nil {
		ns.Annotations = map[string]string{}
	}
	ns.Annotations["pod-security.kubernetes.io/enforce"] = "restricted"

	if clustermeta.IsOpenShiftManaged(a.Mapper) {
		curUid, foundUid := ns.Annotations[tracker.KeyUid]
		_, foundFsGroup := ns.Annotations[tracker.KeyFsGroup]
		if foundUid && foundFsGroup {
			return nil
		}

		if !foundUid {
			nuUid := tracker.Uid.Add(tracker.UidRange)
			curUid = fmt.Sprintf("%d/%d", nuUid, tracker.UidRange)
		}

		if !foundUid {
			ns.Annotations[tracker.KeyUid] = curUid
		}
		if !foundFsGroup {
			ns.Annotations[tracker.KeyFsGroup] = curUid
		}
	}
	log.Info("Annotated Namespace")

	return nil
}

func HasPSA(ns *core.Namespace) bool {
	for k := range ns.Annotations {
		if k == "pod-security.kubernetes.io/enforce" ||
			strings.HasPrefix(k, "pod-security.kubernetes.io/enforce-") ||
			k == "pod-security.kubernetes.io/audit" ||
			strings.HasPrefix(k, "pod-security.kubernetes.io/audit-") ||
			k == "pod-security.kubernetes.io/warn" ||
			strings.HasPrefix(k, "pod-security.kubernetes.io/warn-") {
			return true
		}
	}
	return false
}
