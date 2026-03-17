package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"AegisCourt/audit"
	"github.com/shirou/gopsutil/v3/mem"
)

var rootPublicKey ed25519.PublicKey
var binarySignature []byte

func init() {
	// Hard-coded root public key (32 bytes, 64 hex chars)
	pubHex := "d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a00" // Example generated key
	sigHex := "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000" // Dummy signature, will fail verification

	var err error
	rootPublicKey, err = hex.DecodeString(pubHex)
	if err != nil {
		panic(fmt.Sprintf("failed to decode public key: %v", err))
	}
	binarySignature, err = hex.DecodeString(sigHex)
	if err != nil {
		panic(fmt.Sprintf("failed to decode signature: %v", err))
	}
}

func verifySelfSignature() {
	exe, err := os.Executable()
	if err != nil {
		panic(fmt.Sprintf("failed to get executable path: %v", err))
	}
	file, err := os.Open(exe)
	if err != nil {
		panic(fmt.Sprintf("failed to open executable: %v", err))
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		panic(fmt.Sprintf("failed to hash executable: %v", err))
	}
	hash := hasher.Sum(nil)
	if !ed25519.Verify(rootPublicKey, hash, binarySignature) {
		panic("self-signature verification failed")
	}
	log.Println("Self-signature verified")
}

type Resources struct {
	RAMFreeGB        float64
	HasGPU           bool
	VRAMGB           float64
	RecommendedLLM   string
	SuggestSequential bool
}

func DetectResources() Resources {
	v, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("Error getting memory: %v", err)
		return Resources{}
	}
	ramFreeGB := float64(v.Available) / (1024 * 1024 * 1024)

	hasGPU := false
	vramGB := 0.0
	// Check nvidia-smi
	cmd := exec.Command("nvidia-smi", "--query-gpu=memory.free", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(lines) > 0 {
			if val, err := strconv.ParseFloat(strings.TrimSpace(lines[0]), 64); err == nil {
				vramGB = val / 1024 // MB to GB
				hasGPU = true
			}
		}
	}

	recommendedLLM := "nemotron-3-nano"
	suggestSequential := false
	if ramFreeGB < 9 {
		recommendedLLM = "llama3.2:3b-instruct"
		suggestSequential = true
	}

	return Resources{
		RAMFreeGB:        ramFreeGB,
		HasGPU:           hasGPU,
		VRAMGB:           vramGB,
		RecommendedLLM:   recommendedLLM,
		SuggestSequential: suggestSequential,
	}
}

func main() {
	verifySelfSignature()

	if len(os.Args) > 2 && os.Args[1] == "audit" && os.Args[2] == "verify" {
		intact, errs := audit.Verify()
		if intact {
			fmt.Println("Audit log is intact")
		} else {
			fmt.Println("Audit log is compromised:")
			for _, err := range errs {
				fmt.Println(err)
			}
		}
		return
	}

	resources := DetectResources()
	log.Printf("Resources: %+v", resources)

	// Log to audit
	if err := audit.Append(resources); err != nil {
		log.Printf("Failed to append to audit: %v", err)
	}
}