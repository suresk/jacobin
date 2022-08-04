package management

var counterMap = make(map[string]int64)

var counterChan = make(chan string)

func consume() {
	for name := range counterChan {
		_, ok := counterMap[name]
		if !ok {
			counterMap[name] = 1
		} else {
			counterMap[name] += 1
		}
	}
}

func StartMetricWriter() {
	go consume()
}

func StopMetricWriter() {
	close(counterChan)
}

func IncrementCounter(name string) {
	counterChan <- name
}

func GetCounters() map[string]int64 {
	return counterMap
}
