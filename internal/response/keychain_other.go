//go:build response && !noresponse && !darwin && !linux

package response

import "fmt"

type osStore struct{}

func openOSStore() secretStore { return osStore{} }

func (osStore) Set(string, string, string) error {
	return fmt.Errorf("auto-apply keychain supports macOS and Linux")
}
func (osStore) Get(string) (string, string, error) {
	return "", "", fmt.Errorf("auto-apply keychain supports macOS and Linux")
}
func (osStore) Delete(string) error {
	return fmt.Errorf("auto-apply keychain supports macOS and Linux")
}
func (osStore) List() ([]string, error) {
	return nil, fmt.Errorf("auto-apply keychain supports macOS and Linux")
}
