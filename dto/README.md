Defines the overlay for the LED board. Build and deploy with:

```
dtc -O dtb -o custom-pins.dtbo -b 0 -@ custom-pins.dts
sudo cp custom-pins.dtbo /lib/firmware/
```

Ensure the uEnv.txt for the board includes a reference to the compiled overlay:

```
# /boot/uEnv.txt

# Disable the universal cape manager to ensure no conflicts
#enable_uboot_cape_universal=1

uboot_overlay_addr4=/lib/firmware/custom-pins.dtbo
```
