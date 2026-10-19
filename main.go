package main

import (
	"cmp"
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"sync"
	"time"

	"tinygo.org/x/bluetooth"
)

const (
	renderInterval = 1 * time.Second
	deviceTimeout  = 10 * time.Second

	// Column widths, shared by the header and the rows so they cannot drift.
	addrWidth      = 38 // macOS reports a 36-character UUID here
	nameWidth      = 24
	rssiWidth      = 8
	proximityWidth = 16

	// Proximity reference points.
	defaultNear = -50
	defaultFar  = -72
)

const (
	modeRaw       = "raw"
	modeCalibrate = "calibrate"
)

type device struct {
	Address  string
	Name     string
	RSSI     int16
	LastSeen time.Time
}

func main() {
	mode := flag.String("mode", modeCalibrate, "raw for RSSI only, calibrate for RSSI plus proximity band")
	near := flag.Int("near", defaultNear, "RSSI at or above which a device reads very close")
	far := flag.Int("far", defaultFar, "RSSI below which a device reads far")
	flag.Parse()

	if err := validateMode(*mode); err != nil {
		fmt.Fprintln(os.Stderr, err)
		flag.Usage()
		os.Exit(2)
	}

	adapter := bluetooth.DefaultAdapter

	if err := adapter.Enable(); err != nil {
		log.Fatal("enable bluetooth: ", err)
	}

	fmt.Println("Scanning... (Ctrl+C to stop)")

	// Shared state between the scan callback and render loop.
	var mu sync.Mutex
	devices := make(map[string]device)

	// The scan runs independently because Scan blocks.
	go func() {
		err := adapter.Scan(func(
			a *bluetooth.Adapter,
			d bluetooth.ScanResult,
		) {
			// A non-negative RSSI is the "no measurement" sentinel, not a
			// reading. Storing it would be useless.
			if d.RSSI >= 0 {
				return
			}

			now := time.Now()
			address := d.Address.String()

			mu.Lock()
			defer mu.Unlock()

			// Preserve the previous name if this advertisement doesn't
			// include one.
			dev := devices[address]
			dev.Address = address
			if name := d.LocalName(); name != "" {
				dev.Name = name
			}
			dev.RSSI = d.RSSI
			dev.LastSeen = now

			devices[address] = dev
		})

		if err != nil {
			log.Printf("scan: %v", err)
		}
	}()

	ticker := time.NewTicker(renderInterval)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()

		mu.Lock()

		// 1. Take a snapshot.
		snapshot := make([]device, 0, len(devices))

		// 2. Evict stale devices
		for address, dev := range devices {
			if now.Sub(dev.LastSeen) > deviceTimeout {
				delete(devices, address)
				continue
			}
			snapshot = append(snapshot, dev)
		}

		mu.Unlock()

		// 3. Sort and render outside the lock.
		slices.SortFunc(snapshot, func(a, b device) int {
			if c := cmp.Compare(b.RSSI, a.RSSI); c != 0 {
				return c
			}
			return cmp.Compare(a.Name, b.Name)
		})

		render(snapshot, *near, *far, *mode)
	}
}

func validateMode(mode string) error {
	switch mode {
	case modeRaw, modeCalibrate:
		return nil
	}
	return fmt.Errorf("unknown mode %q: want %q or %q", mode, modeRaw, modeCalibrate)
}

func render(devices []device, near, far int, mode string) {
	// Cursor home, then clear to end of screen: the frame repaints in place
	// instead of scrolling the terminal.
	fmt.Print("\033[H\033[J")

	fmt.Printf("--- BLE devices (%d) ---\n", len(devices))

	showBand := mode == modeCalibrate

	if showBand {
		fmt.Printf("%-*s %-*s %-*s %s\n",
			addrWidth, "ADDRESS",
			nameWidth, "NAME",
			proximityWidth, "PROXIMITY",
			"LAST SEEN",
		)
	} else {
		fmt.Printf("%-*s %-*s %*s  %s\n",
			addrWidth, "ADDRESS",
			nameWidth, "NAME",
			rssiWidth, "RSSI",
			"LAST SEEN",
		)
	}

	for _, dev := range devices {
		name := dev.Name
		if name == "" {
			name = "(unknown)"
		}

		rssi := fmt.Sprintf("%d dBm", dev.RSSI)
		lastSeen := fmt.Sprintf("%v ago", time.Since(dev.LastSeen).Round(time.Millisecond))

		if showBand {
			fmt.Printf("%-*s %-*s %-*s %s\n",
				addrWidth, dev.Address,
				nameWidth, name,
				proximityWidth, proximity(dev.RSSI, int16(near), int16(far)),
				lastSeen,
			)
		} else {
			fmt.Printf("%-*s %-*s %*s  %s\n",
				addrWidth, dev.Address,
				nameWidth, name,
				rssiWidth, rssi,
				lastSeen,
			)
		}
	}
}

func proximity(rssi, near, far int16) string {
	if rssi >= near {
		return "very close"
	} else if rssi >= far {
		return "same room"
	}
	return "far"
}
