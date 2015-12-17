package main

import (
	"bufio"
	"fmt"
	"os"
)

// returns new set of subInfo, and diff between the last. diff may be nil if it
// should not be used for some reason
func cpuDiff(prev subInfo) (subInfo, subInfo, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	var cpuLine string
	buf := bufio.NewReader(f)
	for {
		if cpuLine, err = buf.ReadString('\n'); err != nil {
			return nil, nil, err
		}
		if len(cpuLine) >= 4 && cpuLine[:4] == "cpu " {
			break
		}
	}

	var user, nice, system, idle, iowait, irq, softirq uint64
	if _, err := fmt.Sscanf(
		cpuLine,
		"%s%d%d%d%d%d%d%d",
		new(string), &user, &nice, &system, &idle, &iowait, &irq, &softirq,
	); err != nil {
		return nil, nil, err
	}

	next := subInfo{
		"cpuUser":    user,
		"cpuNice":    nice,
		"cpuSystem":  system,
		"cpuIdle":    idle,
		"cpuIOWait":  iowait,
		"cpuIRQ":     irq,
		"cpuSoftIRQ": softirq,
	}

	return next, next.sub(prev), nil
}
