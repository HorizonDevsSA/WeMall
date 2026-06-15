package routing

import (
	"math"
	"testing"
)

func TestRoutePackage(t *testing.T) {
	// Case 1: Crowdsourced Courier criteria (same city, under 15km, under 10kg, under 125,000 cm³)
	// Origin: Shenzhen Civic Center (22.5430, 114.0578)
	// Destination: Shenzhen Futian Port (22.5080, 114.0620)
	// Distance: ~3.9 km
	dec1 := RoutePackage("Shenzhen", "Shenzhen", 22.5430, 114.0578, 22.5080, 114.0620, 2.5, 20, 15, 10)
	if dec1.CarrierType != "crowdsourced" {
		t.Errorf("Expected carrier type crowdsourced, got %s", dec1.CarrierType)
	}
	if dec1.CarrierName != "WeMall Instant Rider" {
		t.Errorf("Expected carrier WeMall Instant Rider, got %s", dec1.CarrierName)
	}
	expectedFee := 3.00 + (dec1.DistanceKm * 0.50)
	if math.Abs(dec1.ShippingFee - expectedFee) > 0.05 {
		t.Errorf("Expected shipping fee close to %.2f, got %.2f", expectedFee, dec1.ShippingFee)
	}

	// Case 2: 3PL Domestic (SF Express) - Over weight limit (12kg)
	dec2 := RoutePackage("Shenzhen", "Shenzhen", 22.5430, 114.0578, 22.5080, 114.0620, 12.0, 20, 15, 10)
	if dec2.CarrierType != "3pl" {
		t.Errorf("Expected carrier type 3pl for overweight, got %s", dec2.CarrierType)
	}
	if dec2.CarrierName != "SF Express" {
		t.Errorf("Expected carrier SF Express, got %s", dec2.CarrierName)
	}

	// Case 3: 3PL International (DHL Express) - Different cities far apart
	// Origin: Shenzhen (22.5430, 114.0578)
	// Destination: Beijing (39.9042, 116.4074)
	dec3 := RoutePackage("Shenzhen", "Beijing", 22.5430, 114.0578, 39.9042, 116.4074, 5.0, 30, 25, 20)
	if dec3.CarrierType != "3pl" {
		t.Errorf("Expected carrier type 3pl for far distance, got %s", dec3.CarrierType)
	}
	if dec3.CarrierName != "DHL Express" {
		t.Errorf("Expected carrier DHL Express, got %s", dec3.CarrierName)
	}
}
