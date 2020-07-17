package smp

const (
	pairingRequest          = 0x01 // Pairing Request LE-U, ACL-U
	pairingResponse         = 0x02 // Pairing Response LE-U, ACL-U
	pairingConfirm          = 0x03 // Pairing Confirm LE-U
	pairingRandom           = 0x04 // Pairing Random LE-U
	pairingFailed           = 0x05 // Pairing Failed LE-U, ACL-U
	encryptionInformation   = 0x06 // Encryption Information LE-U
	masterIdentification    = 0x07 // Master Identification LE-U
	identityInformation     = 0x08 // Identity Information LE-U, ACL-U
	identityAddrInformation = 0x09 // Identity Address Information LE-U, ACL-U
	signingInformation      = 0x0A // Signing Information LE-U, ACL-U
	securityRequest         = 0x0B // Security Request LE-U
	pairingPublicKey        = 0x0C // Pairing Public Key LE-U
	pairingDHKeyCheck       = 0x0D // Pairing DHKey Check LE-U
	pairingKeypress         = 0x0E // Pairing Keypress Notification LE-U

	passkeyIterationCount = 20

	oobData
	oobDataPreset = 0x01

	authReqBondMask = byte(0x03)
	authReqBond     = byte(0x01)
	authReqNoBond   = byte(0x00)
)
