# Vulpix

## Setup

1. Create app bundle
    ```bash
    go build -o vulpix .
    mkdir -p vulpix.app/Contents/MacOS
    cp vulpix vulpix.app/Contents/MacOS/vulpix
    ```

2. Create Info.plist in Contents dir (mac only, not needed for windows and linux (needs BlueZ running))
    ```xml
    <?xml version="1.0" encoding="UTF-8"?>
    <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
    <plist version="1.0">
    <dict>
            <key>CFBundleExecutable</key><string>vulpix</string>
            <key>CFBundleIdentifier</key><string>com.example.vulpix</string>
            <key>CFBundleName</key><string>vulpix</string>
            <key>CFBundlePackageType</key><string>APPL</string>
            <key>CFBundleVersion</key><string>1</string>
            <key>NSBluetoothAlwaysUsageDescription</key>
            <string>Scans for nearby BLE devices to track signal strength.</string>
    </dict>
    </plist>
    ```

3. To capture RSSI of devices
    ```bash
    ./vulpix.app/Contents/MacOS/vulpix
    ```
    or
    ```bash
    # Start
    open --stdout /tmp/vulpix.log --stderr /tmp/vulpix.log vulpix.app
    # Monitor
    tail -f /tmp/vulpix.log
    # Kill
    pkill -f "vulpix.app/Contents/MacOS/vulpix"
    ```