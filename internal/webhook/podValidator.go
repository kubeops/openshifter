package webhook

import (
	"context"
	"fmt"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	clustermeta "kmodules.xyz/client-go/cluster"
	"kubeops.dev/openshifter/internal/tracker"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/validate--v1-pod,mutating=false,failurePolicy=fail,sideEffects=None,groups="",resources=pods,verbs=create;update,versions=v1,name=vpod.kb.io,admissionReviewVersions=v1

// PodValidator validates Pods
type PodValidator struct {
	client.Reader
	authorizer.Authorizer
	Mapper meta.RESTMapper
}

// validate admits a pod if a specific annotation exists.
func (v *PodValidator) validate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	log := logf.FromContext(ctx)
	pod, ok := obj.(*core.Pod)
	if !ok {
		return nil, fmt.Errorf("expected a Pod but got a %T", obj)
	}

	if tracker.NSSkipList.Has(pod.Namespace) {
		return nil, nil
	}
	if !clustermeta.IsOpenShiftManaged(v.Mapper) {
		return nil, nil
	}

	log.Info("Validating Pod")

	// https://examples.openshift.pub/deploy/scc-anyuid/
	attrs := authorizer.AttributesRecord{
		User: &user.DefaultInfo{
			Name:   fmt.Sprintf("system:serviceaccount:%s:%s", pod.Namespace, pod.Name),
			Groups: []string{"system:serviceaccounts"},
		},
		Verb:            "use",
		Namespace:       pod.Namespace,
		APIGroup:        "security.openshift.io",
		Resource:        "securitycontextconstraints",
		Name:            "anyuid",
		ResourceRequest: true,
	}
	decision, _, err := v.Authorizer.Authorize(ctx, attrs)
	if err != nil {
		return nil, errors.NewInternalError(err)
	}
	if decision == authorizer.DecisionAllow {
		return nil, nil
	}

	uidStart, uidRange, err := tracker.GetUid(v.Reader, pod.Namespace)
	if err != nil {
		return nil, err
	} else if uidStart == tracker.UidNone {
		return nil, nil
	}

	var runAsUser, runAsGroup, fsGroup int64 = -1, -1, -1
	if pod.Spec.SecurityContext != nil {
		if pod.Spec.SecurityContext.RunAsUser != nil {
			runAsUser = *pod.Spec.SecurityContext.RunAsUser
		}
		if pod.Spec.SecurityContext.RunAsGroup != nil {
			runAsGroup = *pod.Spec.SecurityContext.RunAsGroup
		}
		if pod.Spec.SecurityContext.FSGroup != nil {
			fsGroup = *pod.Spec.SecurityContext.FSGroup
		}
	}

	if runAsUser != tracker.UidNone && (runAsUser < uidStart || runAsUser > uidStart+uidRange) {
		return nil, fmt.Errorf("runAsUser %d must be within range %d/%d", runAsUser, uidStart, uidRange)
	}

	if runAsGroup != tracker.UidNone && (runAsGroup < uidStart || runAsGroup > uidStart+uidRange) {
		return nil, fmt.Errorf("runAsGroup %d must be within range %d/%d", runAsGroup, uidStart, uidRange)
	}

	if fsGroup != tracker.UidNone && (fsGroup < uidStart || fsGroup > uidStart+uidRange) {
		return nil, fmt.Errorf("fsGroup %d must be within range %d/%d", fsGroup, uidStart, uidRange)
	}

	for _, c := range pod.Spec.Containers {
		cUid, cGid := runAsUser, runAsGroup
		if c.SecurityContext != nil {
			if c.SecurityContext.RunAsUser != nil {
				cUid = *c.SecurityContext.RunAsUser
			}
			if c.SecurityContext.RunAsGroup != nil {
				cGid = *c.SecurityContext.RunAsGroup
			}
		}

		if cUid == tracker.UidNone {
			return nil, fmt.Errorf("container %s runAsUser is not set", c.Name)
		} else if cUid < uidStart || cUid > uidStart+uidRange {
			return nil, fmt.Errorf("container %s runAsUser %d must be within range %d/%d", c.Name, cUid, uidStart, uidRange)
		}

		if cGid != tracker.UidNone && (cGid < uidStart || cGid > uidStart+uidRange) {
			return nil, fmt.Errorf("container %s runAsGroup %d must be within range %d/%d", c.Name, cGid, uidStart, uidRange)
		}
	}

	return nil, nil
}

func (v *PodValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return v.validate(ctx, obj)
}

func (v *PodValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	return v.validate(ctx, newObj)
}

func (v *PodValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return v.validate(ctx, obj)
}
