package item

import (
	"time"
)

type Smeltable interface {
	SmeltInfo() SmeltInfo
}

type Fuel interface {
	FuelInfo() FuelInfo
}

type SmeltInfo struct {
	Product Stack

	Experience float64

	Food bool

	Ores bool
}

func newSmeltInfo(product Stack, experience float64) SmeltInfo {
	return SmeltInfo{
		Product:    product,
		Experience: experience,
	}
}

func newFoodSmeltInfo(product Stack, experience float64) SmeltInfo {
	return SmeltInfo{
		Product:    product,
		Experience: experience,
		Food:       true,
	}
}

func newOreSmeltInfo(product Stack, experience float64) SmeltInfo {
	return SmeltInfo{
		Product:    product,
		Experience: experience,
		Ores:       true,
	}
}

type FuelInfo struct {
	Duration time.Duration

	Residue Stack
}

func (f FuelInfo) WithResidue(residue Stack) FuelInfo {
	f.Residue = residue
	return f
}

func newFuelInfo(duration time.Duration) FuelInfo {
	return FuelInfo{Duration: duration}
}
