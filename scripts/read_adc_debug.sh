#!/bin/bash
# Read ADC debug information from PRU shared memory

echo "PRU ADC Debug Monitor"
echo "===================="
echo ""

if [ "$EUID" -ne 0 ]; then
    echo "ERROR: Must run as root"
    echo "Usage: sudo $0"
    exit 1
fi

echo "Reading debug info from PRU shared memory..."
echo "Press Ctrl+C to stop"
echo ""

while true; do
    echo -n "[$(date '+%H:%M:%S')] "

    # Read 8 uint32 values from control offset (0x4A312000)
    values=$(dd if=/dev/mem bs=4 skip=$((0x4A312000/4)) count=8 2>/dev/null | od -An -t u4 -w4)

    # Parse values
    adc_ctrl=$(echo "$values" | sed -n '1p' | xargs)
    adc_stepenable=$(echo "$values" | sed -n '2p' | xargs)
    adc_fifo0count=$(echo "$values" | sed -n '3p' | xargs)
    adc_fifo0data=$(echo "$values" | sed -n '4p' | xargs)
    attempts=$(echo "$values" | sed -n '5p' | xargs)
    successful=$(echo "$values" | sed -n '6p' | xargs)
    timeouts=$(echo "$values" | sed -n '7p' | xargs)
    last_sample=$(echo "$values" | sed -n '8p' | xargs)

    # Calculate voltage (ADC is 12-bit, 0-1.8V)
    if [ ! -z "$last_sample" ] && [ "$last_sample" != "0" ]; then
        voltage=$(echo "scale=3; $last_sample * 1.8 / 4095" | bc)
    else
        voltage="0.000"
    fi

    # Print status
    echo "ADC_CTRL=0x$(printf '%08X' $adc_ctrl) STEPENABLE=0x$(printf '%08X' $adc_stepenable) FIFO_CNT=$adc_fifo0count | Attempts=$attempts OK=$successful Timeout=$timeouts | Sample=$last_sample (${voltage}V)"

    sleep 1
done
