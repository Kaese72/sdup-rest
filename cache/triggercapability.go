package cache

import "github.com/Kaese72/sdup-lib/sduptemplates"

func (cache SDUPCacheImpl) TriggerCapability(deviceID sduptemplates.DeviceID, capKey sduptemplates.CapabilityKey, capArg sduptemplates.CapabilityArgument) error {
	return cache.target.TriggerCapability(deviceID, capKey, capArg)
}
