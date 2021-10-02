package cache

import (
	"errors"
	"fmt"

	log "github.com/Kaese72/sdup-lib/logging"
	"github.com/Kaese72/sdup-lib/sduptemplates"
	"github.com/Kaese72/sdup-rest/cache/filters"
	"github.com/Kaese72/sdup-rest/faults"
)

type SDUPCache interface {
	//SDUPCache is naturally a target
	Initialize() ([]sduptemplates.DeviceSpec, chan sduptemplates.DeviceUpdate, error)

	//Device
	Device(sduptemplates.DeviceID) (sduptemplates.DeviceSpec, error)
	//FIXME searchable
	Devices(filters.AttributeFilters) ([]sduptemplates.DeviceSpec, error)
	//Attributes
	//DeviceAttributes(sduptemplates.DeviceID) (sduptemplates.AttributeSpecMap, error)
	//DeviceAttribute(sduptemplates.DeviceID, sduptemplates.AttributeKey) (sduptemplates.AttributeSpec, error)

	//Capabilities
	//DeviceCapabilities(sduptemplates.DeviceID) (sduptemplates.CapabilitySpecMap, error)
	//DeviceCapability(sduptemplates.DeviceID, sduptemplates.CapabilityKey) (sduptemplates.CapabilitySpec, error)

	TriggerCapability(sduptemplates.DeviceID, sduptemplates.CapabilityKey, sduptemplates.CapabilityArgument) error
}

type DeviceStore interface {
	Device(sduptemplates.DeviceID) (sduptemplates.DeviceSpec, error)
	Devices(filters.AttributeFilters) ([]sduptemplates.DeviceSpec, error)
	UpdateDevice(sduptemplates.DeviceUpdate) error
	InsertDevice(sduptemplates.DeviceSpec) error
}

type DeviceStoreImpl struct {
	devices map[sduptemplates.DeviceID]sduptemplates.DeviceSpec
}

func (store *DeviceStoreImpl) Device(deviceID sduptemplates.DeviceID) (spec sduptemplates.DeviceSpec, err error) {
	if device, ok := store.devices[deviceID]; ok {
		spec = device

	} else {
		err = sduptemplates.NoSuchDevice
	}
	return
}

func deviceMatchesFilters(device sduptemplates.DeviceSpec, filters filters.AttributeFilters) (match bool, err error) {
	for _, filter := range filters {
		operator, err := filter.GetOperator()
		if err != nil {
			// Invalid operators lead to wacky scenarios
			return false, err
		}

		if _, _, err := filter.Key.KeyValKeys(); err == nil {
			// Composite key, we should use keyval
			return false, errors.New("keyval currently not supported")

		} else {
			if _, ok := device.Attributes[sduptemplates.AttributeKey(filter.Key)]; !ok {
				// Not having the attribute counts as false
				return false, nil
			}
			// Simple key
			// Get value based on what type the comparator is
			switch comp := filter.Value.(type) {
			case int:
				return matchNumericComparison(device.Attributes[sduptemplates.AttributeKey(filter.Key)].AttributeState.Numeric, float32(comp), operator)

			case float32:
				return matchNumericComparison(device.Attributes[sduptemplates.AttributeKey(filter.Key)].AttributeState.Numeric, comp, operator)

			case string:
				return matchStringComparison(device.Attributes[sduptemplates.AttributeKey(filter.Key)].AttributeState.Text, comp, operator)

			case bool:
				return matchBooleanComparison(device.Attributes[sduptemplates.AttributeKey(filter.Key)].AttributeState.Boolean, comp, operator)

			default:
				// FIXME log better
				return false, errors.New("unsupported filter type")
			}
		}

	}
	return true, nil
}

func matchBooleanComparison(attrVal *bool, compVal bool, operator filters.Operator) (bool, error) {
	if attrVal == nil {
		// Not having the value set is considered
		return false, nil
	}

	switch operator {
	case filters.Equal:
		return *attrVal == compVal, nil
	default:
		return false, errors.New("not a supported operand")
	}
}

func matchStringComparison(attrVal *string, compVal string, operator filters.Operator) (bool, error) {
	if attrVal == nil {
		// Not having the value set is considered
		return false, nil
	}

	switch operator {
	case filters.Equal:
		return *attrVal == compVal, nil
	default:
		return false, errors.New("not a supported operand")
	}
}

func matchNumericComparison(attrVal *float32, compVal float32, operator filters.Operator) (bool, error) {
	if attrVal == nil {
		// Not having the value set is considered
		return false, nil
	}

	switch operator {
	case filters.Equal:
		return *attrVal == compVal, nil
	default:
		return false, errors.New("not a supported operand")
	}
}

func (store *DeviceStoreImpl) Devices(attrFilters filters.AttributeFilters) ([]sduptemplates.DeviceSpec, error) {
	specs := []sduptemplates.DeviceSpec{}
	for _, device := range store.devices {
		match, err := deviceMatchesFilters(device, attrFilters)
		if err != nil {
			return nil, err
		}

		if match {
			specs = append(specs, device)
		}
	}
	return specs, nil
}

func (store *DeviceStoreImpl) UpdateDevice(update sduptemplates.DeviceUpdate) error {
	if device, ok := store.devices[update.ID]; ok {
		for attrKey, attrChange := range update.AttributesDiff {
			if attr, ok := device.Attributes[attrKey]; ok {
				attr.AttributeState = attrChange
				device.Attributes[attrKey] = attr
			} else {
				return sduptemplates.NoSuchAttribute
			}
		}
		store.devices[update.ID] = device

	} else {
		return sduptemplates.NoSuchDevice
	}
	return nil
}

func (store *DeviceStoreImpl) InsertDevice(spec sduptemplates.DeviceSpec) error {
	store.devices[spec.ID] = spec
	return nil
}

type SDUPCacheImpl struct {
	target      sduptemplates.SDUPTarget
	devices     DeviceStore
	updateChan  chan sduptemplates.DeviceUpdate
	initialized bool
}

func NewSDUPCache(target sduptemplates.SDUPTarget) SDUPCache {
	return &SDUPCacheImpl{
		target:     target,
		devices:    &DeviceStoreImpl{devices: map[sduptemplates.DeviceID]sduptemplates.DeviceSpec{}},
		updateChan: make(chan sduptemplates.DeviceUpdate, 10),
	}
}

func (cache *SDUPCacheImpl) Initialize() (specs []sduptemplates.DeviceSpec, channel chan sduptemplates.DeviceUpdate, err error) {
	if cache.initialized {
		panic("SDUP cache already initialized")
	}
	cache.initialized = true

	var upstreamChan chan sduptemplates.DeviceUpdate
	specs, upstreamChan, err = cache.target.Initialize()

	//Populate cache
	for i := range specs {
		cache.devices.InsertDevice(specs[i])
	}

	go func() {
		for update := range upstreamChan {
			log.Info(fmt.Sprintf("Received update on device %s", string(update.ID)))
			//FIXME Lock cache writes
			//FIXME Handle detection of devices first
			if device, err := cache.devices.Device(update.ID); err == nil {
				for attrKey, attrState := range update.AttributesDiff {
					if attrValue, ok := device.Attributes[attrKey]; ok {
						attrValue.AttributeState = attrState
						device.Attributes[attrKey] = attrValue

					} else {
						log.Error("Unknown device attribute", map[string]string{"device": string(update.ID), "attribute": string(attrKey)})
					}

				}
				cache.devices.InsertDevice(device)

			} else {
				//FIXME Create device events
				log.Error("Unknown device", map[string]string{"device": string(update.ID)})
			}

			//Pass the update forward
			cache.updateChan <- update
		}
	}()
	channel = cache.updateChan

	return
}

func (cache SDUPCacheImpl) Device(deviceID sduptemplates.DeviceID) (retSpec sduptemplates.DeviceSpec, err error) {
	retSpec, err2 := cache.devices.Device(deviceID)
	if err2 != nil {
		err = faults.ErrEntityNotFound{ID: deviceID, EntityType: faults.ETDevice}
	}
	return
}

func (cache SDUPCacheImpl) Devices(attrFilters filters.AttributeFilters) (retSpecs []sduptemplates.DeviceSpec, err error) {
	retSpecs, err = cache.devices.Devices(attrFilters)
	return
}
