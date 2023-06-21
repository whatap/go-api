package active

import "time"

func GetActiveStats(instanceCount int32, activeStatChannel chan []int16) []int16 {

	if instanceCount < 1 {
		return nil
	}
	var activeStats []int16
	var activeStatsAcc []int16 = make([]int16, 5, 5)
	for i := 0; i < int(instanceCount); i++ {
		select {
		case activeStats = <-activeStatChannel:
		case <-time.After(100 * time.Millisecond):
			break
		}
		if len(activeStats) == 5 {
			for j := 0; j < 5; j++ {
				activeStatsAcc[j] += activeStats[j]
			}
		}

	}

	return activeStatsAcc
}
