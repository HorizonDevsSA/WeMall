package partners

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

type DHLAdapter struct{}

func NewDHLAdapter() *DHLAdapter {
	return &DHLAdapter{}
}

func (a *DHLAdapter) RegisterShipment(senderName, senderAddress, recipientName, recipientAddress string, weight float64) (*ShipmentInfo, error) {
	nBig, err := rand.Int(rand.Reader, big.NewInt(90000000))
	var num int64
	if err != nil {
		num = 48201908
	} else {
		num = nBig.Int64() + 10000000
	}
	return &ShipmentInfo{
		ExternalTrackingNo: fmt.Sprintf("DHL%d", num),
		CarrierName:        "DHL Express",
	}, nil
}
