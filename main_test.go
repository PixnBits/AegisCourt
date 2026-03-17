package main

import (
	"testing"
)

func TestDetectResources(t *testing.T) {
	resources := DetectResources()
	if resources.RAMFreeGB < 0 {
		t.Errorf("RAMFreeGB should be non-negative, got %f", resources.RAMFreeGB)
	}
	// Note: HasGPU and VRAMGB depend on system
	// RecommendedLLM and SuggestSequential based on RAM
	if resources.RAMFreeGB >= 9 {
		if resources.RecommendedLLM != "nemotron-3-nano" {
			t.Errorf("Expected nemotron-3-nano, got %s", resources.RecommendedLLM)
		}
		if resources.SuggestSequential {
			t.Errorf("Should not suggest sequential")
		}
	} else {
		if resources.RecommendedLLM != "llama3.2:latest" {
			t.Errorf("Expected llama3.2:latest, got %s", resources.RecommendedLLM)
		}
		if !resources.SuggestSequential {
			t.Errorf("Should suggest sequential")
		}
	}
}

// Basic test suggestion for signature (mocked):
// To test verifySelfSignature, you would need to mock os.Executable and the file read.
// Use a test binary with known hash, sign it with the private key, and embed the signature.
// For example:
// func TestVerifySelfSignature(t *testing.T) {
//     // Mock os.Executable to return a test file path
//     // Compute hash of test file, sign with private key, set binarySignature to that
//     // Then call verifySelfSignature, expect no panic
// }
// Since we can't easily mock here, this is a suggestion.