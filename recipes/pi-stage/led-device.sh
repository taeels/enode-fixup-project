#!/bin/sh
set -eu

# 원격 sh의 표준입력으로 전달한다. Pi 파일 설치와 sudo가 필요 없다.
action=${1:-status}
seconds=${2:-10}
case "$action" in heartbeat|on|off|restore|status) ;; *) exit 2 ;; esac
case "$seconds" in ''|*[!0-9]*) exit 2 ;; esac
[ "$seconds" -ge 1 ] && [ "$seconds" -le 60 ] || exit 2
for led in ACT PWR; do
    [ -r "/sys/class/leds/$led/brightness" ] || { echo "Missing LED: $led" >&2; exit 1; }
    if [ "$action" != status ]; then
        [ -w "/sys/class/leds/$led/brightness" ] && [ -w "/sys/class/leds/$led/trigger" ] || {
            echo "LED permission missing: $led; apply 99-led-permissions.rules" >&2
            exit 1
        }
    fi
done
status() {
    for led in ACT PWR; do
        trigger=$(sed -n 's/.*\[\(.*\)\].*/\1/p' "/sys/class/leds/$led/trigger")
        printf '%s trigger=%s brightness=%s\n' "$led" "$trigger" "$(cat "/sys/class/leds/$led/brightness")"
    done
}
restore() {
    echo mmc0 > /sys/class/leds/ACT/trigger
    echo input > /sys/class/leds/PWR/trigger
}
manual() {
    for led in ACT PWR; do echo none > "/sys/class/leds/$led/trigger"; done
}
set_leds() {
    for led in ACT PWR; do
        echo "$1" > "/sys/class/leds/$led/brightness"
        actual=$(cat "/sys/class/leds/$led/brightness")
        if { [ "$1" = 0 ] && [ "$actual" != 0 ]; } || { [ "$1" = 1 ] && [ "$actual" = 0 ]; }; then
            echo "LED readback mismatch: $led requested=$1 actual=$actual" >&2
            return 1
        fi
    done
}
case "$action" in
    status) status ;;
    restore) restore; status ;;
    on|off)
        trap restore EXIT
        manual
        if [ "$action" = on ]; then set_leds 1; else set_leds 0; fi
        status
        trap - EXIT
        ;;
    heartbeat)
        trap restore EXIT
        trap 'exit 130' INT
        trap 'exit 143' TERM HUP
        manual
        cycles=$((seconds * 2))
        while [ "$cycles" -gt 0 ]; do
            set_leds 1
            sleep 0.25
            set_leds 0
            sleep 0.25
            cycles=$((cycles - 1))
        done
        echo "Heartbeat completed: ACT and PWR, 2 Hz, ${seconds}s; readback verified"
        ;;
esac
