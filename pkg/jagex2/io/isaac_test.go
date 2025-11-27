package io

import (
	"testing"
)

func TestNew(t *testing.T) {
	type args struct {
		seed [4]int32
	}
	tests := []struct {
		name string
		args args
		want int32
	}{
		{
			// checks that isaac is shuffling correctly
			name: "seed(0, 0, 0, 0)",
			args: args{
				seed: [4]int32{0, 0, 0, 0},
			},
			want: 1536048213,
		},
		{
			// checks that rsl was populated and that isaac is shuffling correctly
			name: "seed(1, 2, 3, 4)",
			args: args{
				seed: [4]int32{1, 2, 3, 4},
			},
			want: -107094133,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := NewIsaac(tt.args.seed)
			for range 1_000_000 {
				is.TakeNextValue()
			}

			if got := is.TakeNextValue(); got != tt.want {
				t.Errorf("TakeNextValue() = %v, want %v", got, tt.want)
			}
		})
	}
}
