package partners

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

type SFExpressAdapter struct{}

func NewSFExpressAdapter() *SFExpressAdapter {
	return &SFExpressAdapter{}
}

func (a *SFExpressAdapter) RegisterShipment(senderName, senderAddress, recipientName, recipientAddress string, weight float64) (*ShipmentInfo, error) {
	nBig, err := rand.Int(rand.Reader, big.NewInt(9000000000))
	var num int64
	if err != nil {
		num = 1408219082
	} else {
		num = nBig.Int64() + 1000000000
	}
	return &ShipmentInfo{
		ExternalTrackingNo: fmt.Sprintf("SF%d", num),
		CarrierName:        "SF Express",
	}, nil
}
