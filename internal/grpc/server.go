package grpc

import (
	"context"
	"net"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/SilkovMax/go-musthave-metrics/internal/model"
	"github.com/SilkovMax/go-musthave-metrics/internal/proto"
	"github.com/SilkovMax/go-musthave-metrics/internal/repository"
)

// Server реализует grpc сервис Metrics
type Server struct {
	proto.UnimplementedMetricsServer
	storage       repository.Storage
	trustedSubnet string
}

// NewServer создает новый grpc сервер
func NewServer(storage repository.Storage, trustedSubnet string) *Server {
	return &Server{
		storage:       storage,
		trustedSubnet: trustedSubnet,
	}
}

// UpdateMetrics принимает батч метрик и сохраняет их
func (s *Server) UpdateMetrics(ctx context.Context, req *proto.UpdateMetricsRequest) (*proto.UpdateMetricsResponse, error) {
	var metricsToSave []model.Metrics

	for _, pm := range req.Metrics {
		metric := model.Metrics{
			ID:    pm.Id,
			MType: strings.ToLower(pm.Type.String()), // Приводим к единому формату
		}

		// Заполняем только то поле, которое соответствует типу метрики
		if pm.Type == proto.Metric_COUNTER {
			metric.Delta = &pm.Delta
		} else {
			metric.Value = &pm.Value
		}

		metricsToSave = append(metricsToSave, metric)
	}
	// Вызываем SetBatch один раз для всего батча
	if err := s.storage.SetBatch(metricsToSave); err != nil {
		return nil, status.Errorf(codes.Internal, "метрики не сохранились: %v", err)
	}

	return &proto.UpdateMetricsResponse{}, nil
}

// TrustedSubnetInterceptor проверяет IP из метаданных
func TrustedSubnetInterceptor(trustedSubnet string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Если подсеть не задана, пропускаем
		if trustedSubnet == "" {
			return handler(ctx, req)
		}

		// Читаем метаданные
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "отсутствуют метаданные")
		}

		// Берем наш заголовок
		ips := md.Get("x-real-ip")
		if len(ips) == 0 {
			return nil, status.Errorf(codes.Unauthenticated, "отсутствует заголовок x-real-ip")
		}

		clientIP := net.ParseIP(ips[0])
		if clientIP == nil {
			return nil, status.Errorf(codes.InvalidArgument, "неверный формат IP")
		}

		_, subnet, err := net.ParseCIDR(trustedSubnet)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "ошибка конфигурации подсети")
		}

		if !subnet.Contains(clientIP) {
			return nil, status.Errorf(codes.PermissionDenied, "IP не в доверенной подсети")
		}

		return handler(ctx, req)
	}
}
