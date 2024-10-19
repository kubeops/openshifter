package tracker

import (
	"context"
	"fmt"
	core "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
	"sync/atomic"
)

const (
	KeyUid     = "openshift.io/sa.scc.uid-range"
	KeyFsGroup = "openshift.io/sa.scc.supplemental-groups"
	UidRange   = 10000
)

var (
	Uid atomic.Int64
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

	var curUid, curFsGroupUid int64

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

	if curUid > 0 {
		if curUid != curFsGroupUid {
			return fmt.Errorf("runAsUser %d and fsGroup %d uid range does not match", curUid, curFsGroupUid)
		}
		Uid.Store(curUid)
	}
	return nil
}
