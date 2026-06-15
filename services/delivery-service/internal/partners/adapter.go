package partners

type ShipmentInfo struct {
	ExternalTrackingNo string
	CarrierName        string
}

type CarrierAdapter interface {
	RegisterShipment(senderName, senderAddress, recipientName, recipientAddress string, weight float64) (*ShipmentInfo, error)
}
