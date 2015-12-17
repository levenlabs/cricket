package main

import (
	"bufio"
	"fmt"
	"os"
)

// returns new full set of subInfoSet, and diff from the last. diff may be nil
// if it should not be used for some reason
func netDiff(prev subInfoSet) (subInfoSet, subInfoSet, error) {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	next := subInfoSet{}
	buf := bufio.NewReader(f)

	// first two lines are garbage
	for i := 0; i < 2; i++ {
		if _, err := buf.ReadString('\n'); err != nil {
			return nil, nil, err
		}
	}

	for {
		l, err := buf.ReadString('\n')
		if err != nil {
			break
		}

		var dev string
		var rBytes, rPackets, rErrs, rDrop uint64
		var wBytes, wPackets, wErrs, wDrop uint64
		skip := new(int)
		if _, err = fmt.Sscanf(
			l, "%s%d%d%d%d%d%d%d%d%d%d%d%d", &dev,
			&rBytes, &rPackets, &rErrs, &rDrop,
			skip, skip, skip, skip,
			&wBytes, &wPackets, &wErrs, &wDrop,
		); err != nil {
			return nil, nil, err
		}

		next[dev[:len(dev)-1]] = subInfo{
			"rcvBytes":   rBytes,
			"rcvPackets": rPackets,
			"rcvErrs":    rErrs,
			"rcvDrop":    rDrop,
			"txBytes":    wBytes,
			"txPackets":  wPackets,
			"txErrs":     wErrs,
			"txDrop":     wDrop,
		}
	}

	return next, next.sub(prev), nil
}
