package main

import (
	"log"
)

const iniFile = "C:\\Program Files (x86)\\PLANETMEISTER\\Inf\\TPMFTP.ini"

func init() {
	log.SetPrefix("TPMFTPScan: ")
}
func max(nums ...int) int {
	maxNum := nums[0]
	for _, num := range nums[1:] {
		if maxNum < num {
			maxNum = num
		}
	}
	return maxNum
}

func main() {
	//for i := 0; i < 10; i++ {
	//	app := NewFTPConfigListInfo(i)
	//	app.Run()
	//
	//	time.Sleep(1 * time.Second)
	//}
	//
	//time.Sleep(50)
	//var wg sync.WaitGroup
	//for i := 1; i <= 10; i++ {
	//	app := NewFTPConfigListInfo(i)
	//
	//	wg.Add(1)
	//	go app.Run(&wg)
	//}
	//
	//wg.Wait()

}
