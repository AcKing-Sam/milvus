package grpcqueryservice

import (
	"context"
	"log"
	"net"
	"strconv"
	"sync"

	"google.golang.org/grpc"

	"github.com/zilliztech/milvus-distributed/internal/msgstream"
	"github.com/zilliztech/milvus-distributed/internal/proto/commonpb"
	"github.com/zilliztech/milvus-distributed/internal/proto/internalpb2"
	"github.com/zilliztech/milvus-distributed/internal/proto/milvuspb"
	"github.com/zilliztech/milvus-distributed/internal/proto/querypb"
	qs "github.com/zilliztech/milvus-distributed/internal/queryservice"
)

type Server struct {
	grpcServer *grpc.Server
	grpcError  error
	grpcErrMux sync.Mutex

	loopCtx    context.Context
	loopCancel context.CancelFunc

	queryService *qs.QueryService

	msFactory msgstream.Factory
}

func NewServer(ctx context.Context, factory msgstream.Factory) (*Server, error) {
	ctx1, cancel := context.WithCancel(ctx)
	service, err := qs.NewQueryService(ctx1, factory)
	if err != nil {
		cancel()
		return nil, err
	}

	return &Server{
		queryService: service,
		loopCtx:      ctx1,
		loopCancel:   cancel,
		msFactory:    factory,
	}, nil
}

func (s *Server) Init() error {
	log.Println("query service init")
	if err := s.queryService.Init(); err != nil {
		return err
	}
	return nil
}

func (s *Server) Start() error {
	log.Println("start query service ...")

	s.grpcServer = grpc.NewServer()
	querypb.RegisterQueryServiceServer(s.grpcServer, s)
	log.Println("Starting start query service Server")
	lis, err := net.Listen("tcp", ":"+strconv.Itoa(qs.Params.Port))
	if err != nil {
		return err
	}

	go func() {
		if err := s.grpcServer.Serve(lis); err != nil {
			s.grpcErrMux.Lock()
			defer s.grpcErrMux.Unlock()
			s.grpcError = err
		}
	}()

	s.grpcErrMux.Lock()
	err = s.grpcError
	s.grpcErrMux.Unlock()

	if err != nil {
		return err
	}

	s.queryService.Start()
	return nil
}

func (s *Server) Stop() error {
	err := s.queryService.Stop()
	s.loopCancel()
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
	return err
}

func (s *Server) GetComponentStates(ctx context.Context, req *commonpb.Empty) (*internalpb2.ComponentStates, error) {
	componentStates, err := s.queryService.GetComponentStates()
	if err != nil {
		return &internalpb2.ComponentStates{
			Status: &commonpb.Status{
				ErrorCode: commonpb.ErrorCode_UNEXPECTED_ERROR,
				Reason:    err.Error(),
			},
		}, err
	}

	return componentStates, nil
}

func (s *Server) GetTimeTickChannel(ctx context.Context, req *commonpb.Empty) (*milvuspb.StringResponse, error) {
	channel, err := s.queryService.GetTimeTickChannel()
	if err != nil {
		return &milvuspb.StringResponse{
			Status: &commonpb.Status{
				ErrorCode: commonpb.ErrorCode_UNEXPECTED_ERROR,
				Reason:    err.Error(),
			},
		}, err
	}

	return &milvuspb.StringResponse{
		Status: &commonpb.Status{
			ErrorCode: commonpb.ErrorCode_SUCCESS,
			Reason:    "",
		},
		Value: channel,
	}, nil
}

func (s *Server) GetStatisticsChannel(ctx context.Context, req *commonpb.Empty) (*milvuspb.StringResponse, error) {
	statisticsChannel, err := s.queryService.GetStatisticsChannel()
	if err != nil {
		return &milvuspb.StringResponse{
			Status: &commonpb.Status{
				ErrorCode: commonpb.ErrorCode_UNEXPECTED_ERROR,
				Reason:    err.Error(),
			},
		}, err
	}

	return &milvuspb.StringResponse{
		Status: &commonpb.Status{
			ErrorCode: commonpb.ErrorCode_SUCCESS,
			Reason:    "",
		},
		Value: statisticsChannel,
	}, nil
}

func (s *Server) SetMasterService(m qs.MasterServiceInterface) error {
	s.queryService.SetMasterService(m)
	return nil
}

func (s *Server) SetDataService(d qs.DataServiceInterface) error {
	s.queryService.SetDataService(d)
	return nil
}

func (s *Server) RegisterNode(ctx context.Context, req *querypb.RegisterNodeRequest) (*querypb.RegisterNodeResponse, error) {
	return s.queryService.RegisterNode(req)
}

func (s *Server) ShowCollections(ctx context.Context, req *querypb.ShowCollectionRequest) (*querypb.ShowCollectionResponse, error) {
	return s.queryService.ShowCollections(req)
}

func (s *Server) LoadCollection(ctx context.Context, req *querypb.LoadCollectionRequest) (*commonpb.Status, error) {
	return s.queryService.LoadCollection(req)
}

func (s *Server) ReleaseCollection(ctx context.Context, req *querypb.ReleaseCollectionRequest) (*commonpb.Status, error) {
	return s.queryService.ReleaseCollection(req)
}

func (s *Server) ShowPartitions(ctx context.Context, req *querypb.ShowPartitionRequest) (*querypb.ShowPartitionResponse, error) {
	return s.queryService.ShowPartitions(req)
}

func (s *Server) GetPartitionStates(ctx context.Context, req *querypb.PartitionStatesRequest) (*querypb.PartitionStatesResponse, error) {
	return s.queryService.GetPartitionStates(req)
}

func (s *Server) LoadPartitions(ctx context.Context, req *querypb.LoadPartitionRequest) (*commonpb.Status, error) {
	return s.queryService.LoadPartitions(req)
}

func (s *Server) ReleasePartitions(ctx context.Context, req *querypb.ReleasePartitionRequest) (*commonpb.Status, error) {
	return s.queryService.ReleasePartitions(req)
}

func (s *Server) CreateQueryChannel(ctx context.Context, req *commonpb.Empty) (*querypb.CreateQueryChannelResponse, error) {
	return s.queryService.CreateQueryChannel()
}

func (s *Server) GetSegmentInfo(ctx context.Context, req *querypb.SegmentInfoRequest) (*querypb.SegmentInfoResponse, error) {
	return s.queryService.GetSegmentInfo(req)
}
