package steam

import "testing"

func TestReadLibraryPagesFixture(t *testing.T) {
	got, err := ReadLibraryPages(fixture, Account{AccountID: 2})
	if err != nil {
		t.Fatalf("ReadLibraryPages: %v", err)
	}
	if a := got[620]; a.Achieved != 20 || a.Total != 51 {
		t.Errorf("620: got %+v, want 20/51", a)
	}
	if a, ok := got[400]; !ok || a.Total != 0 {
		t.Errorf("400 has a page without achievements: got %+v %v", a, ok)
	}
	if _, ok := got[70]; ok {
		t.Error("70's page is broken and should be absent")
	}
	if len(got) != 2 {
		t.Errorf("want 620 and 400, got %v", got)
	}
}

func TestReadLibraryPagesNoFolder(t *testing.T) {
	got, err := ReadLibraryPages(fixture, Account{AccountID: 99})
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
		{"no achievements section", `[["badge",{}]]`, Achievements{}, true},
		{"zero total", `[["achievements",{"data":{"nTotal":0}}]]`, Achievements{}, true},
		{"odd pair", `[["achievements"]]`, Achievements{}, true},
		{"not a list", `{"achievements":1}`, Achievements{}, false},
	}
	for _, tt := range tests {
		got, ok := parseLibraryPage([]byte(tt.in))
		if ok != tt.ok || got != tt.want {
			t.Errorf("%s: got %+v %v, want %+v %v", tt.name, got, ok, tt.want, tt.ok)
		}
	}
}
