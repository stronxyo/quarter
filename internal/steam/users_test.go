package steam

import (
	"strings"
	"testing"

	"github.com/stronxyo/quarter/internal/vdf"
)

func TestActiveAccountFixture(t *testing.T) {
	acct, err := ActiveAccount(fixture)
	if err != nil {
		t.Fatalf("ActiveAccount: %v", err)
	}
	if acct.SteamID64 != 76561197960265730 || acct.AccountID != 2 {
		t.Errorf("got %+v, want the account with the latest Timestamp (id 2)", acct)
	}
}

func TestPickAccount(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    uint32
		wantErr bool
	}{
		{
			name: "MostRecent wins over a later Timestamp",
			in:   `"users" { "76561197960265729" { "MostRecent" "1" "Timestamp" "1" } "76561197960265730" { "Timestamp" "999" } }`,
			want: 1,
		},
		{
			name: "latest Timestamp without MostRecent",
			in:   `"users" { "76561197960265729" { "Timestamp" "999" } "76561197960265730" { "Timestamp" "1" } }`,
			want: 1,
		},
		{
			name: "single account without Timestamp",
			in:   `"users" { "76561197960265735" { "PersonaName" "x" } }`,
			want: 7,
		},
		{
			name: "non-numeric keys skipped",
			in:   `"users" { "junk" { "MostRecent" "1" } "76561197960265730" { } }`,
			want: 2,
		},
		{name: "no accounts", in: `"users" { }`, wantErr: true},
		{name: "no users block", in: `"other" { }`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, err := vdf.Parse(strings.NewReader(tt.in))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			acct, err := pickAccount(root)
			if tt.wantErr {
				if err == nil {
					t.Fatal("want an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("pickAccount: %v", err)
			}
			if acct.AccountID != tt.want {
				t.Errorf("got account %d, want %d", acct.AccountID, tt.want)
			}
		})
	}
}
