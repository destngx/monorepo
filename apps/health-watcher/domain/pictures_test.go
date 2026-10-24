package domain

import "testing"

func TestGivenPicturesFromTheModelWhenCleanedThenOnlyLargeBoxesOnSentPagesStayClippedToThePage(t *testing.T) {
	out := ReadingOutput{Pictures: []ReadingPicture{
		{Page: 1, Kind: "ultrasound", Caption: "  Gan  ", X0: 0.6, Y0: -0.1, X1: 0.1, Y1: 0.5},
		{Page: 1, Kind: "logo", X0: 0.1, Y0: 0.1, X1: 0.13, Y1: 0.3},
		{Page: 3, Kind: "xray", X0: 0.1, Y0: 0.1, X1: 0.9, Y1: 0.9},
		{Page: 2, Kind: "hologram", X0: 0.1, Y0: 0.1, X1: 0.9, Y1: 1.4},
	}}
	got := out.CleanPictures(2)
	want := []PictureInput{
		{Page: 1, Kind: "ultrasound", Caption: "Gan", Box: Rect{X0: 0.1, Y0: 0, X1: 0.6, Y1: 0.5}},
		{Page: 2, Kind: "other", Box: Rect{X0: 0.1, Y0: 0.1, X1: 0.9, Y1: 1}},
	}
	if len(got) != len(want) {
		t.Fatalf("pictures = %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("picture %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}
