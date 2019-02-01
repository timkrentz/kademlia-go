package main

import (
	"fmt"
	k "github.com/timkrentz/kademlia-go"
	"golang.org/x/sys/unix"
	"math/rand"
	"strings"
	"time"
)

//var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {

	//flag.Parse()
	//if *cpuprofile != "" {
	//	f, err := os.Create(*cpuprofile)
	//	if err != nil {
	//		fmt.Errorf("%s\n",err)
	//	}
	//	pprof.StartCPUProfile(f)
	//	defer pprof.StopCPUProfile()
	//}

	ip := k.GetOutboundIP()
	node := k.NewNode(ip)

	rand.Seed(time.Now().UnixNano())

	startPhase := 15000
	startWait := rand.Int() % startPhase
	time.Sleep(time.Duration(startWait) * time.Millisecond)

	go node.Start()
	defer node.Stop()

	time.Sleep(time.Duration(startPhase-startWait) * time.Millisecond)

	time.Sleep(5 * time.Second)
	node.Print()

	endTime := time.Now()
	endTime = endTime.Add(80 * time.Second)

	if strings.Contains(node.IP, "172.21.20.43") {
		//tic := time.Now()
		for i := 0; i < 10000; i++ {
			timestamp := time.Now()
			ts, _ := unix.TimeToTimespec(timestamp)
			_ = node.Put("PMU_A", i, ts)
			time.Sleep(time.Until(timestamp.Add(17 * time.Millisecond)))
		}
		//fmt.Printf("WRITE 10000: %f seconds\n",time.Since(tic).Nanoseconds()/1000000000.0)
	}

	time.Sleep(time.Until(endTime))

	endTime = time.Now()
	endTime = endTime.Add(30 * time.Second)

	if strings.Contains(node.IP, "172.21.20.42") {
		timestamp := time.Now()
		tsB, err := unix.TimeToTimespec(timestamp)
		if err != nil {
			fmt.Errorf("Main: timespec conversion error: %s", err)
		}
		tsA := tsB
		//retrieve last minute of data
		tsA.Sec -= 80
		values, err := node.Get("PMU_A", tsA, tsB)
		if err != nil {
			fmt.Errorf("Main: Node.Get() error: %s", err)
		}
		fmt.Println("DATA RETRIEVED", values)
	}

	time.Sleep(time.Until(endTime))

	//time.Sleep(10 * time.Second)
	node.Print()
	fmt.Println("Experiment Done!")
}
