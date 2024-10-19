package tracker

import (
	"context"
	"fmt"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
	"sync/atomic"
)

const (
	KeyUid     = "openshift.io/sa.scc.uid-range"
	KeyFsGroup = "openshift.io/sa.scc.supplemental-groups"
	UidRange   = 10000
	UidNone    = -1
)

var (
	Uid        atomic.Int64
	NSSkipList = sets.New[string]("kube-system", "local-path-storage", "cert-manager", "openshifter-system", "aceshifter-system")
)

/*
   openshift.io/sa.scc.supplemental-groups: 1000580000/10000
   openshift.io/sa.scc.uid-range: 1000580000/10000
*/

func Init(kc client.Reader) error {
	var list core.NamespaceList
	err := kc.List(context.TODO(), &list)
	if err != nil {
		return err
	}

	var curUid, curFsGroupUid int64 = -1, -1

	for _, ns := range list.Items {
		if v, ok := ns.Annotations[KeyUid]; ok {
			if strUid, _, ok := strings.Cut(v, "/"); ok {
				if uid, err := strconv.ParseInt(strUid, 10, 64); err == nil && uid > curUid {
					curUid = uid
				}
			}
		}

		if v, ok := ns.Annotations[KeyFsGroup]; ok {
			if strUid, _, ok := strings.Cut(v, "/"); ok {
				if uid, err := strconv.ParseInt(strUid, 10, 64); err == nil && uid > curFsGroupUid {
					curFsGroupUid = uid
				}
			}
		}
	}

	if curUid > -1 {
		if curUid != curFsGroupUid {
			return fmt.Errorf("runAsUser %d and fsGroup %d uid range does not match", curUid, curFsGroupUid)
		}
	} else {
		curUid = 1000100000
	}
	Uid.Store(curUid)

	return nil
}

func GetUid(kc client.Reader, ns string) (int64, int64, error) {
	var obj core.Namespace
	err := kc.Get(context.TODO(), client.ObjectKey{Name: ns}, &obj)
	if err != nil {
		return UidNone, UidNone, err
	}

	curUid, foundUid := obj.Annotations[KeyUid]
	curFsGroupUid, foundFsGroup := obj.Annotations[KeyFsGroup]
	if !foundUid && !foundFsGroup {
		return UidNone, UidNone, nil
	}
	if curUid != curFsGroupUid {
		return UidNone, UidNone, fmt.Errorf("runAsUser %s and fsGroup %s uid range does not match", curUid, curFsGroupUid)
	}

	strUid, strRange, ok := strings.Cut(curUid, "/")
	if !ok {
		return UidNone, UidNone, fmt.Errorf("%s annotation value is not in <start>/<range> format", KeyUid)
	}

	uid, err := strconv.ParseInt(strUid, 10, 64)
	if err != nil {
		return UidNone, UidNone, fmt.Errorf("%s annotation start uid is not an interger", KeyUid)
	}
	uidRange, err := strconv.ParseInt(strRange, 10, 64)
	if err != nil {
		return UidNone, UidNone, fmt.Errorf("%s annotation range is not an interger", KeyUid)
	}
	return uid, uidRange, nil
}
