package main

import (
	"C"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/paypal/gatt"
	"github.com/paypal/gatt/examples/option"
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

func onStateChanged(d gatt.Device, s gatt.State) {
	fmt.Println("State:", s)
	switch s {
	case gatt.StatePoweredOn:
		fmt.Println("scanning...")
		d.Scan([]gatt.UUID{}, false)
		return
	default:
		d.StopScanning()
	}
}

func onPeriphDiscovered(p gatt.Peripheral, a *gatt.Advertisement, rssi int) {
	// TODO add selection list or mac input, not connect to the first one
	if a.LocalName == "STORZ&BICKEL" {
		fmt.Printf("\nPeripheral ID:%s, NAME:(%s)\n", p.ID(), p.Name())
		fmt.Printf("Connecting to %s\n", p.ID())
		p.Device().Connect(p)
	}

}

func onPeriphConnectied(p gatt.Peripheral, err error) {
	defer p.Device().CancelConnection(p)

	log.Println("Discovering Crafty services")

	services, err := p.DiscoverServices([]gatt.UUID{DataServiceUUID})
	if err != nil {
		log.Fatalf("Failed to discover services, err :%s\n", err)
		return
	}

	for _, svc := range services {

		if svc.UUID().Equal(DataServiceUUID) {
			ReadDataServiceCharacteristics(p, svc)
		}

		if svc.UUID().Equal(MetaServiceUUID) {
			// ReadMetadataService(p, svc)
		}

	}
}

func ReadDataServiceCharacteristics(p gatt.Peripheral, svc *gatt.Service) {
	chars, err := p.DiscoverCharacteristics([]gatt.UUID{TempSetpointUUID, CurrentTempUUID, BoostTempUUID, BatteryLevelUUID}, svc)

	if err != nil {
		log.Fatalf("Failed to discover characteristics, err :%s\n", err)
		return
	}

	for _, char := range chars {

		if char.UUID().Equal(TempSetpointUUID) {
			fmt.Printf("Temperature setpoint: ")
			value, err := ReadUint16(p, char)
			if err == nil {
				fmt.Printf("%d C", value/10)
			} else {
				fmt.Printf(" read failed: %s", err)
			}
		}

		if char.UUID().Equal(BoostTempUUID) {
			fmt.Printf("Boost temp: ")
			value, err := ReadUint16(p, char)
			if err == nil {
				fmt.Printf("+%d C", value/10)
			} else {
				fmt.Printf(" read failed: %s", err)
			}
		}

		if char.UUID().Equal(CurrentTempUUID) {
			fmt.Printf("Current temperature: ")
			value, err := ReadUint16(p, char)
			if err == nil {
				fmt.Printf("%d C", value/10)
			} else {
				fmt.Printf(" read failed: %s", err)
			}
		}

		if char.UUID().Equal(BatteryLevelUUID) {
			fmt.Printf("Battery level: ")
			value, err := ReadUint16(p, char)
			if err == nil {
				fmt.Printf("%d%%", value)
			} else {
				fmt.Printf(" read failed: %s", err)
			}
		}

		print("\n")
	}
}

func ReadMetadataService(p gatt.Peripheral, svc *gatt.Service) {
	chars, err := p.DiscoverCharacteristics([]gatt.UUID{ModelUUID, VersionUUID, SerialUUID}, svc)

	if err != nil {
		log.Fatalf("Failed to discover characteristics, err :%s\n", err)
		return
	}

	for _, char := range chars {
		if char.UUID().Equal(ModelUUID) {
			fmt.Printf("Model name: ")
		}
		if char.UUID().Equal(VersionUUID) {
			fmt.Printf("Version number: ")
		}
		if char.UUID().Equal(SerialUUID) {
			fmt.Printf("Serial number:")
		}
		value, err := ReadString(p, char)
		if err == nil {
			print(value)
		} else {
			fmt.Printf(" read failed: %s", err)
		}
		print("\n")
	}
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
	return 0, fmt.Errorf("characteristic %s read != 2 bytes: %x", c.Name(), value)
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

func main() {
	d, err := gatt.NewDevice(option.DefaultClientOptions...)
	if err != nil {
		log.Fatalf("Failed to open device, err: %s\n", err)
		return
	}

	// Register handlers.
	d.Handle(gatt.PeripheralDiscovered(onPeriphDiscovered),
		gatt.PeripheralConnected(onPeriphConnectied))
	d.Init(onStateChanged)
	select {}
}
