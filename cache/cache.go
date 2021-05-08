package cache

import (
	log "github.com/Kaese72/sdup-lib/logging"
	"github.com/Kaese72/sdup-lib/sduptemplates"
)

type SDUPCache interface {
	//SDUPCache is naturally a target
	Initialize() ([]sduptemplates.DeviceSpec, chan sduptemplates.DeviceUpdate, error)

	//Device
	Device(sduptemplates.DeviceID) (sduptemplates.DeviceSpec, error)
	//FIXME searchable
	Devices() ([]sduptemplates.DeviceSpec, error)
	//Attributes
	//DeviceAttributes(sduptemplates.DeviceID) (sduptemplates.AttributeSpecMap, error)
	//DeviceAttribute(sduptemplates.DeviceID, sduptemplates.AttributeKey) (sduptemplates.AttributeSpec, error)

	//Capabilities
	//DeviceCapabilities(sduptemplates.DeviceID) (sduptemplates.CapabilitySpecMap, error)
	//DeviceCapability(sduptemplates.DeviceID, sduptemplates.CapabilityKey) (sduptemplates.CapabilitySpec, error)

	TriggerCapability(sduptemplates.DeviceID, sduptemplates.CapabilityKey, sduptemplates.CapabilityArgument) error
}

type SDUPCacheImpl struct {
	target      sduptemplates.SDUPTarget
	devices     map[sduptemplates.DeviceID]sduptemplates.DeviceSpec
	initialized bool
}

func NewSDUPCache(target sduptemplates.SDUPTarget) SDUPCache {
	return &SDUPCacheImpl{
		target:  target,
		devices: map[sduptemplates.DeviceID]sduptemplates.DeviceSpec{},
	}
}

func (cache *SDUPCacheImpl) Initialize() (specs []sduptemplates.DeviceSpec, channel chan sduptemplates.DeviceUpdate, err error) {
	if cache.initialized {
		panic("Hue target already initialized")
	}
	cache.initialized = true

	var updateChan chan sduptemplates.DeviceUpdate
	specs, updateChan, err = cache.Initialize()

	//Populate cache
	for i := range specs {
		cache.devices[specs[i].ID] = specs[i]
	}

	go func() {
		for update := range updateChan {
			//FIXME Lock cache writes
			//FIXME Handle detection of devices first
			if device, ok := cache.devices[update.ID]; ok {
				for attrKey, attrState := range update.AttributesDiff {
					if attrValue, ok := device.Attributes[attrKey]; ok {
						attrValue.AttributeState = attrState
						device.Attributes[attrKey] = attrValue

					} else {
						log.Error("Unknown device attribute", map[string]string{"device": string(update.ID), "attribute": string(attrKey)})
					}

				}
				cache.devices[update.ID] = device

			} else {
				log.Error("Unknown device", map[string]string{"device": string(update.ID)})
			}

			//Pass the update forward
			channel <- update
		}
	}()

	return
}

func (cache SDUPCacheImpl) Device(deviceID sduptemplates.DeviceID) (retSpec sduptemplates.DeviceSpec, err error) {
	retSpec, ok := cache.devices[deviceID]
	if !ok {
		err = ErrDeviceNotFound{ID: deviceID}
	}
	return
}

func (cache SDUPCacheImpl) Devices() (retSpecs []sduptemplates.DeviceSpec, err error) {
	for _, val := range cache.devices {
		retSpecs = append(retSpecs, val)
	}
	return
}
