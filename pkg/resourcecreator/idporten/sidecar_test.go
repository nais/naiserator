package idporten

import "testing"

func Test_getUnusedPort(t *testing.T) {
	type args struct {
		startingPort int
		usedPorts    []int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "happy go lucky",
			args: args{
				startingPort: 8080,
				usedPorts:    []int{80, 90},
			},
			want: 8080,
		},
		{
			name: "happy go long",
			args: args{
				startingPort: 8080,
				usedPorts:    []int{8080, 8081, 8082, 8083, 8084},
			},
			want: 8085,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getUnusedPort(tt.args.startingPort, tt.args.usedPorts...); got != tt.want {
				t.Errorf("getUnusedPort() = %v, want %v", got, tt.want)
			}
		})
	}
}
