package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"

	llog "github.com/levenlabs/go-llog"
)

func mem() (map[string]interface{}, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	m := map[string]interface{}{}
	buf := bufio.NewReader(f)
	for {
		l, err := buf.ReadString('\n')
		if err != nil {
			break
		}

		var typ, val string
		if _, err := fmt.Sscanf(l, "%s%s", &typ, &val); err != nil {
			return nil, err
		}

		typ = typ[:len(typ)-1]
		m[typ] = val
	}

	neededFields := map[string]bool{
		"MemTotal":     true,
		"MemAvailable": true,
	}
	for f := range neededFields {
		if _, ok := m[f]; !ok {
			return nil, fmt.Errorf("invalid meminfo, no %q field", f)
		}
	}

	parseInt := func(s string) int {
		i, err := strconv.Atoi(s)
		if err != nil {
			llog.Fatal("could not parse int string", llog.KV{"str": s}, llog.ErrKV(err))
		}
		return i
	}

	mtot := parseInt(m["MemTotal"].(string))
	mavail := parseInt(m["MemAvailable"].(string))
	usedPer := float64(mtot-mavail) / float64(mtot)

	return map[string]interface{}{
		"memTotalKB": mtot,
		"memAvailKB": mavail,
		"memUsedKB":  mtot - mavail,
		"memUsedPer": int(usedPer * 100),
	}, nil

}
