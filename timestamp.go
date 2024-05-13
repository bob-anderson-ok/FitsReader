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
	extractEdgeTimeAndStats(fc, 0, midFlashLevel, stats1)
	fmt.Printf("first edge at %0.6f\n", stats1.edgeAt)

	stats2 := new(EdgeStats)
	extractEdgeTimeAndStats(fc, 60, midFlashLevel, stats2)
	fmt.Printf("second edge at %0.6f\n", stats2.edgeAt)
}

type EdgeStats struct {
	intermediatePointIntensity float64
	leftStd                    float64
	leftMean                   float64
	rightStd                   float64
	rightMean                  float64
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
func extractEdgeTimeAndStats(fc []float64, startingFrame int, midFlashLevel float64, edgeStats *EdgeStats) {
	const debugPrint bool = true
	if debugPrint {
		fmt.Printf("\n\n")
	}
	leftWing, rightWing := getFlashEdge(fc[startingFrame:], midFlashLevel)
	if debugPrint {
		prettyPrintWing("left wing:", leftWing)
		prettyPrintWing("right wing:", rightWing)
	}
	leftStd, _ := stats.StandardDeviation(leftWing[0 : len(leftWing)-1])
	leftMean, _ := stats.Mean(leftWing[0 : len(leftWing)-1])
	rightStd, _ := stats.StandardDeviation(rightWing[1 : len(rightWing)-2])
	rightMean, _ := stats.Mean(rightWing[1 : len(rightWing)-2])

	edgeStats.leftStd = leftStd
	edgeStats.leftMean = leftMean
	edgeStats.rightStd = rightStd
	edgeStats.rightMean = rightMean

	intermediatePointsFound := 0
	leftHand := false
	rightHand := false
	leftIntermediatePoint := leftWing[len(leftWing)-1]
	if leftIntermediatePoint > leftMean && leftIntermediatePoint < rightMean {
		if debugPrint {
			fmt.Printf("intermediate point intensity: %0.4f from leftWing\n", leftIntermediatePoint)
		}
		intermediatePointsFound += 1
		leftHand = true
	}

	rightIntermediatePoint := rightWing[0]
	if rightIntermediatePoint > leftMean && rightIntermediatePoint < rightMean {
		if debugPrint {
			fmt.Printf("intermediate point intensity: %0.4f from rightWing\n", rightIntermediatePoint)
		}
		intermediatePointsFound += 1
		rightHand = true
	}

	var p float64
	var pindex int
	switch intermediatePointsFound {
	case 0:
		// TODO Deal with this properly
		fmt.Println("There is no valid intermediate point")
	case 1:
		if leftHand {
			if debugPrint {
				fmt.Println("Choosing singleton lefthand intermediate point")
			}
			p = leftIntermediatePoint
			edgeStats.intermediatePointIntensity = p
			pindex = startingFrame + len(leftWing) - 1
		}
		if rightHand {
			if debugPrint {
				fmt.Println("Choosing singleton righthand intermediate point")
			}
			p = rightIntermediatePoint
			edgeStats.intermediatePointIntensity = p
			pindex = startingFrame + len(leftWing)
		}
	case 2:
		// We choose the one closest to the midFlash level
		deltaLeft := math.Abs(midFlashLevel - leftIntermediatePoint)
		deltaRight := math.Abs(rightIntermediatePoint - midFlashLevel)
		if deltaLeft < deltaRight {
			if debugPrint {
				fmt.Println("Choosing lefthand intermediate point")
			}
			p = leftIntermediatePoint
			edgeStats.intermediatePointIntensity = p
			pindex = startingFrame + len(leftWing) - 1
		} else {
			if debugPrint {
				fmt.Println("Choosing righthand intermediate point")
			}
			p = rightIntermediatePoint
			edgeStats.intermediatePointIntensity = p
			pindex = startingFrame + len(leftWing)
		}
	default:
		fmt.Println("Programming error: intermediate point count invalid")
		panic("Programming error: intermediate point count invalid")
	}

	// 0.0 <= delta <= 1.0
	delta := (rightMean - p) / (rightMean - leftMean)

	edgeAt := float64(pindex) + delta
	edgeStats.edgeAt = edgeAt

	sigmaP := leftStd + (rightStd-leftStd)*(1.0-delta)
	pSNR := p / sigmaP
	edgeStats.pSNR = pSNR
	bSNR := rightMean / rightStd
	edgeStats.bSNR = bSNR
	aSNR := leftMean / leftStd
	edgeStats.aSNR = aSNR

	sigmaFrameFromRatio := delta * math.Sqrt(1.0/(bSNR*bSNR)+1.0/(aSNR*aSNR))
	sigmaFrame := sigmaP / (rightMean - leftMean)

	adjustedSigmaFrame := math.Sqrt(sigmaFrameFromRatio*sigmaFrameFromRatio + sigmaFrame*sigmaFrame)
	edgeStats.edgeSigma = adjustedSigmaFrame

	if debugPrint {
		fmt.Printf("A: %0.4f  B: %0.4f\n", leftMean, rightMean)
		fmt.Printf("sigmaA: %0.4f  sigmaB: %0.4f\n", leftStd, rightStd)
		fmt.Printf("p: %0.4f  delta: %0.6f\n", p, delta)
		fmt.Printf("edge of intermediate point: %0.6f\n", edgeAt)
		fmt.Printf("sigmaP: %0.4f  pSNR: %0.4f\n", sigmaP, pSNR)
		fmt.Printf("bSNR: %0.4f  aSNR: %0.4f\n", bSNR, aSNR)
		fmt.Printf("sigmaFrame: %0.6f\n", sigmaFrame)
		fmt.Printf("sigmaFrame from ratio: %0.6f\n", sigmaFrameFromRatio)
		fmt.Printf("adjustedSigmaFrame: %0.6f\n", adjustedSigmaFrame)
	}

	// Now we deal with the D edge
	p = rightWing[len(rightWing)-1]
	pindex = startingFrame + len(leftWing) + len(rightWing) - 1
	delta = (rightMean - p) / (rightMean - leftMean)
	edgeAt = float64(pindex) + (1.0 - delta)
	fmt.Printf("D p: %0.4f  1.0 - delta: %0.6f\n", p, 1.0-delta)
	fmt.Printf("D edge of intermediate point: %0.6f\n", edgeAt)
	return
}

func getFlashEdge(fc []float64, midFlashLevel float64) (leftWing, rightWing []float64) {
	state := "accumulateLeft"

	for i := 0; i < len(fc); i++ {
		value := fc[i]
		if state == "accumulateLeft" {
			if value < midFlashLevel {
				leftWing = append(leftWing, value)
			} else {
				state = "accumulateRight"
			}
		}
		if state == "accumulateRight" {
			if value >= midFlashLevel {
				rightWing = append(rightWing, value)
			} else {
				rightWing = append(rightWing, value)
				break
			}
		}
	}
	return leftWing, rightWing
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
