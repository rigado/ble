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

type smpDispatcher struct {
	desc    string
	handler func(t *transport, p pdu) ([]byte, error)
}
