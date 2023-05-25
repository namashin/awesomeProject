package main

import (
	"fmt"
	"net/http"
	"regexp"
)

func max(nums ...int) int {
	maxNum := nums[0]

	for _, num := range nums[1:] {
		if maxNum < num {
			maxNum = num
		}
	}

	return maxNum
}

func min(nums ...int) int {
	minNum := nums[0]

	for _, num := range nums[1:] {
		if num < minNum {
			minNum = num
		}
	}

	return minNum
}

func sum(nums ...int) int {
	var total int

	for _, num := range nums {
		total += num
	}

	return total
}

var re = regexp.MustCompile(`/users/([0-9]+)`)

func main() {
	// test
}

func UsersHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	match := re.FindStringSubmatch(path)
	if len(match) > 1 {
		id := match[1]
		fmt.Fprintf(w, "ユーザーID: %s", id)
	} else {
		// マッチしなかった場合の処理
		fmt.Fprintf(w, "全てのユーザ")
	}
}
