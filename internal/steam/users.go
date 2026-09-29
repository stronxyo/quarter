package steam

import (
	"errors"
	"path/filepath"
	"strconv"

	"github.com/stronxyo/quarter/internal/vdf"
)

// ErrNoAccount means no account has signed in to Steam on this PC.
var ErrNoAccount = errors.New("loginusers.vdf: no accounts")

// Account identifies a Steam account that has logged in on this machine.
type Account struct {
	SteamID64 uint64
	AccountID uint32 // names the account's userdata folder
}

// ActiveAccount returns the account most recently logged in on this machine.
func ActiveAccount(steamPath string) (Account, error) {
	root, err := readVDF(filepath.Join(steamPath, "config", "loginusers.vdf"))
	if err != nil {
		return Account{}, err
	}
	return pickAccount(root)
}

// pickAccount prefers the account flagged MostRecent. Recent Steam clients no
// longer write that flag, so otherwise it takes the latest login Timestamp.
func pickAccount(root *vdf.Node) (Account, error) {
	var best *vdf.Node
	var bestTime int64 = -1
	for _, u := range root.Get("users").Entries() {
		if _, err := strconv.ParseUint(u.Key, 10, 64); err != nil {
			continue
		}
		if u.ValueOf("MostRecent") == "1" {
			best = u
			break
		}
		if ts := parseInt(u.ValueOf("Timestamp")); ts > bestTime {
			best, bestTime = u, ts
		}
	}
	if best == nil {
		return Account{}, ErrNoAccount
	}

	id64, _ := strconv.ParseUint(best.Key, 10, 64)
	return Account{SteamID64: id64, AccountID: uint32(id64 & 0xFFFFFFFF)}, nil
}
