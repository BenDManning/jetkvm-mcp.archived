package jetkvm

import (
	"context"
	"encoding/json"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/BenDManning/jetkvm-mcp/internal/mcpserver"
)

type discoveryProvider struct {
	calls atomic.Int32
}

func (provider *discoveryProvider) WithSession(context.Context, DeviceConfig, SessionProfile, func(Session) error) error {
	provider.calls.Add(1)
	return nil
}

func TestManagerListsSortedConfiguredDevicesWithoutPrivateConfiguration(t *testing.T) {
	const private = "JETKVM-PRIVATE-DISCOVERY-SENTINEL-4d92e1"
	alphaURL, err := url.Parse("https://alpha.invalid")
	if err != nil {
		t.Fatal(err)
	}
	zetaURL, err := url.Parse("https://" + strings.ToLower(private) + ".invalid/private/" + private)
	if err != nil {
		t.Fatal(err)
	}
	provider := new(discoveryProvider)
	manager, err := NewManager([]DeviceConfig{
		{
			Name: "zeta", BaseURL: *zetaURL, Password: private,
			MediaDirectory:         "/private/" + private,
			MediaURLAllowedOrigins: []string{"https://media.invalid"},
			WakeOnLAN: map[string]WakeOnLANTarget{
				private: {MACAddress: "02:00:00:00:00:01", BroadcastIP: "192.0.2.255"},
			},
		},
		{Name: "alpha", BaseURL: *alphaURL},
	}, provider)
	if err != nil {
		t.Fatal(err)
	}

	got, err := manager.ListDevices(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := mcpserver.DeviceList{Devices: []mcpserver.ConfiguredDevice{
		{Device: "alpha", Capabilities: mcpserver.DeviceCapabilities{}},
		{Device: "zeta", Capabilities: mcpserver.DeviceCapabilities{
			MountVirtualMediaURL:   true,
			MountVirtualMediaFile:  true,
			UploadVirtualMediaFile: true,
			WakeHostLAN:            true,
		}},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("devices = %#v, want %#v", got, want)
	}
	if calls := provider.calls.Load(); calls != 0 {
		t.Fatalf("provider calls = %d, want none", calls)
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{private, strings.ToLower(private), "alpha.invalid", "media.invalid", "192.0.2.255", "02:00:00:00:00:01", "/private/"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("device discovery exposed %q: %s", forbidden, encoded)
		}
	}
}
