package ble

type AuthData struct {
	// Passkey is the numeric passkey for pairing.
	Passkey int

	// PasskeyFn is a function that returns the passkey for pairing.
	// If Passkey is not set, this function will be called to
	// retrieve the passkey.
	PasskeyFn func() int

	// OOBData is the out-of-band data for pairing.
	OOBData []byte
}

// GetPasskey returns the passkey for pairing. If Passkey is not set,
// it calls PasskeyFn to populate it.
func (a *AuthData) GetPasskey() int {
	if a.Passkey != 0 {
		return a.Passkey
	}
	if a.PasskeyFn != nil {
		a.Passkey = a.PasskeyFn()
	}
	return a.Passkey
}
