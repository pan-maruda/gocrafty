package main

import (
	"flag"
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

func onPeriphDiscovered(craftyID string, done <-chan bool) func(gatt.Peripheral, *gatt.Advertisement, int) {

	return func(p gatt.Peripheral, a *gatt.Advertisement, rssi int) {
		// TODO add selection list or mac input, not connect to the first one
		if p.ID() == craftyID {
			fmt.Printf("Connecting to %s [%s]\n", p.ID(), p.Name())
			p.Device().Connect(p)
		}
	}

}

func onPeriphConnected(craftyID string, done chan bool, action func(gatt.Peripheral, *ble.DataService)) func(gatt.Peripheral, error) {

	return func(p gatt.Peripheral, err error) {

		if p.ID() != craftyID {
			fmt.Printf("Unexpected device ID [%s] connected instead of [%s]. WTF?", p.ID(), craftyID)
			return
		}

		p.Device().StopScanning()

		services, err := p.DiscoverServices([]gatt.UUID{ble.DataServiceUUID, ble.MetaServiceUUID})
		if err != nil {
			log.Fatalf("Failed to discover services, err :%s\n", err)
			return
		}

		var dataSvc *ble.DataService
		// var metaSvc *gatt.Service

		for _, svc := range services {

			if svc.UUID().Equal(ble.DataServiceUUID) {
				// dataSvc = svc

				if disc, err := ble.DiscoverDataService(p, svc); err == nil {
					data, err := ble.ReadDataServiceCharacteristics(p, disc)
					if err != nil {
						log.Printf("Failed to read metadata from Crafty: %s\n", err)
						continue
					}
					fmt.Println(data)
					dataSvc = disc
				} else {
					log.Printf("Failed to discover Crafty data service: %s\n", err)
					continue
				}

			}

			if svc.UUID().Equal(ble.MetaServiceUUID) {
				// metaSvc = svc
				meta, err := ble.ReadMetadataService(p, svc)
				if err != nil {
					log.Printf("Failed to read metadata from Crafty: %s\n", err)
					continue
				}
				fmt.Println(meta)
			}
		}

		action(p, dataSvc)
	}
}

func monitor(p gatt.Peripheral, dataSvc *ble.DataService) {
	if dataSvc == nil {
		fmt.Printf("Data service is nil - not discovered? wtf?")
		return
	}
	dataSvc.SubscribeBattery(p, func(batteryLevel uint16, err error) {
		fmt.Printf("Battery level: %d%%\n", batteryLevel)
	})
	dataSvc.SubscribeTemp(p, func(currentTemp uint16, err error) {
		fmt.Printf("Current Temp: %d.%d C\n", currentTemp/10, currentTemp%10)
	})
	select {}
}

func setValues(temp *int, boost *int, done chan bool) func(gatt.Peripheral, *ble.DataService) {
	// todo validate this somewhere else
	return func(p gatt.Peripheral, ds *ble.DataService) {
		if temp != nil && *temp != -1 {
			if *temp > 210 {
				fmt.Println("Temperature cannot exceed 210.")
			}
			if *temp < 0 {
				fmt.Println("Temperature must be positive.")
			}
			fmt.Printf("Setting temperature point to %d\n", *temp)
			ds.SetTemp(p, *temp)
		}

		if boost != nil && *boost != -1 {
			var validBoost int
			if *boost < 0 {
				fmt.Println("Boost must be positive.")
			}
			if *temp+*boost > 210 {
				validBoost = (210 - *temp)
				fmt.Printf("Clamped boost temp to +%d C\n", validBoost)
			} else {
				validBoost = *boost
			}

			if validBoost != 0 {
				fmt.Printf("Setting boost temp to +%d C\n", validBoost)
				ds.SetBoost(p, validBoost)
			}
		}
		done <- true

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

	// var oneshot = flag.Bool("oneshot", false, "Read data only once (no notifications)")
	var tempFlag = flag.Int("set-temp", -1, "set base vape temperature point")
	var boostTempFlag = flag.Int("set-boost", -1, "set boost value (positive only)")
	flag.Parse()
	d.Init(onStateChanged)
	if *tempFlag == -1 && *boostTempFlag == -1 {
		d.Handle(gatt.PeripheralDiscovered(onPeriphDiscovered(craftyID, done)),
			gatt.PeripheralConnected(onPeriphConnected(craftyID, done, monitor)))
	} else {
		d.Handle(gatt.PeripheralDiscovered(onPeriphDiscovered(craftyID, done)),
			gatt.PeripheralConnected(onPeriphConnected(craftyID, done, setValues(tempFlag, boostTempFlag, done))))
	}

	// Register handlers.
	select {
	case <-done:
		d.StopScanning()
		return
	}
}
