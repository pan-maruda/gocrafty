// +build

package main

import (
	"fmt"
	"log"

	"encoding/binary"

	"github.com/paypal/gatt"
	"github.com/paypal/gatt/examples/option"
)

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
		fmt.Println("  Local Name        =", a.LocalName)
		fmt.Println("  TX Power Level    =", a.TxPowerLevel)
		fmt.Println("  Manufacturer Data =", a.ManufacturerData)
		fmt.Println("  Service Data      =", a.ServiceData)

		fmt.Printf("Connecting to %s\n", p.ID())
		p.Device().Connect(p)
	}

}

func onPeriphConnectied(p gatt.Peripheral, err error) {
	defer p.Device().CancelConnection(p)

	log.Println("Connecting to Crafty data service")
	uuid := gatt.MustParseUUID("00000001-4c45-4b43-4942-265a524f5453")

	services, err := p.DiscoverServices([]gatt.UUID{uuid})
	if err != nil {
		log.Fatalf("Failed to discover services, err :%s\n", err)
		return
	}

	for _, svc := range services {
		if !svc.UUID().Equal(uuid) {
			continue
		}

		fmt.Printf("	Service UUID: %s\n", svc.UUID())
		fmt.Printf("	Characteristics:\n")
		chars, err := p.DiscoverCharacteristics(nil, svc)

		if err != nil {
			log.Fatalf("Failed to discover characteristics, err :%s\n", err)
			return
		}

		for _, char := range chars {
			fmt.Printf("		UUID: %s, props: %s", char.UUID(), char.Properties())
			if (char.Properties() & gatt.CharRead) != 0 {

				value, err := p.ReadCharacteristic(char)
				if err != nil {
					fmt.Printf("	Failed to read, err: %s\n")
				} else {

					fmt.Printf("	value: %x | %q", value, value)
					if len(value) == 2 {
						int_value := binary.LittleEndian.Uint16(value[0:])
						fmt.Printf("\t| uint16 LE: %d", int_value)
					}
				}
			}
			print("\n")

		}
	}
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
