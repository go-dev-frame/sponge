package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"google.golang.org/grpc"

	"github.com/go-dev-frame/sponge/pkg/copier"
	"github.com/go-dev-frame/sponge/pkg/grpc/interceptor"
	"github.com/go-dev-frame/sponge/pkg/logger"
	"github.com/go-dev-frame/sponge/pkg/sgorm/query"

	serverNameExampleV1 "github.com/go-dev-frame/sponge/api/serverNameExample/v1"
	"github.com/go-dev-frame/sponge/internal/cache"
	"github.com/go-dev-frame/sponge/internal/dao"
	"github.com/go-dev-frame/sponge/internal/database"
	"github.com/go-dev-frame/sponge/internal/ecode"
	"github.com/go-dev-frame/sponge/internal/model"
)

func init() {
	registerFns = append(registerFns, func(server *grpc.Server) {
		serverNameExampleV1.RegisterUserExampleServer(server, NewUserExampleServer()) // register service to the rpc service
	})
}

var _ serverNameExampleV1.UserExampleServer = (*userExample)(nil)
var _ time.Time

type userExample struct {
	serverNameExampleV1.UnimplementedUserExampleServer

	iDao dao.UserExampleDao
}

// NewUserExampleServer create a new service
func NewUserExampleServer() serverNameExampleV1.UserExampleServer {
	return &userExample{
		iDao: dao.NewUserExampleDao(
			database.GetDB(), // todo show db driver name here
			cache.NewUserExampleCache(database.GetCacheType()),
		),
	}
}

// Create a new userExample
func (s *userExample) Create(ctx context.Context, req *serverNameExampleV1.CreateUserExampleRequest) (*serverNameExampleV1.CreateUserExampleReply, error) {
	err := req.Validate()
	if err != nil {
		logger.Warn("req.Validate error", logger.Err(err), logger.Any("req", req), interceptor.ServerCtxRequestIDField(ctx))
		return nil, ecode.StatusInvalidParams.Err()
	}
	ctx = interceptor.WrapServerCtx(ctx)

	record := &model.UserExample{}
	err = copier.Copy(record, req)
	if err != nil {
		return nil, ecode.StatusCreateUserExample.Err()
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	err = s.iDao.Create(ctx, record)
	if err != nil {
		logger.Error("Create error", logger.Err(err), logger.Any("userExample", record), interceptor.ServerCtxRequestIDField(ctx))
		return nil, ecode.StatusInternalServerError.ToRPCErr()
	}

	return &serverNameExampleV1.CreateUserExampleReply{Id: record.ID}, nil
}

// DeleteByID delete a userExample by id
func (s *userExample) DeleteByID(ctx context.Context, req *serverNameExampleV1.DeleteUserExampleByIDRequest) (*serverNameExampleV1.DeleteUserExampleByIDReply, error) {
	err := req.Validate()
	if err != nil {
		logger.Warn("req.Validate error", logger.Err(err), logger.Any("req", req), interceptor.ServerCtxRequestIDField(ctx))
		return nil, ecode.StatusInvalidParams.Err()
	}
	ctx = interceptor.WrapServerCtx(ctx)

	err = s.iDao.DeleteByID(ctx, req.Id)
	if err != nil {
		logger.Error("DeleteByID error", logger.Err(err), logger.Any("id", req.Id), interceptor.ServerCtxRequestIDField(ctx))
		return nil, ecode.StatusInternalServerError.ToRPCErr()
	}

	return &serverNameExampleV1.DeleteUserExampleByIDReply{}, nil
}

// UpdateByID update a userExample by id
func (s *userExample) UpdateByID(ctx context.Context, req *serverNameExampleV1.UpdateUserExampleByIDRequest) (*serverNameExampleV1.UpdateUserExampleByIDReply, error) {
	err := req.Validate()
	if err != nil {
		logger.Warn("req.Validate error", logger.Err(err), logger.Any("req", req), interceptor.ServerCtxRequestIDField(ctx))
		return nil, ecode.StatusInvalidParams.Err()
	}
	ctx = interceptor.WrapServerCtx(ctx)

	record := &model.UserExample{}
	err = copier.Copy(record, req)
	if err != nil {
		return nil, ecode.StatusUpdateByIDUserExample.Err()
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here
	record.ID = req.Id

	err = s.iDao.UpdateByID(ctx, record)
	if err != nil {
		logger.Error("UpdateByID error", logger.Err(err), logger.Any("userExample", record), interceptor.ServerCtxRequestIDField(ctx))
		return nil, ecode.StatusInternalServerError.ToRPCErr()
	}

	return &serverNameExampleV1.UpdateUserExampleByIDReply{}, nil
}

// GetByID get a userExample by id
func (s *userExample) GetByID(ctx context.Context, req *serverNameExampleV1.GetUserExampleByIDRequest) (*serverNameExampleV1.GetUserExampleByIDReply, error) {
	err := req.Validate()
	if err != nil {
		logger.Warn("req.Validate error", logger.Err(err), logger.Any("req", req), interceptor.ServerCtxRequestIDField(ctx))
		return nil, ecode.StatusInvalidParams.Err()
	}
	ctx = interceptor.WrapServerCtx(ctx)

	record, err := s.iDao.GetByID(ctx, req.Id)
	if err != nil {
		if errors.Is(err, database.ErrRecordNotFound) {
			logger.Warn("GetByID error", logger.Err(err), logger.Any("id", req.Id), interceptor.ServerCtxRequestIDField(ctx))
			return nil, ecode.StatusNotFound.Err()
		}
		logger.Error("GetByID error", logger.Err(err), logger.Any("id", req.Id), interceptor.ServerCtxRequestIDField(ctx))
		return nil, ecode.StatusInternalServerError.ToRPCErr()
	}

	data, err := convertUserExample(record)
	if err != nil {
		logger.Warn("convertUserExample error", logger.Err(err), logger.Any("userExample", record), interceptor.ServerCtxRequestIDField(ctx))
		return nil, ecode.StatusGetByIDUserExample.Err()
	}

	return &serverNameExampleV1.GetUserExampleByIDReply{UserExample: data}, nil
}

// List get a paginated list of userExamples by custom conditions
func (s *userExample) List(ctx context.Context, req *serverNameExampleV1.ListUserExampleRequest) (*serverNameExampleV1.ListUserExampleReply, error) {
	err := req.Validate()
	if err != nil {
		logger.Warn("req.Validate error", logger.Err(err), logger.Any("req", req), interceptor.ServerCtxRequestIDField(ctx))
		return nil, ecode.StatusInvalidParams.Err()
	}
	ctx = interceptor.WrapServerCtx(ctx)

	params := &query.Params{}
	err = copier.Copy(params, req.Params)
	if err != nil {
		return nil, ecode.StatusListUserExample.Err()
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	records, total, err := s.iDao.GetByColumns(ctx, params)
	if err != nil {
		if strings.Contains(err.Error(), "query params error:") {
			logger.Warn("GetByColumns error", logger.Err(err), logger.Any("params", params), interceptor.ServerCtxRequestIDField(ctx))
			return nil, ecode.StatusInvalidParams.Err()
		}
		logger.Error("GetByColumns error", logger.Err(err), logger.Any("params", params), interceptor.ServerCtxRequestIDField(ctx))
		return nil, ecode.StatusInternalServerError.ToRPCErr()
	}

	userExamples := []*serverNameExampleV1.UserExample{}
	for _, record := range records {
		data, err := convertUserExample(record)
		if err != nil {
			logger.Warn("convertUserExample error", logger.Err(err), logger.Any("id", record.ID), interceptor.ServerCtxRequestIDField(ctx))
			continue
		}
		userExamples = append(userExamples, data)
	}

	return &serverNameExampleV1.ListUserExampleReply{
		Total:        total,
		UserExamples: userExamples,
	}, nil
}

func convertUserExample(record *model.UserExample) (*serverNameExampleV1.UserExample, error) {
	value := &serverNameExampleV1.UserExample{}
	err := copier.Copy(value, record)
	if err != nil {
		return nil, err
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	return value, nil
}
