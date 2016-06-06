package statsd

import (
	"math/rand"
	"net"
	"runtime"
	"testing"
	"time"

	cloudTypes "github.com/atlassian/gostatsd/cloudprovider/types"
	"github.com/atlassian/gostatsd/tester/fakesocket"
	"github.com/atlassian/gostatsd/types"

	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"golang.org/x/time/rate"
)

// TestStatsdThroughput emulates statsd work using fake network socket and null backend to
// measure throughput.
func TestStatsdThroughput(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	var memStatsStart, memStatsFinish runtime.MemStats
	runtime.ReadMemStats(&memStatsStart)
	s := Server{
		Backends: []string{"null"},
		CloudProvider: &fakeProvider{
			instance: &cloudTypes.Instance{
				ID:     "i-13123123",
				Region: "us-west-3",
				Tags:   types.Tags{"tag1", "tag2:234"},
			},
		},
		Limiter:          rate.NewLimiter(DefaultMaxCloudRequests, DefaultBurstCloudRequests),
		DefaultTags:      DefaultTags,
		ExpiryInterval:   DefaultExpiryInterval,
		FlushInterval:    DefaultFlushInterval,
		MaxReaders:       DefaultMaxReaders,
		MaxWorkers:       DefaultMaxWorkers,
		MaxQueueSize:     DefaultMaxQueueSize,
		PercentThreshold: DefaultPercentThreshold,
		Viper:            viper.New(),
	}
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	socket := new(fakesocket.CountingFakeRandomPacketConn)
	err := s.RunWithCustomSocket(ctx, func() (net.PacketConn, error) { return socket, nil })
	if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		t.Errorf("statsd run failed: %v", err)
	}
	runtime.ReadMemStats(&memStatsFinish)
	totalAlloc := memStatsFinish.TotalAlloc - memStatsStart.TotalAlloc
	numMetrics := socket.NumReads
	mallocs := memStatsFinish.Mallocs - memStatsStart.Mallocs
	t.Logf("Processed metrics: %d\nTotalAlloc: %d (%d per metric)\nMallocs: %d (%d per metric)\nNumGC: %d\nGCCPUFraction: %f",
		numMetrics,
		totalAlloc, totalAlloc/numMetrics,
		mallocs, mallocs/numMetrics,
		memStatsFinish.NumGC-memStatsStart.NumGC,
		memStatsFinish.GCCPUFraction)
}

type fakeProvider struct {
	instance *cloudTypes.Instance
}

func (fp *fakeProvider) ProviderName() string {
	return "fakeProvider"
}

func (fp *fakeProvider) SampleConfig() string {
	return ""
}

func (fp *fakeProvider) Instance(IP types.IP) (*cloudTypes.Instance, error) {
	return fp.instance, nil
}
