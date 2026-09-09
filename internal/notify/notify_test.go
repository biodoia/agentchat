package notify

import (
	"testing"
)

func TestDesktopDoesNotPanic(t *testing.T) {
	// notify-send may not be available in CI; just verify no panic
	err := Desktop("Test", "body", "", "normal")
	// err may be non-nil if notify-send not installed; that's OK
	_ = err
}

func TestChatMessageDoesNotPanic(t *testing.T) {
	err := ChatMessage("TestBot", "hello", "#FF0000")
	_ = err
}

func TestStealFocusDoesNotPanic(t *testing.T) {
	err := StealFocus("UrgentBot", "attention!")
	_ = err
}
