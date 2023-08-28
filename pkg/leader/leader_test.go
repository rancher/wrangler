package leader

import (
	"os"
	"reflect"
	"testing"
	"time"

	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

func Test_computeConfig(t *testing.T) {
	type args struct {
		rl  resourcelock.Interface
		cbs leaderelection.LeaderCallbacks
	}
	type env struct {
		key   string
		value string
	}
	tests := []struct {
		name    string
		args    args
		envs    []env
		want    *leaderelection.LeaderElectionConfig
		wantErr bool
	}{
		{
			name: "all defaults",
			args: args{
				rl:  nil,
				cbs: leaderelection.LeaderCallbacks{},
			},
			envs: []env{},
			want: &leaderelection.LeaderElectionConfig{
				Lock:            nil,
				LeaseDuration:   defaultLeaseDuration,
				RenewDeadline:   defaultRenewDeadline,
				RetryPeriod:     defaultRetryPeriod,
				Callbacks:       leaderelection.LeaderCallbacks{},
				ReleaseOnCancel: true,
			},
			wantErr: false,
		},
		{
			name: "dev mode",
			args: args{
				rl:  nil,
				cbs: leaderelection.LeaderCallbacks{},
			},
			envs: []env{
				{key: "CATTLE_DEV_MODE", value: "true"},
			},
			want: &leaderelection.LeaderElectionConfig{
				Lock:            nil,
				LeaseDuration:   developmentLeaseDuration,
				RenewDeadline:   developmentRenewDeadline,
				RetryPeriod:     defaultRetryPeriod,
				Callbacks:       leaderelection.LeaderCallbacks{},
				ReleaseOnCancel: true,
			},
			wantErr: false,
		},
		{
			name: "all overridden",
			args: args{
				rl:  nil,
				cbs: leaderelection.LeaderCallbacks{},
			},
			envs: []env{
				{key: "CATTLE_DEV_MODE", value: "true"},
				{key: "CATTLE_ELECTION_LEASE_DURATION", value: "1s"},
				{key: "CATTLE_ELECTION_RENEW_DEADLINE", value: "2s"},
				{key: "CATTLE_ELECTION_RETRY_PERIOD", value: "3m"},
			},
			want: &leaderelection.LeaderElectionConfig{
				Lock:            nil,
				LeaseDuration:   time.Second,
				RenewDeadline:   2 * time.Second,
				RetryPeriod:     3 * time.Minute,
				Callbacks:       leaderelection.LeaderCallbacks{},
				ReleaseOnCancel: true,
			},
			wantErr: false,
		},
		{
			name: "unparseable lease duration",
			args: args{
				rl:  nil,
				cbs: leaderelection.LeaderCallbacks{},
			},
			envs: []env{
				{key: "CATTLE_ELECTION_LEASE_DURATION", value: "bomb"},
				{key: "CATTLE_ELECTION_RENEW_DEADLINE", value: "2s"},
				{key: "CATTLE_ELECTION_RETRY_PERIOD", value: "3m"},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "unparseable renew deadline",
			args: args{
				rl:  nil,
				cbs: leaderelection.LeaderCallbacks{},
			},
			envs: []env{
				{key: "CATTLE_ELECTION_LEASE_DURATION", value: "1s"},
				{key: "CATTLE_ELECTION_RENEW_DEADLINE", value: "bomb"},
				{key: "CATTLE_ELECTION_RETRY_PERIOD", value: "3m"},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "unparseable retry period",
			args: args{
				rl:  nil,
				cbs: leaderelection.LeaderCallbacks{},
			},
			envs: []env{
				{key: "CATTLE_ELECTION_LEASE_DURATION", value: "1s"},
				{key: "CATTLE_ELECTION_RENEW_DEADLINE", value: "2s"},
				{key: "CATTLE_ELECTION_RETRY_PERIOD", value: "bomb"},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, e := range tt.envs {
				err := os.Setenv(e.key, e.value)
				if err != nil {
					t.Errorf("could not SetEnv: %v", err)
					return
				}
			}
			got, err := computeConfig(tt.args.rl, tt.args.cbs)
			if (err != nil) != tt.wantErr {
				t.Errorf("computeConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("computeConfig() got = %v, want %v", got, tt.want)
			}
		})
	}
}
