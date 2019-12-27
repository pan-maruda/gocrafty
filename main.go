package main

import (
	"fmt"
	"log"
	"os"

	"github.com/pan-maruda/gocrafty/ble"
	"github.com/paypal/gatt"
	"github.com/paypal/gatt/examples/option"
)

func onStateChanged(d gatt.Device, s gatt.State) {
	fmt.Println("State:", s)
	switch s {
	case gatt.StatePoweredOn:
		fmt.Println("scanning...")
		d.Scan([]gatt.UUID{ble.DataServiceUUID, ble.MetaServiceUUID}, false)
		return
	default:
		d.StopScanning()
	}
}

func onPeriphDiscovered(craftyID string) func(gatt.Peripheral, *gatt.Advertisement, int) {

	return func(p gatt.Peripheral, a *gatt.Advertisement, rssi int) {
		// TODO add selection list or mac input, not connect to the first one
		if p.ID() == craftyID {
			fmt.Printf("Connecting to %s [%s]\n", p.ID(), p.Name())
			p.Device().Connect(p)
		}
	}

}

func onPeriphConnected(craftyID string) func(gatt.Peripheral, error) {

	return func(p gatt.Peripheral, err error) {
		if p.ID() != craftyID {
			fmt.Printf("Unexpected device ID [%s] connected instead of [%s]. WTF?", p.ID(), craftyID)
			return
		}

		p.Device().StopScanning()

		defer p.Device().CancelConnection(p)
		defer fmt.Printf("Disconnected from %s", p.ID())
		defer p.Device().Scan([]gatt.UUID{ble.DataServiceUUID, ble.MetaServiceUUID}, false)
		log.Println("Discovering Crafty services")

		services, err := p.DiscoverServices([]gatt.UUID{ble.DataServiceUUID, ble.MetaServiceUUID})
		if err != nil {
			log.Fatalf("Failed to discover services, err :%s\n", err)
			return
		}

		for _, svc := range services {

			if svc.UUID().Equal(ble.DataServiceUUID) {
				data, err := ble.ReadDataServiceCharacteristics(p, svc)
				if err != nil {
					log.Printf("Failed to read metadata from Crafty: %s\n", err)
					continue
				}
				fmt.Println(data)
			}

			if svc.UUID().Equal(ble.MetaServiceUUID) {
				meta, err := ble.ReadMetadataService(p, svc)
				if err != nil {
					log.Printf("Failed to read metadata from Crafty: %s\n", err)
					continue
				}
				fmt.Println(meta)

			}

		}
	}
}

func main() {
	done := make(chan bool, 1)
	craftyID, found := os.LookupEnv("CRAFTY_ID")
	if !found {
		fmt.Println("CRAFTY_ID not set, use ./scanner to find your Crafty.")
		os.Exit(1)
	}

	d, err := gatt.NewDevice(option.DefaultClientOptions...)
	if err != nil {
		log.Fatalf("Failed to open device, err: %s\n", err)
		return
	}

	// Register handlers.
	d.Handle(gatt.PeripheralDiscovered(onPeriphDiscovered(craftyID)),
		gatt.PeripheralConnected(onPeriphConnected))
	d.Init(onStateChanged)
	select {
	case <-done:
		os.Exit(0)
	}
}
