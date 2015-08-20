package models

import "testing"

func TestValidStages(t *testing.T) {
	availableStages := []DeploymentStage{"ONE", "TWO", "THREE", "FOUR"}
	target := &Target{AvailableStages: availableStages}

	tests := []struct {
		stages         []DeploymentStage
		expectedResult bool
	}{
		{[]DeploymentStage{"ONE", "TWO", "THREE"}, true},
		{[]DeploymentStage{"ONE", "TWO"}, true},
		{[]DeploymentStage{"TWO", "THREE"}, true},
		{[]DeploymentStage{"ONE"}, true},
		{[]DeploymentStage{"TWO"}, true},
		{[]DeploymentStage{"THREE"}, true},
		{[]DeploymentStage{"THREE", "TWO", "ONE"}, false},
		{[]DeploymentStage{"TWO", "ONE"}, false},
		{[]DeploymentStage{"THREE", "ONE"}, false},
		{[]DeploymentStage{"THREE", "TWO"}, false},
		{[]DeploymentStage{"NON-EXISTANT"}, false},
		{[]DeploymentStage{"ONE", "TWO", "NON-EXISTANT"}, false},
		{[]DeploymentStage{"ONE", "NON-EXISTANT", "TWO", "THREE"}, false},
	}

	for _, test := range tests {
		result := target.AreValidStages(test.stages)
		if result != test.expectedResult {
			t.Errorf("expected to be %v, got %v (given=%+v)", test.expectedResult, result, test.stages)
		}
	}
}
