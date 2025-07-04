package crypto

import (
	"bytes"
	"errors"
	"io"
	"strings"

	"filippo.io/age"
	kilnErrors "github.com/thunderbottom/kiln/internal/errors"
)

type AgeManager struct {
	recipients []age.Recipient
	identities []age.Identity
}

func NewAgeManager(publicKeys []string) (*AgeManager, error) {
	recipients := make([]age.Recipient, 0, len(publicKeys))

	for _, key := range publicKeys {
		recipient, err := age.ParseX25519Recipient(key)
		if err != nil {
			return nil, kilnErrors.Wrapf(err, "invalid public key: %s", key)
		}
		recipients = append(recipients, recipient)
	}

	return &AgeManager{
		recipients: recipients,
	}, nil
}

func (am *AgeManager) AddIdentity(privateKey string) error {
	identity, err := age.ParseX25519Identity(privateKey)
	if err != nil {
		return kilnErrors.Wrap(err, "invalid private key")
	}

	am.identities = append(am.identities, identity)
	return nil
}

func (am *AgeManager) Encrypt(data []byte) ([]byte, error) {
	if len(am.recipients) == 0 {
		return nil, errors.New("no recipients configured")
	}

	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, am.recipients...)
	if err != nil {
		return nil, kilnErrors.Wrap(err, "failed to create age writer")
	}

	if _, err := w.Write(data); err != nil {
		return nil, kilnErrors.Wrap(err, "failed to write data")
	}

	if err := w.Close(); err != nil {
		return nil, kilnErrors.Wrap(err, "failed to close age writer")
	}

	return buf.Bytes(), nil
}

func (am *AgeManager) Decrypt(data []byte) ([]byte, error) {
	if len(am.identities) == 0 {
		return nil, errors.New("no identities configured")
	}

	r, err := age.Decrypt(bytes.NewReader(data), am.identities...)
	if err != nil {
		return nil, kilnErrors.Wrap(err, "failed to decrypt data")
	}

	result, err := io.ReadAll(r)
	if err != nil {
		return nil, kilnErrors.Wrap(err, "failed to read decrypted data")
	}

	return result, nil
}

func GenerateKeyPair() (string, string, error) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return "", "", kilnErrors.Wrap(err, "failed to generate identity")
	}

	privateKey := identity.String()
	publicKey := identity.Recipient().String()

	return privateKey, publicKey, nil
}

func ValidatePublicKey(key string) error {
	key = strings.TrimSpace(key)
	if !strings.HasPrefix(key, "age1") {
		return errors.New("public key must start with 'age1'")
	}

	_, err := age.ParseX25519Recipient(key)
	if err != nil {
		return kilnErrors.Wrap(err, "invalid public key format")
	}

	return nil
}

func ValidatePrivateKey(key string) error {
	key = strings.TrimSpace(key)
	if !strings.HasPrefix(key, "AGE-SECRET-KEY-") {
		return errors.New("private key must start with 'AGE-SECRET-KEY-'")
	}

	_, err := age.ParseX25519Identity(key)
	if err != nil {
		return kilnErrors.Wrap(err, "invalid private key format")
	}

	return nil
}
