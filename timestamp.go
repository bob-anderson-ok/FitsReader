package main

import (
	"fmt"
	"github.com/montanaflynn/stats"
	"math"
)

func findFlashEdges() {
	// Find flash R edge
	fc := myWin.lightcurve // Shortened name for flash lightcurve
	maxFlashLevel, _ := maxInSlice(fc)
	minFlashLevel, _ := minInSlice(fc)
	midFlashLevel := (maxFlashLevel + minFlashLevel) / 2.0

	stats1 := new(EdgeStats)
	extractEdgeTimeAndStats(fc, "left", midFlashLevel, stats1)
	fmt.Printf("first edge at %0.6f\n", stats1.edgeAt)

	stats2 := new(EdgeStats)
	extractEdgeTimeAndStats(fc, "right", midFlashLevel, stats2)
	fmt.Printf("second edge at %0.6f\n", stats2.edgeAt)
}

type EdgeStats struct {
	intermediatePointIntensity float64
	bottomStd                  float64
	bottomMean                 float64
	topStd                     float64
	topMean                    float64
	edgeSigma                  float64
	edgeAt                     float64
	pSNR                       float64
	bSNR                       float64
	aSNR                       float64
}

func prettyPrintWing(wingName string, values []float64) {
	fmt.Printf("\n%s", wingName)
	for i := 0; i < len(values); i++ {
		if i%10 == 0 {
			fmt.Println()
		}
		fmt.Printf("%10.0f ", values[i])
	}
	fmt.Println()
}

func mean(data []float64) float64 {
	if len(data) == 0 {
		return 0.0
	}

	var sum = 0.0
	for _, d := range data {
		sum += d
	}
	return sum / float64(len(data))
}

func getTransitionPointData(flashWing []float64) (meanBottom, stdBottom, meanTop, stdTop float64, transitionIndex int) {
	transitionIndex = 0
	var a = 0
	var b = 0
	var maxDelta = 0.0

	for i := 0; i < len(flashWing)-2; i += 1 {
		delta := flashWing[i+2] - flashWing[i]
		if delta > maxDelta {
			maxDelta = delta
			a = i + 1
			b = i + 2
		}
	}

	//At this point, either a or b is the correct index for the transition point. We use logic to
	//pick the best

	bottom := flashWing[0:a]
	top := flashWing[a+1 : len(flashWing)-1]
	meanBottom = mean(bottom)
	stdBottom, _ = stats.StandardDeviation(bottom)
	meanTop = mean(top)
	stdTop, _ = stats.StandardDeviation(top)

	aIsCandidate := !(flashWing[a] < meanBottom)
	bIsCandidate := !(flashWing[b] > meanTop)

	if !(aIsCandidate || bIsCandidate) {
		transitionIndex = b
	} else {
		if !aIsCandidate {
			transitionIndex = b
		} else if !bIsCandidate {
			transitionIndex = a
		} else {
			bDelta := meanTop - flashWing[b]
			aDelta := flashWing[a] - meanBottom
			if bDelta > aDelta {
				transitionIndex = b
			} else {
				transitionIndex = a
			}
		}
	}
	return meanBottom, stdBottom, meanTop, stdTop, transitionIndex
}

func extractEdgeTimeAndStats(fc []float64, goalpost string, midFlashLevel float64, edgeStats *EdgeStats) {
	const debugPrint bool = true
	if debugPrint {
		fmt.Printf("\n\n")
	}

	var leftWing []float64
	var rightWing []float64
	var startingIndex = 0

	// left and right here refer to the left goalpost and the rightmost goalpost

	if goalpost == "left" {
		leftWing = getLeftFlashEdge(fc, midFlashLevel)
	} else {
		rightWing, startingIndex = getRightFlashEdge(fc, midFlashLevel)
	}

	var bottomStd, bottomMean, topStd, topMean float64
	var transitionPoint int

	if goalpost == "left" {
		bottomMean, bottomStd, topMean, topStd, transitionPoint = getTransitionPointData(leftWing)
		if debugPrint {
			fmt.Printf("leftBottom:  mean %f   std %f   leftTop: mean %f   std %f  leftTransitionPoint: %d",
				bottomMean, bottomStd, topMean, topStd, transitionPoint)
			prettyPrintWing("left wing:", leftWing)
		}
	}

	if goalpost == "right" {
		bottomMean, bottomStd, topMean, topStd, transitionPoint = getTransitionPointData(rightWing)
		if debugPrint {
			fmt.Printf("rightBottom:  mean %f   std %f   rightTop: mean %f   std %f  rightTransitionPoint: %d",
				bottomMean, bottomStd, topMean, topStd, transitionPoint)
			prettyPrintWing("right wing:", rightWing)
		}
	}

	edgeStats.bottomStd = bottomStd
	edgeStats.bottomMean = bottomMean
	edgeStats.topStd = topStd
	edgeStats.topMean = topMean

	p := fc[transitionPoint+startingIndex]
	delta := (topMean - p) / (topMean - bottomMean)

	edgeAt := float64(transitionPoint+startingIndex) + delta
	edgeStats.edgeAt = edgeAt

	sigmaP := bottomStd + (topStd-bottomStd)*(1.0-delta)
	pSNR := p / sigmaP
	edgeStats.pSNR = pSNR
	bSNR := topMean / topStd
	edgeStats.bSNR = bSNR
	aSNR := bottomMean / bottomStd
	edgeStats.aSNR = aSNR

	sigmaFrameFromRatio := delta * math.Sqrt(1.0/(bSNR*bSNR)+1.0/(aSNR*aSNR))
	sigmaFrame := sigmaP / (topMean - bottomMean)

	adjustedSigmaFrame := math.Sqrt(sigmaFrameFromRatio*sigmaFrameFromRatio + sigmaFrame*sigmaFrame)
	edgeStats.edgeSigma = adjustedSigmaFrame

	if debugPrint {
		fmt.Printf("\nA: %0.4f  B: %0.4f\n", bottomMean, topMean)
		fmt.Printf("sigmaA: %0.4f  sigmaB: %0.4f\n", bottomStd, topStd)
		fmt.Printf("p: %0.4f  delta: %0.6f\n", p, delta)
		fmt.Printf("edge of intermediate point: %0.6f\n", edgeAt)
		fmt.Printf("sigmaP: %0.4f  pSNR: %0.4f\n", sigmaP, pSNR)
		fmt.Printf("bSNR: %0.4f  aSNR: %0.4f\n", bSNR, aSNR)
		fmt.Printf("sigmaFrame: %0.6f\n", sigmaFrame)
		fmt.Printf("sigmaFrame from ratio: %0.6f\n", sigmaFrameFromRatio)
		fmt.Printf("adjustedSigmaFrame: %0.6f\n", adjustedSigmaFrame)
	}

	return
}

func getLeftFlashEdge(fc []float64, midFlashLevel float64) (leftWing []float64) {
	state := "accumulateBottom"

	for i := 0; i < len(fc); i++ {
		value := fc[i]
		if state == "accumulateBottom" {
			if value < midFlashLevel {
				leftWing = append(leftWing, value)
			} else {
				state = "accumulateTop"
			}
		}
		if state == "accumulateTop" {
			if value >= midFlashLevel {
				leftWing = append(leftWing, value)
			} else {
				break
			}
		}
	}
	return leftWing
}

func getRightFlashEdge(fc []float64, midFlashLevel float64) (rightWing []float64, startingIndex int) {
	var lastFlashBottomStart int
	var lastFlashTopEnd int

	state := "traverseRightBottom"
	k := len(fc) - 1 // We use k to iterate backwards through the flashLightCurve

	for {
		value := fc[k]
		if state == "traverseRightBottom" {
			if value < midFlashLevel { // We're still in the flash off portion of the tail
				k -= 1
			} else {
				state = "traverseTop"
				lastFlashTopEnd = k // Save this because we need to know where the top of the last flash ends
			}
		}
		if state == "traverseTop" {
			if value >= midFlashLevel { // we're still in the flash on portion
				k -= 1
			} else {
				state = "traverseLeftBottom"
			}
		}

		if state == "traverseLeftBottom" {
			//k -= FLASH_OFF_FRAME_COUNT
			k -= 10 // TODO Make this more general - this only works if the acquisition program enforces this value
			lastFlashBottomStart = k
			break
		}
	}
	rightWing = fc[lastFlashBottomStart : lastFlashTopEnd+1]
	return rightWing, lastFlashBottomStart
}

func maxInSlice(data []float64) (biggest float64, index int) {
	// data cannot be empty - there is no error check - panic will occur if empty
	biggest = data[0]
	index = 0
	for i := 1; i < len(data); i++ {
		if data[i] > biggest {
			biggest = data[i]
			index = i
		}
	}
	return biggest, index
}

func minInSlice(data []float64) (smallest float64, index int) {
	// data cannot be empty - there is no error check - panic will occur if empty
	smallest = data[0]
	index = 0
	for i := 1; i < len(data); i++ {
		if data[i] < smallest {
			smallest = data[i]
			index = i
		}
	}
	return smallest, index
}
