package cam

import "testing"

func TestFormat(t *testing.T) {

	var camStats CamStats
	camStats.currentTime = 100
	camStats.duration = 200
	camStats.speed = 1

	formattedCamStats := camStats.Format()

	expected := "100.0/200.0 | Speed: 1.00x"

	if formattedCamStats != expected {
		t.Errorf("wrong format, expected %s got %s", expected, formattedCamStats)
	}

}

func TestIncreaseSpeed(t *testing.T) {
	var camStats CamStats

	tests := []struct {
		startSpeed float64
		finalSpeed float64
	}{
		{0, 1},
		{2, 4},
		{50, 64},
	}

	for _, tt := range tests {

		camStats.speed = tt.startSpeed
		camStats.IncreaseSpeed()

		if camStats.speed != tt.finalSpeed {
			t.Errorf("error IncreaseSpeed, expected %f got %f", tt.finalSpeed, camStats.speed)
		}
	}

}

func TestDecreaseSpeed(t *testing.T) {
	var camStats CamStats

	tests := []struct {
		startSpeed float64
		finalSpeed float64
	}{
		{2, 1},
		{0.25, 0.25},
		{64, 32},
	}

	for _, tt := range tests {

		camStats.speed = tt.startSpeed
		camStats.DecreaseSpeed()

		if camStats.speed != tt.finalSpeed {
			t.Errorf("error DecreaseSpeed, expected %f got %f", tt.finalSpeed, camStats.speed)
		}
	}

}
