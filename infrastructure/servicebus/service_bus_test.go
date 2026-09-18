package servicebus

import (
	"context"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
)

func TestNewServiceBus(t *testing.T) {
	azServiceBusClient, _ := NewServiceBus(
		context.Background(),
		"gra-sb-gateway.servicebus.windows.net",
	)
	type args struct {
		ctx       context.Context
		namespace string
	}
	tests := []struct {
		name    string
		args    args
		want    *azservicebus.Client
		wantErr bool
	}{
		{
			name: "TestNewServiceBus - 1",
			args: args{
				ctx:       context.Background(),
				namespace: "gra-sb-gateway.servicebus.windows.net",
			},
			want:    azServiceBusClient,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewServiceBus(tt.args.ctx, tt.args.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewServiceBus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// if !reflect.DeepEqual(got, tt.want) {
			// 	t.Errorf("NewServiceBus() = %v, want %v", got, tt.want)
			// }
		})
	}
}
