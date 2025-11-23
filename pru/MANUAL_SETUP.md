# Manual PRU Setup Guide

If the automated setup script doesn't work, follow these manual steps.

## Step 1: Install PRU Compiler

Try these options in order:

### Option A: GCC PRU (Recommended)
```bash
sudo apt-get update
sudo apt-get install gcc-pru
```

### Option B: Download Pre-built GCC PRU
```bash
cd /tmp
wget https://github.com/dinuxbg/gnupru/releases/download/v2021.08/gnupru-2021.08-1-amd64.deb
sudo dpkg -i gnupru-2021.08-1-amd64.deb
```

### Option C: Build from Source
```bash
git clone https://github.com/dinuxbg/gnupru.git
cd gnupru
# Follow build instructions in repository
```

## Step 2: Install Device Tree Compiler

```bash
sudo apt-get install device-tree-compiler
```

## Step 3: Check if PRU is Already Enabled

Most modern BeagleBone images have PRU support built-in:

```bash
# Check if PRU remoteproc exists
ls /sys/class/remoteproc/

# Should see remoteproc1 (or remoteproc2)
# Check PRU state
cat /sys/class/remoteproc/remoteproc1/state
```

If you see "offline" or "running", PRU support is already there!

## Step 4: Build PRU Firmware

```bash
cd pru
make clean
make
```

If you get include errors, the firmware will work without the TI headers. The code is self-contained.

## Step 5: Install Device Tree Overlay

```bash
sudo cp PRU-ADC-00A0.dts /lib/firmware/
cd /lib/firmware
sudo dtc -O dtb -o PRU-ADC-00A0.dtbo -b 0 -@ PRU-ADC-00A0.dts
```

## Step 6: Enable PRU at Boot

Edit `/boot/uEnv.txt`:

```bash
sudo nano /boot/uEnv.txt
```

Add or uncomment:
```
uboot_overlay_pru=/lib/firmware/AM335X-PRU-RPROC-4-19-TI-00A0.dtbo
```

And add:
```
dtb_overlay=/lib/firmware/PRU-ADC-00A0.dtbo
```

## Step 7: Install PRU Firmware

```bash
sudo cp gen/pru0_adc_sampler.out /lib/firmware/am335x-pru0-fw
```

## Step 8: Reboot

```bash
sudo reboot
```

## Step 9: Verify PRU is Running

After reboot:

```bash
cat /sys/class/remoteproc/remoteproc1/state
```

Should show: `running`

If not, manually start it:
```bash
echo 'start' | sudo tee /sys/class/remoteproc/remoteproc1/state
```

## Step 10: Test

```bash
cd /path/to/project
make build
sudo ./bin/led-sound-light-control pru-test
```

## Troubleshooting

### No PRU Compiler Found

If no compiler is available, you can cross-compile on another machine and copy the `.out` file to your BeagleBone.

### Cannot Find remoteproc

Your BeagleBone image might use a different PRU interface. Check:

```bash
ls /sys/devices/platform/ocp/ | grep pru
```

Or:

```bash
dmesg | grep -i pru
```

### Build Errors with Missing Headers

The firmware is designed to work without the full TI PRU support package. If you get errors about missing headers, the build should still work - the essential definitions are in the C file itself.

### PRU Won't Start

Check kernel messages:
```bash
dmesg | grep pru | tail -20
```

Common issues:
- Firmware file wrong location (must be `/lib/firmware/am335x-pru0-fw`)
- Permissions on firmware file (should be readable)
- Device tree overlay not loaded

### Still Need Help?

Check what BeagleBone image you're running:
```bash
cat /etc/dogtag
uname -a
```

Different images (Debian, Ubuntu, older vs newer) have different PRU setups.
