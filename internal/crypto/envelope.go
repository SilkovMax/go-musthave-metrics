package crypto

// Envelope используется для гибридного шифрования (AES + RSA)
type Envelope struct {
	EncryptedKey  string `json:"encrypted_key"`
	EncryptedData string `json:"encrypted_data"`
}
