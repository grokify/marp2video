package orchestrator

import "testing"

func TestTotalStages(t *testing.T) {
	tests := []struct {
		name       string
		individual string
		avatar     *AvatarConfig
		want       int
	}{
		{"base", "", nil, 5},
		{"individual only", "/out", nil, 6},
		{"avatar only", "", &AvatarConfig{}, 6},
		{"individual and avatar", "/out", &AvatarConfig{}, 7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Orchestrator{config: Config{
				OutputIndividualDir: tt.individual,
				Avatar:              tt.avatar,
			}}
			if got := o.totalStages(); got != tt.want {
				t.Errorf("totalStages() = %d, want %d", got, tt.want)
			}
		})
	}
}
