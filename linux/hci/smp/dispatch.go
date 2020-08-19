package smp

var dispatcher = map[byte]smpDispatcher{
	//pairingRequest:          smpDispatcher{"pairing request", smpOnPairingRequest},
	pairingResponse:         smpDispatcher{"pairing response", smpOnPairingResponse},
	pairingConfirm:          smpDispatcher{"pairing confirm", smpOnPairingConfirm},
	pairingRandom:           smpDispatcher{"pairing random", smpOnPairingRandom},
	pairingFailed:           smpDispatcher{"pairing failed", smpOnPairingFailed},
	encryptionInformation:   smpDispatcher{"encryption info", smpOnEncryptionInformation},
	masterIdentification:    smpDispatcher{"master id", smpOnMasterIdentification},
	identityInformation:     smpDispatcher{"id info", nil},
	identityAddrInformation: smpDispatcher{"id addr info", nil},
	signingInformation:      smpDispatcher{"signing info", nil},
	securityRequest:         smpDispatcher{"security req", smpOnSecurityRequest},
	pairingPublicKey:        smpDispatcher{"pairing pub key", smpOnPairingPublicKey},
	pairingDHKeyCheck:       smpDispatcher{"pairing dhkey check", smpOnDHKeyCheck},
	pairingKeypress:         smpDispatcher{"pairing keypress", nil},
}

//Core spec v5.2, Vol 3, Part H, 3.5.5, Table 3.7
var pairingFailedReason = []string{
	"reserved",
	"passkey entry failed",
	"oob not available",
	"authentication requirements",
	"confirm value failed",
	"pairing not support",
	"encryption key size",
	"command not supported",
	"unspecified reason",
	"repeated attempts",
	"invalid parameters",
	"dhkey check failed",
	"numeric comparaison failed",
	"BR/EDR pairing in progress",
	"Cross-transport Key Derivation/Generation not allowed",
}

type smpDispatcher struct {
	desc string
	//todo: only errors are returned from these functions
	//so the []byte return should be removed
	handler func(t *transport, p pdu) ([]byte, error)
}
