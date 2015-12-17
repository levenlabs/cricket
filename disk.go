package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

func disk() ([]map[string]interface{}, error) {
	out, err := exec.Command("/usr/bin/df").CombinedOutput()
	if err != nil {
		return nil, err
	}

	var ret []map[string]interface{}
	buf := bufio.NewReader(bytes.NewBuffer(out))
	for {
		l, err := buf.ReadString('\n')
		if err != nil {
			break
		} else if len(l) >= 11 && l[:11] == "Filesystem " {
			continue
		}

		var fs, mountPoint string
		var blocksTotal, blocksUsed, blocksAvail int
		_, err = fmt.Sscanf(l, "%s%d%d%d%s%s", &fs, &blocksTotal, &blocksUsed, &blocksAvail, new(string), &mountPoint)
		if err != nil {
			return nil, err
		}

		if len(fs) < 5 || fs[:5] != "/dev/" {
			continue
		}

		ret = append(ret, map[string]interface{}{
			"fs":         fs,
			"bytesTotal": blocksTotal * 1024,
			"bytesUsed":  blocksUsed * 1024,
			"bytesAvail": blocksAvail * 1024,
			"mountPoint": mountPoint,
		})

	}
	return ret, nil
}

func inDevList(d string, dd []string) bool {
	for i := range dd {
		if d == dd[i] {
			return true
		}
	}
	return false
}

// returns new full set of subInfoSet, and diff from the last. diff may be nil
// if it should not be used for some reason
func diskDiff(prev subInfoSet, devices []string) (subInfoSet, subInfoSet, error) {
	f, err := os.Open("/proc/diskstats")
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	next := subInfoSet{}

	buf := bufio.NewReader(f)
	for {
		l, err := buf.ReadString('\n')
		if err != nil {
			break
		}

		var dev string
		var rCom, rMer, rSec, rMill uint64
		var wCom, wMer, wSec, wMill uint64
		var ioCount, ioMill, weightedIOMill uint64
		if _, err = fmt.Sscanf(
			l, "%d%d%s%d%d%d%d%d%d%d%d%d%d%d",
			new(int), new(int), &dev,
			&rCom, &rMer, &rSec, &rMill,
			&wCom, &wMer, &wSec, &wMill,
			&ioCount, &ioMill, &weightedIOMill,
		); err != nil {
			return nil, nil, err
		}

		dev = "/dev/" + dev
		if !inDevList(dev, devices) {
			continue
		}

		next[dev] = subInfo{
			"readsCompleted":  rCom,
			"readsMerged":     rMer,
			"readSectors":     rSec,
			"readMillis":      rMill,
			"writesCompleted": wCom,
			"writesMerged":    wMer,
			"writtenSectors":  wSec,
			"writeMillis":     wMill,

			// This is the only field which isn't an accumulation, so it can't
			// be subtracted from the previous value. We'd have to do something
			// special to handle it
			//"iosInProgress": ioCount,

			"ioMillis":         ioMill,
			"weightedIOMillis": weightedIOMill,
		}
	}

	return next, next.sub(prev), nil
}
