package main

import (
	"flag"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/imroc/req"
)

var (
	round       = 1
	parallelNum = 1
)

func main() {
	_ = flag.Set("logtostderr", "true")
	flag.Parse()

	totalSum, totalAvg, totalDuration, totalErrCount := time.Duration(0), time.Duration(0), make([]time.Duration, 0), 0
	for i := 0; i < round; i++ {
		sum, avg, duration, errCount := parallelAsk(parallelNum, ask)
		totalSum += sum
		totalAvg += avg
		totalDuration = append(totalDuration, duration...)
		totalErrCount += errCount
	}

	fmt.Println("total sum:", totalSum)
	fmt.Println("total avg:", totalAvg/time.Duration(parallelNum))
	fmt.Println("total error count:", totalErrCount)
	fmt.Println("total error rate:", float32(totalErrCount)/float32(parallelNum*round))
	fmt.Println("total succeed rate:", float32(1)-float32(totalErrCount)/float32(parallelNum*round))

	sort.Slice(totalDuration, func(i, j int) bool {
		return totalDuration[i] < totalDuration[j]
	})
	fmt.Println("totalDuration:", totalDuration)
	s := time.Duration(0)
	num50, num75, num90, num99 := time.Duration(0), time.Duration(0), time.Duration(0), time.Duration(0)
	for i, d := range totalDuration {
		s += d
		if float32(i+1)/float32(len(totalDuration)) > float32(0.5) && num50 == time.Duration(0) {
			num50 = d
		}
		if float32(i+1)/float32(len(totalDuration)) > float32(0.75) && num75 == time.Duration(0) {
			num75 = d
		}
		if float32(i+1)/float32(len(totalDuration)) > float32(0.9) && num90 == time.Duration(0) {
			num90 = d
		}
		if float32(i+1)/float32(len(totalDuration)) > float32(0.99) && num99 == time.Duration(0) {
			num99 = d
		}
	}
	fmt.Println("total duration avg:", s/time.Duration(len(totalDuration)))
	fmt.Println("total duration 50%:", num50)
	fmt.Println("total duration 75%:", num75)
	fmt.Println("total duration 90%:", num90)
	fmt.Println("total duration 99%:", num99)
}

func parallelAsk(n int, f func() (time.Duration, error)) (time.Duration, time.Duration, []time.Duration, int) {
	if n < 1 {
		return time.Duration(0), time.Duration(0), nil, 0
	}

	type counter struct {
		start    time.Time
		end      time.Time
		duration sync.Map
		err      error
	}
	countSlice := make([]*counter, n)
	var wg sync.WaitGroup
	for i := 1; i <= n; i++ {
		index := i
		wg.Add(1)
		go func() {
			defer wg.Done()

			c := &counter{}

			func() {
				fmt.Printf("No.%d started\n", index)
				c.start = time.Now()
			}()

			defer func() {
				fmt.Printf("No.%d done\n", index)
				c.end = time.Now()
				countSlice[index-1] = c
			}()

			if duration, err := f(); err != nil {
				fmt.Printf("No.%d err: %v", index, err)
				c.err = err
			} else {
				// only count succeed duration
				c.duration.Store(index, duration)
				fmt.Println("index:", index, duration)
			}

		}()
	}
	wg.Wait()

	sum, durationSum := time.Duration(0), make([]time.Duration, 0)
	errCount := 0

	for _, c := range countSlice {
		d := c.end.Sub(c.start)
		sum += d
		if c.err != nil {
			errCount++
		}

		c.duration.Range(func(k, v interface{}) bool {
			fmt.Println("range:", k, v)
			durationSum = append(durationSum, v.(time.Duration))
			return true
		})

	}
	fmt.Println("sum:", sum)
	fmt.Println("avg:", sum/time.Duration(n))
	fmt.Println("durationSum:", durationSum)
	fmt.Println("error count:", errCount)

	return sum, sum / time.Duration(n), durationSum, errCount
}

func ask() (time.Duration, error) {
	fmt.Println("start ask...")
	duration := time.Duration(0)

	start := time.Now()

	resp, err := req.Get("http://localhost:8585/httpcase")
	if err != nil {
		return duration, err
	}
	end := time.Now()
	duration = end.Sub(start)

	actualBody, err := resp.ToBytes()
	if err != nil {
		return duration, err
	}
	fmt.Println("wait duration:", duration, string(actualBody))

	return duration, nil
}
