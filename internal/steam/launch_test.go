package steam

import "testing"

func TestURIs(t *testing.T) {
	tests := []struct {
		got, want string
	}{
		{PlayURI(620), "steam://rungameid/620"},
		{InstallURI(620), "steam://install/620"},
		{StoreURI(620), "steam://store/620"},
		{UninstallURI(620), "steam://uninstall/620"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("got %q, want %q", tt.got, tt.want)
		}
	}
}
