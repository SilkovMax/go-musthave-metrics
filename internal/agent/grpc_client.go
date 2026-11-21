package agent

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/SilkovMax/go-musthave-metrics/internal/model"
	"github.com/SilkovMax/go-musthave-metrics/internal/proto"
)

// GRPCClient обёртка над gRPC клиентом
type GRPCClient struct {
	client  proto.MetricsClient
	localIP string
}

// IPInjectorInterceptor добавляет IP в метаданные каждого запроса
func IPInjectorInterceptor(ip string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// Добавляем x-real-ip к исходящему контексту
		ctx = metadata.AppendToOutgoingContext(ctx, "x-real-ip", ip)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// NewGRPCClient инициализирует grpc-соединение
func NewGRPCClient(address, localIP string) (*GRPCClient, error) {
	conn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(IPInjectorInterceptor(localIP)),
	)
	if err != nil {
		return nil, err
	}

	return &GRPCClient{
		client:  proto.NewMetricsClient(conn),
		localIP: localIP,
	}, nil
}

// SendBatch отправляет батч метрик через grpc
func (c *GRPCClient) SendBatch(metrics []model.Metrics) error {
	var protoMetrics []*proto.Metric

	// Преобразуем метрики в proto.Metric
	for _, m := range metrics {
		pm := &proto.Metric{
			Id:   m.ID,
			Type: convertMType(m.MType),
		}

		if m.MType == "counter" && m.Delta != nil {
			pm.Delta = *m.Delta
		} else if m.MType == "gauge" && m.Value != nil {
			pm.Value = *m.Value
		}

		protoMetrics = append(protoMetrics, pm)
	}

	req := &proto.UpdateMetricsRequest{
		Metrics: protoMetrics,
	}

	_, err := c.client.UpdateMetrics(context.Background(), req)
	return err
}

// convertMType конвертирует строковый тип в proto enum
func convertMType(mtype string) proto.Metric_MType {
	if mtype == "counter" {
		return proto.Metric_COUNTER
	}
	return proto.Metric_GAUGE
}
