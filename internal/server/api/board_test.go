package api

import (
	"encoding/json"
	"testing"
)

// darts-o-cam reads the game a dart or takeout went to from the answer (camapp.server.post).
func TestBoardAnswer(t *testing.T) {
	t.Parallel()
	got, err := json.Marshal(newBoardAnswer(7))
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"game":{"id":7}}`; string(got) != want {
		t.Errorf("answer %s, want %s", got, want)
	}
}
