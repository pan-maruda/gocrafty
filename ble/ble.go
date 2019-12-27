package ble

import (
	"encoding/binary"
	"fmt"
	"log"

	"github.com/paypal/gatt"
)

var CurrentTempUUID = gatt.MustParseUUID("000000114c454b434942265a524f5453")
var TempSetpointUUID = gatt.MustParseUUID("000000214c454b434942265a524f5453")
var BoostTempUUID = gatt.MustParseUUID("000000314c454b434942265a524f5453")
var BatteryLevelUUID = gatt.MustParseUUID("000000414c454b434942265a524f5453")

var DataServiceUUID = gatt.MustParseUUID("00000001-4c45-4b43-4942-265a524f5453")

var MetaServiceUUID = gatt.MustParseUUID("00000002-4c45-4b43-4942-265a524f5453")
var ModelUUID = gatt.MustParseUUID("00000022-4c45-4b43-4942-265a524f5453")
var VersionUUID = gatt.MustParseUUID("00000032-4c45-4b43-4942-265a524f5453")
var SerialUUID = gatt.MustParseUUID("00000052-4c45-4b43-4942-265a524f5453")

type CraftyMeta struct {
	modelName    string
	fwVersion    string
	serialNumber string
	id           string
}

type CraftyStatus struct {
	id           string
	currentTemp  uint16
	tempSetpoint uint16
	boostTemp    uint16
	batteryLevel uint16
}

func (c CraftyMeta) ModelName() string {
	return c.modelName
}

func (c CraftyMeta) FwVersion() string {
	return c.fwVersion
}

func (c CraftyMeta) SerialNumber() string {
	return c.serialNumber
}

func (c CraftyMeta) ID() string {
	return c.id
}

func (c CraftyStatus) ID() string {
	return c.id
}

func (c CraftyStatus) CurrentTemp() uint16 {
	return c.currentTemp
}

func (c CraftyStatus) Setpoint() uint16 {
	return c.tempSetpoint
}

func (c CraftyStatus) BoostTemp() uint16 {
	return c.boostTemp
}

func (c CraftyStatus) BatteryLevel() uint16 {
	return c.batteryLevel
}

func (c CraftyStatus) String() string {
	return fmt.Sprintf("Current Temp: %d C\nSetpoint: %d C\nBoost: +%d C\n Battery level: %d%%",
		c.CurrentTemp()/10,
		c.Setpoint()/10,
		c.BoostTemp()/10,
		c.BatteryLevel())
}

func (c CraftyMeta) String() string {
	return fmt.Sprintf(
		"%s SN:%s FW:%s ID:%s",
		c.ModelName(), c.SerialNumber(), c.FwVersion(), c.ID(),
	)
}

func ReadUint16(p gatt.Peripheral, c *gatt.Characteristic) (uint16, error) {
	value, err := p.ReadCharacteristic(c)
	if err != nil {
		return 0, err
	}

	if len(value) == 2 {
		intValue := binary.LittleEndian.Uint16(value[0:])
		return intValue, nil
	}
	return 0, fmt.Errorf("characteristic %s read != 2 bytes: %x", c.UUID(), value)
}
func ReadString(p gatt.Peripheral, c *gatt.Characteristic) (string, error) {
	bytes, err := p.ReadCharacteristic(c)

	if err != nil {
		return "", err
	}

	return string(bytes[:clen(bytes)]), nil
}

func clen(n []byte) int {
	for i := 0; i < len(n); i++ {
		if n[i] == 0 {
			return i
		}
	}
	return len(n)
}

// Reads characteristics from the devices
func ReadMetadataService(p gatt.Peripheral, svc *gatt.Service) (*CraftyMeta, error) {
	chars, err := p.DiscoverCharacteristics([]gatt.UUID{ModelUUID, VersionUUID, SerialUUID}, svc)

	if err != nil {
		log.Fatalf("Failed to discover characteristics, err :%s\n", err)
		return nil, err
	}

	metadata := CraftyMeta{}
	metadata.id = p.ID()
	for _, char := range chars {

		value, err := ReadString(p, char)
		if err != nil {
			fmt.Printf("%s read failed: %s", svc.UUID(), err)
		}

		if char.UUID().Equal(ModelUUID) {
			metadata.modelName = value
		}
		if char.UUID().Equal(VersionUUID) {
			metadata.fwVersion = value
		}
		if char.UUID().Equal(SerialUUID) {
			metadata.serialNumber = value
		}
	}

	return &metadata, nil
}

func ReadDataServiceCharacteristics(p gatt.Peripheral, svc *gatt.Service) (*CraftyStatus, error) {
	chars, err := p.DiscoverCharacteristics([]gatt.UUID{TempSetpointUUID, CurrentTempUUID, BoostTempUUID, BatteryLevelUUID}, svc)

	if err != nil {
		log.Fatalf("Failed to discover characteristics, err :%s\n", err)
		return nil, err
	}

	craftyStatus := CraftyStatus{}

	for _, char := range chars {
		intValue, err := ReadUint16(p, char)
		if err != nil {
			// TODO check if the characteristic is really in the ones we want
			// fmt.Printf(" read failed: %s", err)
		} else {
			if char.UUID().Equal(TempSetpointUUID) {
				craftyStatus.tempSetpoint = intValue
			}
			if char.UUID().Equal(BoostTempUUID) {
				craftyStatus.boostTemp = intValue
			}
			if char.UUID().Equal(CurrentTempUUID) {
				craftyStatus.currentTemp = intValue
			}
			if char.UUID().Equal(BatteryLevelUUID) {
				craftyStatus.batteryLevel = intValue
			}
		}

	}

	return &craftyStatus, nil
}
