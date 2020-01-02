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
var LedUUID = gatt.MustParseUUID("000000514c454b434942265a524f5453")

var DataServiceUUID = gatt.MustParseUUID("00000001-4c45-4b43-4942-265a524f5453")

var MetaServiceUUID = gatt.MustParseUUID("00000002-4c45-4b43-4942-265a524f5453")
var ModelUUID = gatt.MustParseUUID("00000022-4c45-4b43-4942-265a524f5453")
var VersionUUID = gatt.MustParseUUID("00000032-4c45-4b43-4942-265a524f5453")
var SerialUUID = gatt.MustParseUUID("00000052-4c45-4b43-4942-265a524f5453")

// CraftyMeta contains metadata about connected Crafty, such as FW version, serial number etc.
type CraftyMeta struct {
	modelName    string
	fwVersion    string
	serialNumber string
	id           string
}

// CraftyStatus represents current state of Crafty, most important to end users.
type CraftyStatus struct {
	id           string
	currentTemp  uint16
	tempSetpoint uint16
	boostTemp    uint16
	batteryLevel uint16
	led          uint16
}

// DataService struct contains pointers to discovered characteristics
// used by this app
type DataService struct {
	currentTemp  *gatt.Characteristic
	tempSetpoint *gatt.Characteristic
	boostTemp    *gatt.Characteristic
	batteryLevel *gatt.Characteristic
	led          *gatt.Characteristic
}

func (ds DataService) String() string {
	return fmt.Sprintf("currentTemp  %s, tempSetpoint %s, boostTemp    %s, batteryLevel %s, led %s",
		ds.currentTemp.UUID(),
		ds.tempSetpoint.UUID(),
		ds.boostTemp.UUID(),
		ds.batteryLevel.UUID(),
		ds.led.UUID(),
	)
}

// SubscribeBattery subscribes to battery level notifications sent by Crafty.
func (ds DataService) SubscribeBattery(p gatt.Peripheral, f func(uint16, error)) {
	callback := func(c *gatt.Characteristic, byteValue []byte, err error) {
		if err != nil {
			fmt.Printf("Error received in notify handler %s", err)
			return
		}
		intValue, err := ReadUint16(p, c)
		f(intValue, err)
	}

	p.DiscoverDescriptors(nil, ds.batteryLevel)

	err := p.SetNotifyValue(ds.batteryLevel, callback)
	if err != nil {
		fmt.Printf("Failed to subscrbe to battery characteristic: %s\n", err)
	}
}

// SubscribeTemp subscribes to temperature notifications sent by Crafty.
func (ds DataService) SubscribeTemp(p gatt.Peripheral, f func(uint16, error)) {
	callback := func(c *gatt.Characteristic, byteValue []byte, err error) {
		if err != nil {
			fmt.Printf("Error received in notify handler %s", err)
			return
		}
		intValue, err := ReadUint16(p, c)
		f(intValue, err)
	}
	p.DiscoverDescriptors(nil, ds.currentTemp)

	err := p.SetNotifyValue(ds.currentTemp, callback)
	if err != nil {
		fmt.Printf("Failed to subscrbe to current temperature characteristic: %s\n", err)
	}
}

// SetTemp sends a write command to set the temperature setpoint.
// Important: validate this before sending to device. This method does not
// check whether it's a valid temperature.
func (ds DataService) SetTemp(p gatt.Peripheral, temp int) {
	bytes := []byte{0, 0}
	binary.LittleEndian.PutUint16(bytes, uint16(temp*10))

	err := p.WriteCharacteristic(ds.tempSetpoint, bytes, true)
	if err != nil {
		fmt.Printf("failed to set temperature: %s", err)
	}
}

// SetBoost sends a write command to set the boost level.
// Important: validate this before sending to device. This method does not
// check whether it's a valid temperature.
func (ds DataService) SetBoost(p gatt.Peripheral, boost int) {
	bytes := []byte{0, 0}
	binary.LittleEndian.PutUint16(bytes, uint16(boost*10))

	err := p.WriteCharacteristic(ds.boostTemp, bytes, true)
	if err != nil {
		fmt.Printf("failed to set boost: %s", err)
	}
}

// ModelName is model name read from Crafty device.
// Usually something like "Crafty       "
func (c CraftyMeta) ModelName() string {
	return c.modelName
}

// FwVersion is firmware version read from Crafty device.
func (c CraftyMeta) FwVersion() string {
	return c.fwVersion
}

// SerialNumber read from Crafty device.
func (c CraftyMeta) SerialNumber() string {
	return c.serialNumber
}

// ID is the device ID as understood as OS bluetooth stack.
func (c CraftyMeta) ID() string {
	return c.id
}

// ID is the device ID as understood as OS bluetooth stack.
func (c CraftyStatus) ID() string {
	return c.id
}

// CurrentTemp returns read chamber temperature in degrees celsius * 10
// e.g. for current temperature 21.3 deg C this would return 213
func (c CraftyStatus) CurrentTemp() uint16 {
	return c.currentTemp
}

// Setpoint returns set vaping temperature in degrees celsius * 10
// e.g. for 175 deg Celsius this would return 1750
func (c CraftyStatus) Setpoint() uint16 {
	return c.tempSetpoint
}

// BoostTemp returns current boost in degrees celsius * 10
// e.g. for boost +20 deg Celsius this would return 200
func (c CraftyStatus) BoostTemp() uint16 {
	return c.boostTemp
}

// BatteryLevel returns charge level in percent (0-100)
func (c CraftyStatus) BatteryLevel() uint16 {
	return c.batteryLevel
}

// LEDBrightness returns LED Brightness level in percent (0-100)
func (c CraftyStatus) LEDBrightness() uint16 {
	return c.led
}

func (c CraftyStatus) String() string {
	return fmt.Sprintf("Current Temp: %d.%d C\nSetpoint: %d C\nBoost: +%d C\nBattery level: %d%%\nLED brightness: %d%%",
		c.CurrentTemp()/10, c.CurrentTemp()%10,
		c.Setpoint()/10,
		c.BoostTemp()/10,
		c.BatteryLevel(),
		c.LEDBrightness())
}

func (c CraftyMeta) String() string {
	return fmt.Sprintf(
		"Model: [%s] SN: [%s] FW: [%s] ID [%s]",
		c.ModelName(), c.SerialNumber(), c.FwVersion(), c.ID(),
	)
}

// ReadUint16 reads a little-endian uint16 from given characteristic
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

// ReadString reads a NUL-terminated ASCII string from given characteristic
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

// ReadMetadataService fills in a CraftyMetadata struct wiht values read from device.
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

// DiscoverDataService discovers characteristics and returns a pointer to DataService struct
func DiscoverDataService(p gatt.Peripheral, svc *gatt.Service) (*DataService, error) {
	chars, err := p.DiscoverCharacteristics([]gatt.UUID{TempSetpointUUID, CurrentTempUUID, BoostTempUUID, BatteryLevelUUID}, svc)

	if err != nil {
		log.Fatalf("Failed to discover characteristics, err :%s\n", err)
		return nil, err
	}

	dataService := DataService{}

	for _, char := range chars {
		if char.UUID().Equal(TempSetpointUUID) {
			dataService.tempSetpoint = char
			continue
		}
		if char.UUID().Equal(BoostTempUUID) {
			dataService.boostTemp = char
			continue

		}
		if char.UUID().Equal(CurrentTempUUID) {
			dataService.currentTemp = char
			continue

		}
		if char.UUID().Equal(BatteryLevelUUID) {
			dataService.batteryLevel = char
			continue
		}
		if char.UUID().Equal(LedUUID) {
			dataService.led = char
			continue
		}
	}

	return &dataService, nil
}

// ReadDataServiceCharacteristics reads values from data service and returns a pointer to CraftyStatus struct
func ReadDataServiceCharacteristics(p gatt.Peripheral, ds *DataService) (*CraftyStatus, error) {
	craftyStatus := CraftyStatus{}

	if intValue, err := ReadUint16(p, ds.currentTemp); err == nil {
		craftyStatus.currentTemp = intValue
	} else {
		fmt.Printf("error reading currentTemp: %s", err)
	}

	if intValue, err := ReadUint16(p, ds.boostTemp); err == nil {
		craftyStatus.boostTemp = intValue
	} else {
		fmt.Printf("error reading boostTemp: %s", err)
	}

	if intValue, err := ReadUint16(p, ds.tempSetpoint); err == nil {
		craftyStatus.tempSetpoint = intValue
	} else {
		fmt.Printf("error reading tempSetpoint: %s", err)
	}

	if intValue, err := ReadUint16(p, ds.batteryLevel); err == nil {
		craftyStatus.batteryLevel = intValue
	} else {
		fmt.Printf("error reading batteryLevel: %s", err)
	}

	if intValue, err := ReadUint16(p, ds.led); err == nil {
		craftyStatus.led = intValue
	} else {
		fmt.Printf("error reading led: %s", err)
	}

	return &craftyStatus, nil
}
