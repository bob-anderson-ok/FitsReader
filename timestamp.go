package main

import (
	"fmt"
	"github.com/montanaflynn/stats"
	"math"
)

func findFlashEdges() {
	// Find first flash R edge
	fc := myWin.lightcurve // Shortened name for flash lightcurve
	maxFlashLevel, _ := maxInSlice(fc)
	minFlashLevel, _ := minInSlice(fc)
	midFlashLevel := (maxFlashLevel + minFlashLevel) / 2.0

	leftWing, rightWing := getFlashEdge(fc, midFlashLevel)
	leftStd, _ := stats.StandardDeviation(leftWing[0 : len(leftWing)-1])
	leftMean, _ := stats.Mean(leftWing[0 : len(leftWing)-1])
	rightStd, _ := stats.StandardDeviation(rightWing[1:])
	rightMean, _ := stats.Mean(rightWing[1:])

	intermediatePointsFound := 0
	leftHand := false
	rightHand := false
	leftIntermediatePoint := leftWing[len(leftWing)-1]
	if leftIntermediatePoint > leftMean && leftIntermediatePoint < rightMean {
		fmt.Printf("intermediate point intensity: %0.4f from leftWing\n", leftIntermediatePoint)
		intermediatePointsFound += 1
		leftHand = true
	}

	rightIntermediatePoint := rightWing[0]
	if rightIntermediatePoint > leftMean && rightIntermediatePoint < rightMean {
		fmt.Printf("intermediate point intensity: %0.4f from rightWing\n", rightIntermediatePoint)
		intermediatePointsFound += 1
		rightHand = true
	}

	var p float64
	switch intermediatePointsFound {
	case 0:
		fmt.Println("There is no valid intermediate point")
	case 1:
		if leftHand {
			fmt.Println("Choosing singleton lefthand intermediate point")
			p = leftIntermediatePoint
		}
		if rightHand {
			fmt.Println("Choosing singleton righthand intermediate point")
			p = rightIntermediatePoint
		}
	case 2:
		// We choose the on closest to the midFlash level
		deltaLeft := math.Abs(midFlashLevel - leftIntermediatePoint)
		deltaRight := math.Abs(rightIntermediatePoint - midFlashLevel)
		if deltaLeft < deltaRight {
			fmt.Println("Choosing lefthand intermediate point")
			p = leftIntermediatePoint
		} else {
			fmt.Println("Choosing righthand intermediate point")
			p = rightIntermediatePoint
		}
	default:
		fmt.Println("Programming error: intermediate point count invalid")
		panic("Programming error: intermediate point count invalid")
	}

	fmt.Printf("leftMean: %0.4f  rightMean: %0.4f\n", leftMean, rightMean)
	fmt.Printf("leftStd: %0.4f  rightStd: %0.4f\n", leftStd, rightStd)

	delta := (rightMean - p) / (rightMean - leftMean)
	fmt.Printf("p: %0.4f  delta: %0.6f\n", p, delta)

	sigmaP := leftStd + (rightStd-leftStd)*(1.0-delta)
	pSNR := p / sigmaP
	bSNR := rightMean / rightStd
	aSNR := leftMean / leftStd
	fmt.Printf("sigmaP: %0.4f  pSNR: %0.4f\n", sigmaP, pSNR)
	fmt.Printf("bSNR: %0.4f  aSNR: %0.4f\n", bSNR, aSNR)
	addedSigmaP := p * math.Sqrt(1.0/(bSNR*bSNR)+1.0/(aSNR*aSNR))
	fmt.Printf("addedSigmaP: %0.4f\n", addedSigmaP)
	adjustedSigmaP := math.Sqrt(sigmaP*sigmaP + addedSigmaP*addedSigmaP)
	fmt.Printf("adjustedSigmaP: %0.4f\n", adjustedSigmaP)
	sigmaFrame := adjustedSigmaP / (rightMean - leftMean)
	fmt.Printf("sigmaFrame: %0.6f\n", sigmaFrame)
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
