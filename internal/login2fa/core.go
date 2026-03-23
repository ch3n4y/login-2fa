package login2fa

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/denisbrodbeck/machineid"
)

const (
	DefaultStep       = 30
	DefaultDigits     = 6
	DefaultWindow     = 1
	MachineCodeLength = 16
)

var DefaultMasterKeyPath = "/etc/security/login-2fa.key"

func CollectMachineMaterial() map[string]string {
	protectedID, err := machineid.ProtectedID("login-2fa")
	if err != nil {
		return map[string]string{}
	}
	return map[string]string{
		"machine_id": FormatMachineCode(protectedID),
		"source":     "github.com/denisbrodbeck/machineid",
	}
}

func ComputeMachineCode(material map[string]string) string {
	for _, key := range []string{"machine_id", "id"} {
		if value := strings.TrimSpace(material[key]); value != "" {
			return FormatMachineCode(value)
		}
	}
	return ""
}

func LocalMachineCode() (string, map[string]string) {
	material := CollectMachineMaterial()
	if value := ComputeMachineCode(material); value != "" {
		return value, material
	}
	id, err := machineid.ProtectedID("login-2fa")
	if err != nil {
		return "", material
	}
	code := FormatMachineCode(id)
	material["machine_id"] = code
	return code, material
}

func GenerateCode(masterKey, machineCode string, timestamp int64, step, digits int) (string, error) {
	if strings.TrimSpace(masterKey) == "" {
		return "", errors.New("master key is required")
	}
	machineCode = NormalizeMachineCode(machineCode)
	if machineCode == "" {
		return "", errors.New("machine code is required")
	}
	if step <= 0 {
		step = DefaultStep
	}
	if digits < 4 || digits > 10 {
		return "", errors.New("digits must be between 4 and 10")
	}

	secret := deriveSecret(masterKey, machineCode)
	counter := uint64(timestamp / int64(step))
	msg := []byte{
		byte(counter >> 56), byte(counter >> 48), byte(counter >> 40), byte(counter >> 32),
		byte(counter >> 24), byte(counter >> 16), byte(counter >> 8), byte(counter),
	}
	mac := hmac.New(sha1.New, secret)
	_, _ = mac.Write(msg)
	sum := mac.Sum(nil)
	offset := int(sum[len(sum)-1] & 0x0f)
	binary := (uint32(sum[offset])&0x7f)<<24 |
		uint32(sum[offset+1])<<16 |
		uint32(sum[offset+2])<<8 |
		uint32(sum[offset+3])
	mod := uint32(1)
	for range digits {
		mod *= 10
	}
	code := binary % mod
	return fmt.Sprintf("%0*d", digits, code), nil
}

func VerifyCode(code, masterKey, machineCode string, timestamp int64, step, digits, window int) (bool, error) {
	machineCode = NormalizeMachineCode(machineCode)
	if window < 0 {
		window = DefaultWindow
	}
	for offset := -window; offset <= window; offset++ {
		current := timestamp + int64(offset*step)
		expected, err := GenerateCode(masterKey, machineCode, current, step, digits)
		if err != nil {
			return false, err
		}
		if hmac.Equal([]byte(code), []byte(expected)) {
			return true, nil
		}
	}
	return false, nil
}

func ValidateBuiltins() error {
	if DefaultStep <= 0 {
		return errors.New("DefaultStep must be > 0")
	}
	if DefaultDigits < 4 || DefaultDigits > 10 {
		return errors.New("DefaultDigits must be between 4 and 10")
	}
	if DefaultWindow < 0 {
		return errors.New("DefaultWindow must be >= 0")
	}
	return nil
}

func readTrimmed(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func ResolveMasterKey() (string, error) {
	if value := strings.TrimSpace(os.Getenv("LOGIN2FA_MASTER_KEY")); value != "" {
		return value, nil
	}

	for _, path := range candidateKeyPaths() {
		value := readTrimmed(path)
		if value != "" {
			return value, nil
		}
	}

	return "", fmt.Errorf("master key not found; checked env LOGIN2FA_MASTER_KEY and key files")
}

func candidateKeyPaths() []string {
	var paths []string
	seen := map[string]bool{}

	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" || seen[path] {
			return
		}
		seen[path] = true
		paths = append(paths, path)
	}

	if value := os.Getenv("LOGIN2FA_MASTER_KEY_FILE"); value != "" {
		add(value)
	}

	add(DefaultMasterKeyPath)
	add("login-2fa.key")

	if exePath, err := os.Executable(); err == nil {
		add(filepath.Join(filepath.Dir(exePath), "login-2fa.key"))
	}

	add("/lib/security/login-2fa.key")
	add("/lib/x86_64-linux-gnu/security/login-2fa.key")
	add("/usr/lib/security/login-2fa.key")
	add("/usr/lib/x86_64-linux-gnu/security/login-2fa.key")

	return paths
}

func deriveSecret(masterKey, machineCode string) []byte {
	payload := []byte("login-2fa:" + machineCode)
	mac := hmac.New(sha256.New, []byte(masterKey))
	_, _ = mac.Write(payload)
	return mac.Sum(nil)
}

func NormalizeMachineCode(machineCode string) string {
	var builder strings.Builder
	for _, char := range strings.ToUpper(machineCode) {
		if (char >= '0' && char <= '9') || (char >= 'A' && char <= 'F') {
			builder.WriteRune(char)
		}
	}
	value := builder.String()
	if len(value) > MachineCodeLength {
		value = value[:MachineCodeLength]
	}
	return value
}

func FormatMachineCode(machineCode string) string {
	value := NormalizeMachineCode(machineCode)
	if value == "" {
		return ""
	}
	var parts []string
	for len(value) > 0 {
		size := 4
		if len(value) < size {
			size = len(value)
		}
		parts = append(parts, value[:size])
		value = value[size:]
	}
	return strings.Join(parts, "-")
}

func NowUnix() int64 {
	return time.Now().Unix()
}
