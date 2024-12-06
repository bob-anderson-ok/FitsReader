package main

import (
	"fmt"
	"time"

	// "fmt"
	"github.com/montanaflynn/stats"
	"log"
	"math"
)

func findFlashEdges() bool {
	const baseZone = 8
	const flashThresholdFactor = 1.1
	//const flashRegion = 35
	log.Println("")
	log.Println("================= flash edge detection ===================")
	fc := myWin.lightcurve // Shortened name for flash lightcurve

	//baseStdDev, _ := stats.StandardDeviationPopulation(fc[0:baseZone])
	baseMean, _ := stats.Mean(fc[0:baseZone])
	var maxFlashLevel float64
	var midFlashLevel float64
	var foundMaxFlashLevel = false
	for i := range len(fc) {
		if fc[i] > flashThresholdFactor*baseMean {
			if i+2 < len(fc) {
				maxFlashLevel = fc[i+2]
				foundMaxFlashLevel = true
				break
			} else if i+1 < len(fc) {
				maxFlashLevel = fc[i+1]
				foundMaxFlashLevel = true
				break
			} else {
				maxFlashLevel = fc[i]
				foundMaxFlashLevel = true
				break
			}
		}
	}

	if foundMaxFlashLevel {
		midFlashLevel = (maxFlashLevel + baseMean) / 2.0
	} else {
		// Fatal condition in findFlashEdges
		log.Println("")
		log.Println("      flash edge detection failed")
		return false
	}

	myWin.leftGoalpostStats = new(EdgeStats)
	extractEdgeTimeAndStats(fc, "left", midFlashLevel, myWin.leftGoalpostStats)
	log.Printf("first edge at %0.6f\n", myWin.leftGoalpostStats.edgeAt)

	myWin.rightGoalpostStats = new(EdgeStats)
	extractEdgeTimeAndStats(fc, "right", midFlashLevel, myWin.rightGoalpostStats)
	log.Printf("second edge at %0.6f\n", myWin.rightGoalpostStats.edgeAt)

	return true
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
	totalTimeErr               float64
}

func prettyPrintWing(wingName string, values []float64) {
	log.Printf("\n%s", wingName)
	for i := 0; i < len(values); i++ {
		if i%10 == 0 {
			log.Println()
		}
		log.Printf("%3d  %10.0f ", i, values[i])
	}
	log.Println()
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

	if len(flashWing) < 8 {
		log.Fatal("flash wing too short")
	}

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
		log.Printf("\n")
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
			log.Printf("leftBottom:  mean %f   std %f",
				bottomMean, bottomStd)
			log.Printf("leftTop:     mean %f   std %f",
				topMean, topStd)
			log.Printf("left transition value: %f @ %d",
				leftWing[transitionPoint], transitionPoint)
			prettyPrintWing("left wing:", leftWing)
		}
	}

	if goalpost == "right" {
		bottomMean, bottomStd, topMean, topStd, transitionPoint = getTransitionPointData(rightWing)
		if debugPrint {
			log.Printf("rightBottom:  mean %f   std %f",
				bottomMean, bottomStd)
			log.Printf("rightTop:     mean %f   std %f",
				topMean, topStd)
			log.Printf("right transition value: %f @ %d",
				rightWing[transitionPoint], transitionPoint)
			prettyPrintWing("right wing:", rightWing)
		}
	}

	edgeStats.bottomStd = bottomStd
	edgeStats.bottomMean = bottomMean
	edgeStats.topStd = topStd
	edgeStats.topMean = topMean

	topThresholdForValidTransitionPoint := topMean - topStd          // Arbitrary criteria of  1 std
	bottomThresholdForValidTransitionPoint := bottomMean + bottomStd // Arbitrary criteria of  1 std

	averagePixelValueInTop := topMean / float64(myWin.numPixels)
	log.Printf("average pixel value in top: %0.1f", averagePixelValueInTop)

	indexOfTransitionPoint := transitionPoint + startingIndex
	p := fc[indexOfTransitionPoint]
	var delta float64
	if bottomThresholdForValidTransitionPoint < p && p < topThresholdForValidTransitionPoint {
		delta = (topMean - p) / (topMean - bottomMean)
	} else {
		delta = 0.0
	}

	edgeAt := float64(transitionPoint+startingIndex) + delta
	edgeStats.edgeAt = edgeAt

	systemTimeAtTransitionPoint := myWin.sysStartTimes[indexOfTransitionPoint]
	timeCorrectionSeconds := delta * myWin.sysTimeDeltaSeconds[indexOfTransitionPoint]
	systemTimeAtEdge := systemTimeAtTransitionPoint.Add(time.Duration(timeCorrectionSeconds * 1_000_000_000))
	fmt.Println(systemTimeAtEdge, timeCorrectionSeconds)
	sysUtcOffset := systemTimeAtEdge.Sub(systemTimeAtTransitionPoint)
	fmt.Println("sysUtcOffset", sysUtcOffset)
	// Calculate system time at edge from myWin.sysStartTimes

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

	if averagePixelValueInTop > maxAllowedFlashLevel {
		myWin.flashIntensityValid = false
		log.Println("!!! flash too bright !!!  Setting edgeSigma to 0.5")
		edgeStats.edgeSigma = 0.5
		adjustedSigmaFrame = 0.5
		sigmaFrame = 0.5
		sigmaFrameFromRatio = 0.5
	}

	if debugPrint {
		log.Printf("\nA: %0.4f  B: %0.4f\n", bottomMean, topMean)
		log.Printf("sigmaA: %0.4f  sigmaB: %0.4f\n", bottomStd, topStd)
		log.Printf("transition point thresholds:  bottom %0.4f   top %0.4f\n",
			bottomThresholdForValidTransitionPoint, topThresholdForValidTransitionPoint)
		log.Printf("p: %0.4f  delta: %0.6f\n", p, delta)
		log.Printf("edge of intermediate point: %0.6f\n", edgeAt)
		log.Printf("sigmaP: %0.4f  pSNR: %0.4f\n", sigmaP, pSNR)
		log.Printf("bSNR: %0.4f  aSNR: %0.4f\n", bSNR, aSNR)
		log.Printf("sigmaFrame: %0.6f\n", sigmaFrame)
		log.Printf("sigmaFrame from ratio: %0.6f\n", sigmaFrameFromRatio)
		log.Printf("adjustedSigmaFrame: %0.6f\n", adjustedSigmaFrame)
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
			// There should reliably be at least 10 baseline values to the left of the righthand goalpost
			// flash beginning, so this is a safe calculation.
			k -= 10
			lastFlashBottomStart = k
			break
		}
	}
	rightWing = fc[lastFlashBottomStart : lastFlashTopEnd+1]
	return rightWing, lastFlashBottomStart
}
