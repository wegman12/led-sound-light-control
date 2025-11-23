# PRU Setup for BeagleBone Black High-Speed ADC Sampling

This guide will help you set up the PRU (Programmable Real-time Unit) on your BeagleBone Black for 48 kHz ADC sampling.

## Prerequisites

- BeagleBone Black with Debian-based Linux
- Root access
- Internet connection for downloading tools

## Step 1: Install PRU Development Tools

```bash
# Update package list
sudo apt-get update

# Install PRU compiler and tools
sudo apt-get install -y ti-pru-cgt-installer

# Or if using the open-source GCC-based toolchain:
sudo apt-get install -y gcc-pru

# Install PRU software support package
sudo apt-get install -y am335x-pru-package

# Install device tree compiler
sudo apt-get install -y device-tree-compiler
```

## Step 2: Enable PRU Remoteproc

Edit `/boot/uEnv.txt` and ensure the PRU remoteproc is enabled:

```bash
sudo nano /boot/uEnv.txt
```

Add or uncomment this line:
```
uboot_overlay_pru=/lib/firmware/AM335X-PRU-RPROC-4-19-TI-00A0.dtbo
```

## Step 3: Install Device Tree Overlay

```bash
# Copy the device tree overlay
sudo cp pru/PRU-ADC-00A0.dts /lib/firmware/

# Compile the device tree overlay
cd /lib/firmware
sudo dtc -O dtb -o PRU-ADC-00A0.dtbo -b 0 -@ PRU-ADC-00A0.dts

# Enable the overlay at boot by adding to /boot/uEnv.txt:
# dtb_overlay=/lib/firmware/PRU-ADC-00A0.dtbo
```

## Step 4: Build PRU Firmware

```bash
cd pru
make
```

This will create `pru0_adc_sampler.out`.

## Step 5: Load PRU Firmware

```bash
# Copy firmware to PRU firmware directory
sudo cp gen/pru0_adc_sampler.out /lib/firmware/am335x-pru0-fw

# Reboot or reload PRU
sudo reboot

# Or manually reload PRU:
echo 'stop' | sudo tee /sys/class/remoteproc/remoteproc1/state
echo 'start' | sudo tee /sys/class/remoteproc/remoteproc1/state
```

## Step 6: Verify PRU is Running

```bash
# Check PRU status
cat /sys/class/remoteproc/remoteproc1/state
# Should show "running"

# Check kernel messages
dmesg | grep pru
```

## Memory Map

- **PRU Shared Memory**: 0x4A310000 (12 KB shared between PRU0 and PRU1)
- **Sample Buffer**: First 8 KB of shared memory
- **Control Structure**: Offset 0x2000 in shared memory

## Sample Buffer Structure

```
Offset 0x0000: Ring buffer (8192 bytes = 4096 samples)
Offset 0x2000: Control structure:
  - uint32_t write_index (current write position)
  - uint32_t read_index (current read position)
  - uint32_t sample_count (total samples written)
  - uint32_t overrun_count (buffer overruns)
```

Each ADC sample is a 16-bit value (0-4095 for 12-bit ADC).

## Troubleshooting

### PRU won't start
- Check `dmesg` for errors
- Verify device tree overlay is loaded: `ls /sys/devices/platform/ocp/`
- Check PRU remoteproc: `ls /sys/class/remoteproc/`

### No ADC readings
- Verify analog input is connected to P9_40 (AIN1)
- Check voltage is between 0-1.8V (BeagleBone ADC limit)
- Check PRU shared memory: `sudo dd if=/dev/mem bs=1 skip=$((0x4A310000)) count=16 | hexdump -C`

### Buffer overruns
- Increase read frequency in Go code
- Reduce sample rate in PRU firmware
- Increase buffer size
