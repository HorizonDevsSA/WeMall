package routing

import (
	"math"
)

// CalculateDistance calculates the distance in kilometers between two points using the Haversine formula.
func CalculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371.0 // km
	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0

	rLat1 := lat1 * math.Pi / 180.0
	rLat2 := lat2 * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(rLat1)*math.Cos(rLat2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

type RoutingDecision struct {
	CarrierType string  // 'crowdsourced' | '3pl'
	CarrierName string  // e.g. "WeMall Instant Rider" or "SF Express" / "DHL"
	ShippingFee float64 // calculated price in USD
	DistanceKm  float64
}

// RoutePackage evaluates parcel specs and returns the best carrier type and pricing.
func RoutePackage(senderCity, recipientCity string, lat1, lon1, lat2, lon2 float64, weightKg float64, length, width, height int32) *RoutingDecision {
	distance := CalculateDistance(lat1, lon1, lat2, lon2)
	isSameCity := senderCity == recipientCity

	// Crowdsourced Courier criteria: same city, under 15km, under 10kg, volume under 125,000 cm³
	if isSameCity && distance < 15.0 && weightKg < 10.0 && (length*width*height) < 125000 {
		fee := 3.00 + (distance * 0.50)
		return &RoutingDecision{
			CarrierType: "crowdsourced",
			CarrierName: "WeMall Instant Rider",
			ShippingFee: math.Round(fee*100) / 100,
			DistanceKm:  distance,
		}
	}

	// Fallback to 3PL: SF Express for domestic, DHL for international/overseas
	var carrierName string
	var baseFee float64
	var perKmRate float64

	if senderCity == recipientCity || distance < 200.0 {
		carrierName = "SF Express"
		baseFee = 5.00
		perKmRate = 0.08
	} else {
		carrierName = "DHL Express"
		baseFee = 12.00
		perKmRate = 0.15
	}

	// Calculate 3PL fee based on distance + weight surcharge
	fee := baseFee + (distance * perKmRate) + (weightKg * 1.50)
	return &RoutingDecision{
		CarrierType: "3pl",
		CarrierName: carrierName,
		ShippingFee: math.Round(fee*100) / 100,
		DistanceKm:  distance,
	}
}
