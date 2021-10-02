package faults

import (
	"fmt"

	"github.com/Kaese72/sdup-lib/sduptemplates"
)

type EntityType string

const (
	ETDevice     EntityType = "Device"
	ETAttribute  EntityType = "Attribute"
	ETCapability EntityType = "Capability"
)

type ErrEntityNotFound struct {
	ID         sduptemplates.DeviceID
	EntityType EntityType
}

func (err ErrEntityNotFound) Error() string {
	return fmt.Sprintf("Could not find '%s' with ID='%s'", err.EntityType, err.ID)
}
