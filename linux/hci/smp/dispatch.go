package smp

var dispatcher = map[byte]smpDispatcher{
	//pairingRequest:        {"pairing request", smpOnPairingRequest},
	pairingResponse:         {"pairing response", smpOnPairingResponse},
	pairingConfirm:          {"pairing confirm", smpOnPairingConfirm},
	pairingRandom:           {"pairing random", smpOnPairingRandom},
	pairingFailed:           {"pairing failed", smpOnPairingFailed},
	encryptionInformation:   {"encryption info", smpOnEncryptionInformation},
	masterIdentification:    {"master id", smpOnMasterIdentification},
	identityInformation:     {"id info", nil},
	identityAddrInformation: {"id addr info", nil},
	signingInformation:      {"signing info", nil},
	securityRequest:         {"security req", smpOnSecurityRequest},
	pairingPublicKey:        {"pairing pub key", smpOnPairingPublicKey},
	pairingDHKeyCheck:       {"pairing dhkey check", smpOnDHKeyCheck},
	pairingKeypress:         {"pairing keypress", nil},
}

//Core spec v5.0, Vol 3, Part H, 3.5.5, Table 3.7
var pairingFailedReason = map[byte]string{
	0x0: "reserved",
	0x1: "passkey entry failed",
	0x2: "oob not available",
	0x3: "authentication requirements",
	0x4: "confirm value failed",
	0x5: "pairing not support",
	0x6: "encryption key size",
	0x7: "command not supported",
	0x8: "unspecified reason",
	0x9: "repeated attempts",
	0xa: "invalid parameters",
	0xb: "DHKey check failed",
	0xc: "numeric comparison failed",
	0xd: "BR/EDR pairing in progress",
	0xe: "cross-transport key derivation/generation not allowed",
}

type smpDispatcher struct {
	desc string
	//todo: only errors are returned from these functions
	//so the []byte return should be removed
	handler func(t *transport, p pdu) ([]byte, error)
}
