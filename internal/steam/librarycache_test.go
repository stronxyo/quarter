package steam

import "testing"

func TestReadAchievementsFixture(t *testing.T) {
	got, err := ReadAchievements(fixture, Account{AccountID: 2})
	if err != nil {
		t.Fatalf("ReadAchievements: %v", err)
	}
	if a := got[620]; a.Achieved != 20 || a.Total != 51 {
		t.Errorf("620: got %+v, want 20/51", a)
	}
	for _, id := range []uint32{400, 70} {
		if _, ok := got[id]; ok {
			t.Errorf("%d has no achievements (none, or a broken file) and should be absent", id)
		}
	}
	if len(got) != 1 {
		t.Errorf("want only 620, got %v", got)
	}
}

func TestReadAchievementsNoFolder(t *testing.T) {
	got, err := ReadAchievements(fixture, Account{AccountID: 99})
	if err != nil || len(got) != 0 {
		t.Errorf("a missing folder means no achievements, not an error: %v %v", got, err)
	}
}

func TestParseLibraryPage(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want Achievements
		ok   bool
	}{
		{"normal", `[["badge",{}],["achievements",{"data":{"nTotal":10,"nAchieved":3}}]]`, Achievements{3, 10}, true},
		{"all done", `[["achievements",{"data":{"nTotal":5,"nAchieved":5}}]]`, Achievements{5, 5}, true},
		{"more achieved than total is capped", `[["achievements",{"data":{"nTotal":5,"nAchieved":9}}]]`, Achievements{5, 5}, true},
		{"no achievements section", `[["badge",{}]]`, Achievements{}, false},
		{"zero total", `[["achievements",{"data":{"nTotal":0}}]]`, Achievements{}, false},
		{"odd pair", `[["achievements"]]`, Achievements{}, false},
		{"not a list", `{"achievements":1}`, Achievements{}, false},
	}
	for _, tt := range tests {
		got, ok := parseLibraryPage([]byte(tt.in))
		if ok != tt.ok || got != tt.want {
			t.Errorf("%s: got %+v %v, want %+v %v", tt.name, got, ok, tt.want, tt.ok)
		}
	}
}
