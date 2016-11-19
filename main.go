package main

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/levenlabs/go-llog"
	ping "github.com/levenlabs/go-ping"
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

// a count of zero or less can be given for an unlimited ticker, which is
// effectively like calling time.Tick
func limitedTicker(c int, interval time.Duration) <-chan time.Time {
	t := time.Tick(interval)
	ch := make(chan time.Time)
	go func() {
		ch <- time.Now()
		c-- // subtract one because we just wrote one
		for i := 0; c < 0 || i < c; i++ {
			ch <- <-t
		}
		close(ch)
	}()
	return ch
}

func main() {
	l := lever.New("cricket", nil)
	l.Add(lever.Param{
		Name:        "--limit",
		Description: "Number of times to print each metric. Less than 0 means no limit",
		Default:     "-1",
	})
	l.Add(lever.Param{
		Name:        "--cpu-interval",
		Description: "Interval at which to display cpu stats. Blank to not show",
		Default:     "1s",
	})
	l.Add(lever.Param{
		Name:        "--mem-interval",
		Description: "Interval at which to display memory stats. Blank to not show",
		Default:     "1s",
	})
	l.Add(lever.Param{
		Name:        "--disk-interval",
		Description: "Interval at which to display disk stats. Blank to not show",
		Default:     "1s",
	})
	l.Add(lever.Param{
		Name:        "--disk-io-interval",
		Description: "Interval at which to display disk io stats. Blank to not show",
		Default:     "1s",
	})
	l.Add(lever.Param{
		Name:        "--net-interval",
		Description: "Interval at which to display net stats. Blank to not show",
		Default:     "1s",
	})
	l.Add(lever.Param{
		Name:        "--ping-interval",
		Description: "Interval at which to ping hosts. Blank to not ping",
		Default:     "1s",
	})
	l.Add(lever.Param{
		Name:        "--ping-hosts",
		Description: "Comma-delimited set of hostnames or ips to ping. Blank to not ping",
	})
	l.Add(lever.Param{
		Name:        "--ping-count",
		Description: "Number of pings to send per interval",
		Default:     "3",
	})
	l.Parse()

	var wg sync.WaitGroup
	goTick := func(param string, extraTick bool, fn func(tick <-chan time.Time)) {
		s, _ := l.ParamStr(param)
		if s == "" {
			return
		}
		d, err := time.ParseDuration(s)
		if err != nil {
			llog.Fatal("could not parse duration", llog.KV{"param": param}, llog.ErrKV(err))
		} else if d == 0 {
			return
		}

		// extraTick is used in cases like diskIOLoop and cpuLoop because those
		// are actually printing out a diff every loop, so the first time they
		// run they can't print out anything useful because they don't have
		// anything to diff with.
		limit, _ := l.ParamInt("--limit")
		if extraTick && limit > 0 {
			limit++
		}

		wg.Add(1)
		go func() {
			fn(limitedTicker(limit, d))
			wg.Done()
		}()
	}

	goTick("--cpu-interval", true, cpuLoop)
	goTick("--mem-interval", false, memLoop)
	goTick("--disk-interval", false, diskLoop)
	goTick("--disk-io-interval", true, diskIOLoop)
	goTick("--net-interval", true, netLoop)

	hostsRaw, _ := l.ParamStr("--ping-hosts")
	if hostsRaw != "" {
		hosts := strings.Split(hostsRaw, ",")
		count, _ := l.ParamInt("--ping-count")
		goTick("--ping-interval", false, func(tick <-chan time.Time) {
			pingLoop(tick, hosts, count)
		})
	}

	wg.Wait()
	llog.Flush()
}

func cpuLoop(ch <-chan time.Time) {
	var full, diff subInfo
	var err error
	var doPrint bool
	for range ch {
		if full, diff, err = cpuDiff(full); err != nil {
			llog.Fatal("err calling cpuDiff", llog.ErrKV(err))
		}

		if doPrint && diff != nil {
			llog.Info("cpu stats (diff)", diff.toKV())
		}
		doPrint = true
	}
}

func memLoop(ch <-chan time.Time) {
	for range ch {
		m, err := mem()
		if err != nil {
			llog.Fatal("err calling mem", llog.ErrKV(err))
		}
		llog.Info("mem stats", llog.KV(m))
	}
}

func diskLoop(ch <-chan time.Time) {
	for range ch {
		dd, err := disk()
		if err != nil {
			llog.Fatal("err calling disk", llog.ErrKV(err))
		}
		for _, d := range dd {
			llog.Info("disk usage stats", llog.KV(d))
		}
	}
}

func diskIOLoop(ch <-chan time.Time) {
	// get the device list
	devList := func() []string {
		dd, err := disk()
		if err != nil {
			llog.Fatal("err calling disk", llog.ErrKV(err))
		}

		devs := make([]string, 0, len(dd))
		for _, d := range dd {
			devs = append(devs, d["fs"].(string))
		}
		return devs
	}

	var full, diff subInfoSet
	var err error
	var doPrint bool
	for range ch {
		if full, diff, err = diskDiff(full, devList()); err != nil {
			llog.Fatal("err calling diskDiff", llog.ErrKV(err))
		}

		if doPrint {
			for dev, d := range diff {
				kv := d.toKV()
				kv["fs"] = dev
				llog.Info("disk io stats (diff)", kv)
			}
		}
		doPrint = true
	}
}

func netLoop(ch <-chan time.Time) {
	var full, diff subInfoSet
	var err error
	for range ch {
		if full, diff, err = netDiff(full); err != nil {
			llog.Fatal("err calling netDiff", llog.ErrKV(err))
		}

		for dev, d := range diff {
			kv := d.toKV()
			kv["dev"] = dev
			llog.Info("net stats (diff)", kv)
		}

	}
}

type pingRes struct {
	d   time.Duration
	err error
}

func pingPromise(count int, addr string) chan pingRes {
	ch := make(chan pingRes, 1)
	go func() {
		p, err := ping.NewPinger(addr)
		if err != nil {
			ch <- pingRes{err: err}
			return
		}
		p.Count = count
		p.SetPrivileged(true)
		p.OnFinish = func(stats *ping.Statistics) {
			if stats.PacketsRecv == 0 {
				ch <- pingRes{err: errors.New("no pings completed")}
			} else {
				ch <- pingRes{d: stats.AvgRtt}
			}
		}
		p.Run()
	}()
	return ch
}

func pingLoop(ch <-chan time.Time, hosts []string, count int) {
	for range ch {
		proms := make([]chan pingRes, len(hosts))
		for i, host := range hosts {
			proms[i] = pingPromise(count, host)
		}

		for i := range proms {
			pr := <-proms[i]
			kv := llog.KV{"host": hosts[i]}
			if pr.err != nil {
				llog.Warn("ping failed", kv, llog.ErrKV(pr.err))
			} else {
				// convert to int because if it's left a float there's a ton of
				// trailing decimals in the log
				tkv := llog.KV{
					"tookMSAvg": int64(pr.d / time.Millisecond),
				}
				llog.Info("ping result", kv, tkv)
			}
		}
	}
}
