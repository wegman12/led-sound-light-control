#!/bin/bash
#
# Test GPIO 20 IR Sensor
# Monitors GPIO state to verify IR sensor is working
#

GPIO=20
GPIO_PATH="/sys/class/gpio/gpio${GPIO}"

echo "IR Sensor GPIO Test"
echo "==================="
echo "GPIO: $GPIO"
echo ""

# Export GPIO if not already exported
if [ ! -d "$GPIO_PATH" ]; then
    echo "Exporting GPIO $GPIO..."
    echo $GPIO | sudo tee /sys/class/gpio/export > /dev/null
    sleep 0.5
fi

# Set direction to input
echo "in" | sudo tee ${GPIO_PATH}/direction > /dev/null

echo "Monitoring GPIO $GPIO state..."
echo "- Idle state (no IR): should be 1 (HIGH, pulled up)"
echo "- Signal present (IR transmitting): should be 0 (LOW)"
echo ""
echo "Press IR remote button and watch for state changes:"
echo ""

LAST_VALUE=""
COUNT=0

while true; do
    VALUE=$(cat ${GPIO_PATH}/value 2>/dev/null)

    if [ "$VALUE" != "$LAST_VALUE" ]; then
        TIMESTAMP=$(date '+%H:%M:%S.%N' | cut -c1-12)

        if [ "$VALUE" = "0" ]; then
            echo "[$TIMESTAMP] GPIO: 0 (LOW)  <-- IR SIGNAL DETECTED"
        else
            echo "[$TIMESTAMP] GPIO: 1 (HIGH) <-- IDLE"
        fi

        LAST_VALUE=$VALUE
        COUNT=$((COUNT + 1))
    fi

    # Exit after some transitions
    if [ "$COUNT" -ge 100 ]; then
        echo ""
        echo "Test complete. Saw $COUNT state transitions."
        break
    fi

    sleep 0.0001  # 100μs
done
