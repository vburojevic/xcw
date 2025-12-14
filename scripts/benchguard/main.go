package main

import (
	"bufio"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

type benchResult struct {
	Name     string
	TimeNs   float64
	BytesOp  float64
	AllocsOp float64
}

var timeUnitToNs = map[string]float64{
	"ns/op": 1,
	"us/op": 1e3,
	"µs/op": 1e3,
	"ms/op": 1e6,
	"s/op":  1e9,
}

func parseBenchFile(path string) (map[string]benchResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	results := make(map[string]benchResult)
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "Benchmark") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		name := fields[0]
		timeIdx := -1
		for i := 1; i < len(fields); i++ {
			if _, ok := timeUnitToNs[fields[i]]; ok {
				timeIdx = i
				break
			}
		}
		if timeIdx < 1 {
			continue
		}

		timeVal, err := strconv.ParseFloat(fields[timeIdx-1], 64)
		if err != nil {
			continue
		}
		timeNs := timeVal * timeUnitToNs[fields[timeIdx]]

		bytesOp := math.NaN()
		for i := 1; i < len(fields); i++ {
			if fields[i] != "B/op" {
				continue
			}
			if i < 1 {
				break
			}
			v, err := strconv.ParseFloat(fields[i-1], 64)
			if err == nil {
				bytesOp = v
			}
			break
		}

		allocsOp := math.NaN()
		for i := 1; i < len(fields); i++ {
			if fields[i] != "allocs/op" {
				continue
			}
			if i < 1 {
				break
			}
			v, err := strconv.ParseFloat(fields[i-1], 64)
			if err == nil {
				allocsOp = v
			}
			break
		}

		results[name] = benchResult{
			Name:     name,
			TimeNs:   timeNs,
			BytesOp:  bytesOp,
			AllocsOp: allocsOp,
		}
	}

	if err := sc.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

type regression struct {
	Name   string
	Metric string
	Base   float64
	Head   float64
	Ratio  float64
}

func ratio(base, head float64) float64 {
	if base == 0 {
		if head == 0 {
			return 1
		}
		return math.Inf(1)
	}
	return head / base
}

func main() {
	var basePath string
	var headPath string
	var maxTimeRatio float64
	var maxBytesRatio float64
	var maxAllocsRatio float64

	flag.StringVar(&basePath, "base", "", "Path to base benchmark output")
	flag.StringVar(&headPath, "head", "", "Path to head benchmark output")
	flag.Float64Var(&maxTimeRatio, "max-time-ratio", 2.0, "Fail if time/op regresses by more than this ratio")
	flag.Float64Var(&maxBytesRatio, "max-bytes-ratio", 1.5, "Fail if B/op regresses by more than this ratio")
	flag.Float64Var(&maxAllocsRatio, "max-allocs-ratio", 1.5, "Fail if allocs/op regresses by more than this ratio")
	flag.Parse()

	if basePath == "" || headPath == "" {
		_, _ = fmt.Fprintln(os.Stderr, "usage: benchguard --base <file> --head <file>")
		os.Exit(2)
	}

	base, err := parseBenchFile(basePath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to parse base: %v\n", err)
		os.Exit(2)
	}
	head, err := parseBenchFile(headPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to parse head: %v\n", err)
		os.Exit(2)
	}

	var regressions []regression
	compared := 0
	for name, b := range base {
		h, ok := head[name]
		if !ok {
			continue
		}
		compared++

		if r := ratio(b.TimeNs, h.TimeNs); r > maxTimeRatio {
			regressions = append(regressions, regression{Name: name, Metric: "time/op", Base: b.TimeNs, Head: h.TimeNs, Ratio: r})
		}
		if !math.IsNaN(b.BytesOp) && !math.IsNaN(h.BytesOp) {
			if r := ratio(b.BytesOp, h.BytesOp); r > maxBytesRatio {
				regressions = append(regressions, regression{Name: name, Metric: "B/op", Base: b.BytesOp, Head: h.BytesOp, Ratio: r})
			}
		}
		if !math.IsNaN(b.AllocsOp) && !math.IsNaN(h.AllocsOp) {
			if r := ratio(b.AllocsOp, h.AllocsOp); r > maxAllocsRatio {
				regressions = append(regressions, regression{Name: name, Metric: "allocs/op", Base: b.AllocsOp, Head: h.AllocsOp, Ratio: r})
			}
		}
	}

	if compared == 0 {
		_, _ = fmt.Fprintln(os.Stderr, "no overlapping benchmarks found between base and head outputs")
		os.Exit(2)
	}

	sort.Slice(regressions, func(i, j int) bool {
		if regressions[i].Ratio == regressions[j].Ratio {
			if regressions[i].Name == regressions[j].Name {
				return regressions[i].Metric < regressions[j].Metric
			}
			return regressions[i].Name < regressions[j].Name
		}
		return regressions[i].Ratio > regressions[j].Ratio
	})

	if len(regressions) == 0 {
		fmt.Printf("benchguard: ok (%d benchmarks compared)\n", compared)
		return
	}

	fmt.Printf("benchguard: found %d regressions (%d benchmarks compared)\n", len(regressions), compared)
	for _, r := range regressions {
		switch r.Metric {
		case "time/op":
			fmt.Printf("- %s %s: %.0fns -> %.0fns (x%.2f)\n", r.Name, r.Metric, r.Base, r.Head, r.Ratio)
		default:
			fmt.Printf("- %s %s: %.0f -> %.0f (x%.2f)\n", r.Name, r.Metric, r.Base, r.Head, r.Ratio)
		}
	}

	os.Exit(1)
}
