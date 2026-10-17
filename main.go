package main

import (
	"cmp"
	"fmt"
	"log"
	"slices"
	"sync"
	"time"

	"tinygo.org/x/bluetooth"
)

const (
	renderInterval = 1 * time.Second
	deviceTimeout  = 10 * time.Second

	// Column widths, shared by the header and the rows so they cannot drift.
	addrWidth = 38 // macOS reports a 36-character UUID here
	nameWidth = 24
	rssiWidth = 8
)

type device struct {
	Address  string
	Name     string
	RSSI     int16
	LastSeen time.Time
}

func main() {
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

			// Preserve the previous name if this advertisement
			// doesn't include one.
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

		render(snapshot)
	}
}

func render(devices []device) {
	// Cursor home, then clear to end of screen: the frame repaints in place
	// instead of scrolling the terminal.
	fmt.Print("\033[H\033[J")

	fmt.Printf("--- BLE devices (%d) ---\n", len(devices))
	fmt.Printf("%-*s %-*s %*s  %s\n",
		addrWidth, "ADDRESS",
		nameWidth, "NAME",
		rssiWidth, "RSSI",
		"LAST SEEN",
	)

	for _, dev := range devices {
		name := dev.Name
		if name == "" {
			name = "(unknown)"
		}

		fmt.Printf("%-*s %-*s %*s  %v ago\n",
			addrWidth, dev.Address,
			nameWidth, name,
			rssiWidth, fmt.Sprintf("%d dBm", dev.RSSI),
			time.Since(dev.LastSeen).Round(time.Millisecond),
		)
	}
}
