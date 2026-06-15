package waybill

import (
	"encoding/base64"
	"fmt"
)

// GenerateEWaybill generates a standardized 100x150mm thermal waybill label layout in HTML, encoded in Base64.
func GenerateEWaybill(trackingNo string, senderName, senderAddress, recipientName, recipientAddress, carrierName, externalTrackingNo string, weight float64) string {
	htmlLabel := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
<style>
  .label-container {
    width: 380px;
    height: 570px;
    border: 2px solid #000;
    font-family: Arial, sans-serif;
    padding: 10px;
    box-sizing: border-box;
  }
  .header {
    border-bottom: 2px solid #000;
    padding-bottom: 5px;
    text-align: center;
    font-weight: bold;
    font-size: 16px;
  }
  .section {
    border-bottom: 1px solid #000;
    padding: 8px 0;
    font-size: 12px;
  }
  .barcode {
    text-align: center;
    padding: 15px 0;
    font-family: 'Courier New', Courier, monospace;
    font-size: 14px;
    letter-spacing: 3px;
  }
  .meta-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    font-size: 11px;
  }
</style>
</head>
<body>
<div class="label-container">
  <div class="header">
    WeMall Logistics - Cainiao Network
  </div>
  <div class="barcode">
    [|||||||||||||||||||||||||||||||||||]<br>
    *%s*
  </div>
  <div class="section">
    <strong>FROM:</strong> %s<br>
    <strong>ADDRESS:</strong> %s
  </div>
  <div class="section">
    <strong>TO:</strong> %s<br>
    <strong>ADDRESS:</strong> %s
  </div>
  <div class="section">
    <strong>CARRIER:</strong> %s<br>
    <strong>REF NO:</strong> %s
  </div>
  <div class="section" style="border-bottom: none;">
    <div class="meta-grid">
      <div><strong>WEIGHT:</strong> %.2f kg</div>
      <div><strong>ZONE:</strong> SEA-05-A</div>
    </div>
  </div>
</div>
</body>
</html>
`, trackingNo, senderName, senderAddress, recipientName, recipientAddress, carrierName, externalTrackingNo, weight)

	return base64.StdEncoding.EncodeToString([]byte(htmlLabel))
}
