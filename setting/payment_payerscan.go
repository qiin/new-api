package setting

// PayerScan hosted crypto checkout configuration. The gateway is enabled once
// the operator turns PayerScanEnabled on and both credentials are populated;
// PayerScanBaseURL stays empty for the documented production host.
var (
	PayerScanEnabled    bool
	PayerScanMerchantID string
	PayerScanApiKey     string
	PayerScanBaseURL    string
	PayerScanUnitPrice  float64 = 1.0
	PayerScanMinTopUp   int     = 1
)
