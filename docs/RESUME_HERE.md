# Resume Point: Driver Modification Debugging

**Start Here When Resuming This Work**

---

## TL;DR - Where We Left Off

✅ Modified davinci-mcasp driver with clk_set_rate() support
✅ Built and installed on BeagleBone
✅ Recording runs without errors
❌ Audio data still all zeros
❌ No debug messages appearing in dmesg

**Most likely issue**: Compressed `.ko.xz` module being loaded instead of our modified `.ko`

---

## First Thing to Try

```bash
ssh bbb2.wegman

# Remove the compressed original module (it takes priority)
sudo rm /lib/modules/5.10.168-ti-r83/kernel/sound/soc/ti/snd-soc-davinci-mcasp.ko.xz
sudo depmod -a
sudo reboot

# After reboot, check for our messages
ssh bbb2.wegman "dmesg | grep -i 'functional\|Set system clock\|DEBUG'"
```

If you see "DEBUG: About to get functional clock" → Our module is running! Continue debugging.
If still nothing → See full debugging steps in `CURRENT_STATUS_DRIVER_MOD.md`

---

## Files You Need

**Modified driver source**: `/tmp/davinci-mcasp-fresh.c`
**Full status document**: `docs/CURRENT_STATUS_DRIVER_MOD.md`
**Manual edit guide**: `docs/MANUAL_EDIT_GUIDE.md`

**On BeagleBone**:
- Build dir: `~/mcasp-clkrate-build/`
- Installed module: `/lib/modules/5.10.168-ti-r83/kernel/sound/soc/ti/snd-soc-davinci-mcasp.ko`

---

## Quick Test Commands

```bash
# Rebuild and install (after editing /tmp/davinci-mcasp-fresh.c)
scp /tmp/davinci-mcasp-fresh.c bbb2.wegman:~/mcasp-clkrate-build/davinci-mcasp.c
ssh bbb2.wegman "cd ~/mcasp-clkrate-build && make clean && make"
ssh bbb2.wegman "sudo cp ~/mcasp-clkrate-build/snd-soc-davinci-mcasp.ko /lib/modules/5.10.168-ti-r83/kernel/sound/soc/ti/"
ssh bbb2.wegman "sudo rm /lib/modules/5.10.168-ti-r83/kernel/sound/soc/ti/snd-soc-davinci-mcasp.ko.xz"
ssh bbb2.wegman "sudo depmod -a && sudo reboot"

# Wait and test
sleep 45 && ssh bbb2.wegman "dmesg | grep -i mcasp && arecord -D hw:0,0 -f S32_LE -r 48000 -c 1 -d 2 /tmp/test.wav && hexdump -C /tmp/test.wav | head -20"
```

---

## What Success Looks Like

You'll see in dmesg:
```
davinci-mcasp 48038000.mcasp: Set system clock to 73728000 Hz (target 73728000 Hz)
```

And in hexdump:
```
00000030  a3 f2 01 00 7b e4 00 00  [non-zero audio data...]
```

---

## If Still Stuck After 1 Hour

Consider ordering external 24.576 MHz oscillator ($1-5, 1-2 day shipping):
- Digikey: Abracon ASFLMB-24.576MHZ-LC-T
- Guaranteed to work with no software changes

---

Read **CURRENT_STATUS_DRIVER_MOD.md** for full details, hypotheses, and debugging steps.
