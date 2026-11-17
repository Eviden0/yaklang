package resources_monitor

import (
	"github.com/yaklang/yaklang/common/consts"
)

type EmbedResourcesMonitor struct {
	resourceID  string
	buildInHash string
}

func NewEmbedResourcesMonitor(resourceID string, buildInHash string) *EmbedResourcesMonitor {
	return &EmbedResourcesMonitor{
		resourceID:  resourceID,
		buildInHash: buildInHash,
	}
}

func (m *EmbedResourcesMonitor) getCurrentHash(currentHashGetter func() string) string {
	if consts.IsDevMode() {
		return currentHashGetter()
	} else {
		return m.buildInHash
	}
}

func (m *EmbedResourcesMonitor) MonitorModifiedWithAction(currentHashGetter func() string, callBack func() error) error {
	resourceHash := m.getCurrentHash(currentHashGetter)
	if resourceHash != getResourceHash(m.resourceID) {
		err := callBack()
		if err != nil {
			return err
		} else {
			setResourceHash(m.resourceID, resourceHash)
		}
	}
	return nil
}
