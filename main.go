package main

import (
	"strconv"
	"time"

	"github.com/levenlabs/go-llog"
	"github.com/mediocregopher/lever"
)

type subInfo map[string]uint64

func (c subInfo) sub(c2 subInfo) subInfo {
	if c == nil {
		return subInfo{}
	} else if c2 == nil {
		c2 = subInfo{}
	}
	out := subInfo{}
	for k := range c {
		// If the new value is less than the old it means we've overflowed the
		// unsigned int and are starting over. In this case we just skip this
		// diff entirely, and will print the next time around
		if c[k] < c2[k] {
			return nil
		}
		out[k] = c[k] - c2[k]
	}
	return out
}

func (c subInfo) toKV() llog.KV {
	kv := llog.KV{}
	for k, v := range c {
		kv[k] = v
	}
	return kv
}

type subInfoSet map[string]subInfo

func (s subInfoSet) sub(s2 subInfoSet) subInfoSet {
	diff := subInfoSet{}
	for k, si := range s {
		si2, ok := s2[k]
		if !ok {
			continue
		}
		siDiff := si.sub(si2)
		// If any of the contained diffs should be skipped, we just skip the
		// whole lot of them, for consistency
		if siDiff == nil {
			return nil
		}
		diff[k] = siDiff
	}
	return diff
}

func mustParseDuration(s string) (time.Duration, bool) {
	if s == "" {
		return 0, false
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		llog.Fatal("could not parse duration string", llog.KV{
			"str": s,
			"err": err,
		})
	}
	return d, d != 0
}

func mustParseInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		llog.Fatal("could not parse int string", llog.KV{
			"str": s,
			"err": err,
		})
	}
	return i
}

func main() {
	l := lever.New("cricket", nil)
	l.Add(lever.Param{
		Name:        "--cpu-interval",
		Description: "interval at which to display cpu stats. Blank to not show",
		Default:     "1s",
	})
	l.Add(lever.Param{
		Name:        "--mem-interval",
		Description: "interval at which to display memory stats. Blank to not show",
		Default:     "1s",
	})
	l.Add(lever.Param{
		Name:        "--disk-interval",
		Description: "interval at which to display disk stats. Blank to not show",
		Default:     "1s",
	})
	l.Add(lever.Param{
		Name:        "--net-interval",
		Description: "interval at which to display net stats. Blank to not show",
		Default:     "1s",
	})
	l.Parse()

	i, _ := l.ParamStr("--cpu-interval")
	if d, ok := mustParseDuration(i); ok {
		go cpuLoop(d)
	}

	i, _ = l.ParamStr("--mem-interval")
	if d, ok := mustParseDuration(i); ok {
		go memLoop(d)
	}

	i, _ = l.ParamStr("--disk-interval")
	if d, ok := mustParseDuration(i); ok {
		go diskLoop(d)
	}

	i, _ = l.ParamStr("--net-interval")
	if d, ok := mustParseDuration(i); ok {
		go netLoop(d)
	}

	select {}
}

func cpuLoop(interval time.Duration) {
	var full, diff subInfo
	var err error
	var doPrint bool
	for tick := time.Tick(interval); ; {
		if full, diff, err = cpuDiff(full); err != nil {
			llog.Fatal("err calling cpuDiff", llog.KV{"err": err})
		}

		if doPrint && diff != nil {
			llog.Info("cpu stats (diff)", diff.toKV())
		}
		doPrint = true
		<-tick
	}
}

func memLoop(interval time.Duration) {
	for tick := time.Tick(interval); ; {
		m, err := mem()
		if err != nil {
			llog.Fatal("err calling mem", llog.KV{"err": err})
		}

		llog.Info("mem stats", llog.KV(m))
		<-tick
	}
}

func diskLoop(interval time.Duration) {
	var full, diff subInfoSet
	for tick := time.Tick(interval); ; {
		dd, err := disk()
		if err != nil {
			llog.Fatal("err calling disk", llog.KV{"err": err})
		}

		devs := make([]string, 0, len(dd))
		for _, d := range dd {
			devs = append(devs, d["fs"].(string))
			llog.Info("disk usage stats", llog.KV(d))
		}

		if full, diff, err = diskDiff(full, devs); err != nil {
			llog.Fatal("err calling diskDiff", llog.KV{"err": err})
		}

		for dev, d := range diff {
			kv := d.toKV()
			kv["fs"] = dev
			llog.Info("disk io stats (diff)", kv)
		}

		<-tick
	}
}

func netLoop(interval time.Duration) {
	var full, diff subInfoSet
	var err error
	for tick := time.Tick(interval); ; {
		if full, diff, err = netDiff(full); err != nil {
			llog.Fatal("err calling netDiff", llog.KV{"err": err})
		}

		for dev, d := range diff {
			kv := d.toKV()
			kv["dev"] = dev
			llog.Info("net stats (diff)", kv)
		}

		<-tick
	}
}
