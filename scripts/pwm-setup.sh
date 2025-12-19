#!/bin/bash
# PWM Setup Script - Export and enable PWM channels

# Function to setup a PWM channel
setup_pwm() {
    local chip=$1
    local channel=$2
    local pwm_path="/sys/class/pwm/pwmchip${chip}"
    local channel_path="${pwm_path}/pwm${channel}"
    
    # Export the channel if not already exported
    if [ ! -d "${channel_path}" ]; then
        echo ${channel} > ${pwm_path}/export
        sleep 0.1
    fi
    
    # Set period to 20 kHz (50000ns = 50us)
    # Only set if period is 0
    current_period=$(cat ${channel_path}/period 2>/dev/null || echo "0")
    if [ "$current_period" -eq 0 ]; then
        echo 50000 > ${channel_path}/period
    fi
    
    # Set duty cycle to 0 (off by default)
    echo 0 > ${channel_path}/duty_cycle
    
    # Enable the PWM channel
    echo 1 > ${channel_path}/enable
}

# Wait for PWM devices to be ready
sleep 2

# Setup PWM channels
# pwmchip2: ehrpwm2 (P8_19, P8_13)
setup_pwm 2 0  # P8_19
setup_pwm 2 1  # P8_13

# pwmchip4: ehrpwm0 or other PWM controller
setup_pwm 4 0
setup_pwm 4 1

echo "PWM channels exported and enabled"
