package cache

import (
	"fmt"

	"github.com/Kaese72/sdup-lib/sduptemplates"
)

type ErrDeviceNotFound struct {
	ID sduptemplates.DeviceID
}

func (err ErrDeviceNotFound) Error() string {
	return fmt.Sprintf("Could not find Device with ID='%s'", err.ID)
}
