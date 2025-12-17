# Makefile for PRU I2S Monitor
TARGET := pru0_i2s_monitor
SOURCE := $(TARGET).c

INCLUDE := --include_path=. \
           --include_path=/usr/share/ti/cgt-pru/include \
           --include_path=/usr/share/ti/cgt-pru/lib \
           --include_path=/usr/lib/ti/pru-software-support-package-v6.3/include \
           --include_path=/usr/lib/ti/pru-software-support-package-v6.3/include/am335x

CFLAGS := -v3 -O2 --display_error_number --endian=little --hardware_mac=on \
          --obj_directory=gen

LIB_INCLUDE := -i/usr/share/ti/cgt-pru/lib -i/usr/share/ti/cgt-pru/include

.PHONY: all clean

all: gen/$(TARGET).out

gen/$(TARGET).out: $(SOURCE)
	@mkdir -p gen
	clpru $(INCLUDE) $(CFLAGS) --output_file=gen/$(TARGET).out \
	      $(SOURCE) \
	      -z am335x_pru.cmd $(LIB_INCLUDE) -llibc.a
	@echo "✓ Built gen/$(TARGET).out"

clean:
	rm -rf gen

install: gen/$(TARGET).out
	sudo cp gen/$(TARGET).out /lib/firmware/am335x-pru0-fw
	@echo "✓ Installed to /lib/firmware/am335x-pru0-fw"
