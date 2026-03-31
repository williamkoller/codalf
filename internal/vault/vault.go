package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	vaultDir  = ".codalf"
	vaultFile = "vault.json"
)

type Config struct {
	Provider      string    `json:"provider"`
	Model         string    `json:"model"`
	OllamaHost    string    `json:"ollamaHost"`
	OpenAIKey     string    `json:"openAIKey,omitempty"`
	AnthropicKey  string    `json:"anthropicKey,omitempty"`
	CopilotToken  string    `json:"copilotToken,omitempty"`
	OpenAIBaseURL string    `json:"openAIBaseURL,omitempty"`
	AnthropicURL  string    `json:"anthropicURL,omitempty"`
	CopilotURL    string    `json:"copilotURL,omitempty"`
	EncryptionKey string    `json:"encryptionKey,omitempty"`
	Offline       bool      `json:"offline"`
	CreatedAt     time.Time `json:"createdAt"`
	Checksum      string    `json:"checksum"`
}

func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	return filepath.Join(home, vaultDir, vaultFile), nil
}

func Exists() bool {
	p, err := Path()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

func Save(cfg *Config) error {
	p, err := Path()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return fmt.Errorf("could not create vault directory: %w", err)
	}

	cfg.Checksum = ""
	body, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("could not marshal config: %w", err)
	}

	sum := sha256.Sum256(body)
	cfg.Checksum = hex.EncodeToString(sum[:])

	final, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal config: %w", err)
	}

	if err := os.WriteFile(p, final, 0600); err != nil {
		return fmt.Errorf("could not write vault: %w", err)
	}

	return nil
}

func Load() (*Config, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("vault not found — run 'codalf init' first")
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("vault is corrupted: %w", err)
	}

	if err := verify(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func verify(cfg *Config) error {
	stored := cfg.Checksum
	cfg.Checksum = ""

	body, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("could not verify vault: %w", err)
	}

	sum := sha256.Sum256(body)
	expected := hex.EncodeToString(sum[:])

	cfg.Checksum = stored

	if stored != expected {
		return fmt.Errorf("vault checksum mismatch — file may have been tampered with")
	}

	return nil
}

// ValidateHost ensures the Ollama host is local only (no internet).
func ValidateHost(host string) error {
	allowed := []string{"localhost", "127.0.0.1", "::1"}
	for _, a := range allowed {
		if strings.Contains(host, a) {
			return nil
		}
	}
	return fmt.Errorf("vault only allows local Ollama hosts (localhost/127.0.0.1), got: %s", host)
}

func EncryptAPIKey(key, password string) (string, error) {
	if key == "" {
		return "", nil
	}
	if password == "" {
		return key, nil
	}

	block, err := aes.NewCipher([]byte(password[:32]))
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to read nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(key), nil)
	return hex.EncodeToString(ciphertext), nil
}

func DecryptAPIKey(encryptedKey, password string) (string, error) {
	if encryptedKey == "" {
		return "", nil
	}
	if password == "" {
		return encryptedKey, nil
	}

	data, err := hex.DecodeString(encryptedKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode key: %w", err)
	}

	block, err := aes.NewCipher([]byte(password[:32]))
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}
