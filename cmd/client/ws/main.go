package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gorilla/websocket"
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

	h := make(map[string][]string)
	c, _, err := websocket.DefaultDialer.Dial("ws://localhost:8585/websocket", http.Header(h))
	if err != nil {
		return duration, err
	}

	fmt.Println("websocket on!")

	content := getBytesN(10 * 1024 * 1024)

	err = WriteAudioRealtime(content, 1000*10, func(a []byte) error {
		fmt.Println("msg len:", len(a))
		err = c.WriteMessage(websocket.BinaryMessage, a)
		return err
	})
	fmt.Println("send content done")
	if err != nil {
		return duration, err
	}

	err = c.WriteMessage(websocket.BinaryMessage, []byte{0x45, 0x4f, 0x53})
	if err != nil {
		return duration, err
	}

	start := time.Now()

	_, msg, err := c.ReadMessage()
	if err != nil {
		return duration, err
	}

	end := time.Now()
	duration = end.Sub(start)
	fmt.Println("wait duration:", duration, string(msg))

	return duration, nil
}

var AudioChunkSize = 1024 * 4 // 4Kb

func WriteAudioRealtime(audio []byte, audioLenInMilli int, write func([]byte) error) error {
	audioSize := len(audio)
	if audioSize == 0 {
		return nil
	}

	chunkNum := audioSize / AudioChunkSize
	lastChunkSize := audioSize - (chunkNum * AudioChunkSize)
	chunkTime := 0
	if chunkNum > 0 {
		chunkTime = audioLenInMilli / chunkNum
	}
	lastChunkTime := audioLenInMilli - (chunkNum * chunkTime)
	for i := 0; i < chunkNum; i++ {
		AtLeast(chunkTime, func() error {
			return write(audio[i*AudioChunkSize : (i+1)*AudioChunkSize])
		})
	}

	if lastChunkSize > 0 {
		AtLeast(lastChunkTime, func() error {
			return write(audio[audioSize-lastChunkSize : audioSize])
		})
	}

	return nil
}

func AtLeast(latencyInMilli int, action func() error) error {
	timer := time.NewTimer(time.Millisecond * time.Duration(latencyInMilli))
	defer func() {
		timer.Stop()
	}()

	err := action()
	if err != nil {
		return err
	}

	<-timer.C
	return nil
}

func getBytesN(n int) []byte {
	token := make([]byte, n)
	rand.Read(token)
	return token
}
